package cleaner

import (
	"net/url"
	"vsphere-cleaner/parser"
	"vsphere-cleaner/vsphere"
)

type Cleaner struct {
	lockPath             string
	yamlParser           parser.Parser
	vSphereClientBuilder func(*url.URL) (vsphere.Client, error)
}

func NewCleaner(lockPath string, yamlParser parser.Parser, vSphereClientBuilder func(*url.URL) (vsphere.Client, error)) Cleaner {
	return Cleaner{lockPath: lockPath, yamlParser: yamlParser, vSphereClientBuilder: vSphereClientBuilder}
}

func (c Cleaner) Clean() error {
	config, err := c.yamlParser.Parse(c.lockPath)
	if err != nil {
		return err
	}
	vSphereClient, err := c.vSphereClientBuilder(config.BuildUrl())
	if err != nil {
		return err
	}
	err = vSphereClient.DeleteVM(config.DirectorIP())
	if err != nil {
		return err
	}

	ips, err := config.UsedIPs()
	if err != nil {
		return err
	}
	for _, ip := range ips {
		err = vSphereClient.DeleteVM(ip)
		if err != nil {
			return err
		}
	}
	return err
}
