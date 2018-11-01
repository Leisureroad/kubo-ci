package generic_test

import (
	"os"
	"strconv"
	"tests/test_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const defaultHPATimeout = "210s"

var (
	hpaDeployment = test_helpers.PathFromRoot("specs/hpa-php-apache.yml")
)

var _ = Describe("Horizontal Pod Autoscaling", func() {
	BeforeEach(func() {
		createHPADeployment()
	})

	AfterEach(func() {
		runner.RunKubectlCommand("delete", "-f", hpaDeployment).Wait("60s")
	})

	It("scales the pods accordingly", func() {
		HPATimeout := os.Getenv("HPA_TIMEOUT")
		if HPATimeout == "" {
			HPATimeout = defaultHPATimeout
		}

		By("creating more pods when the CPU load increases")

		increaseCPULoad()
		Eventually(func() int {
			session := runner.RunKubectlCommand("get", "hpa/php-apache", "-o", "jsonpath={.status.currentReplicas}")
			Eventually(session, "10s").Should(gexec.Exit(0))
			replicas, _ := strconv.Atoi(string(session.Out.Contents()))
			return replicas
		}, HPATimeout).Should(BeNumerically(">", 1))

		By("decreasing the number of pods when the CPU load decreases")

		session := runner.RunKubectlCommand("delete", "pod/load-generator", "--now")
		Eventually(session, "30s").Should(gexec.Exit(0))

		Eventually(func() int {
			session := runner.RunKubectlCommand("get", "hpa/php-apache", "-o", "jsonpath={.status.currentReplicas}")
			Eventually(session, "10s").Should(gexec.Exit(0))
			replicas, _ := strconv.Atoi(string(session.Out.Contents()))
			return replicas
		}, HPATimeout).Should(BeNumerically("==", 1))
	})
})

func createHPADeployment() {
	session := runner.RunKubectlCommand("apply", "-f", hpaDeployment)
	Eventually(session, "10s").Should(gexec.Exit(0))

	Eventually(func() string {
		return runner.GetPodStatusBySelector(runner.Namespace(), "app=php-apache")
	}, "120s").Should(Equal("Running"))
}

func increaseCPULoad() {
	remoteCommand := "while true; do wget -q -O- http://php-apache; done"

	session := runner.RunKubectlCommand("run", "-i", "--tty", "load-generator", "--generator=run-pod/v1", "--image=busybox", "--", "/bin/sh", "-c", remoteCommand)
	Eventually(session, "10s").Should(gexec.Exit(0))

	Eventually(func() string {
		return runner.GetPodStatus(runner.Namespace(), "load-generator")
	}, "120s").Should(Equal("Running"))
}
