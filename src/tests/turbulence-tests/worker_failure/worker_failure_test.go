package workload_test

import (
	"tests/config"
	. "tests/test_helpers"

	"github.com/cloudfoundry/bosh-cli/director"
	"github.com/cppforlife/turbulence/incident"
	"github.com/cppforlife/turbulence/incident/selector"
	"github.com/cppforlife/turbulence/tasks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Worker failure scenarios", func() {

	var (
		deployment          director.Deployment
		countRunningWorkers func() int
		kubectl             *KubectlRunner
		nginxDaemonSetSpec  = PathFromRoot("specs/nginx-daemonset.yml")
		testconfig          *config.Config
	)

	BeforeSuite(func() {
		var err error
		testconfig, err = config.InitConfig()
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		var err error
		director := NewDirector(testconfig.Bosh)
		deployment, err = director.FindDeployment(testconfig.Bosh.Deployment)
		Expect(err).NotTo(HaveOccurred())
		countRunningWorkers = CountDeploymentVmsOfType(deployment, WorkerVmType, VmRunningState)

		kubectl = NewKubectlRunner(testconfig.Kubernetes.PathToKubeConfig)
		kubectl.CreateNamespace()

		Expect(countRunningWorkers()).To(Equal(3))
		Expect(AllBoshWorkersHaveJoinedK8s(deployment, kubectl)).To(BeTrue())
	})

	AfterEach(func() {
		kubectl.RunKubectlCommand("delete", "-f", nginxDaemonSetSpec)
		kubectl.RunKubectlCommand("delete", "namespace", kubectl.Namespace())
	})

	Specify("K8s applications are scheduled on the resurrected node", func() {
		By("Deleting the Worker VM")
		hellRaiser := TurbulenceClient(testconfig.Turbulence)
		killOneWorker := incident.Request{
			Selector: selector.Request{
				Deployment: &selector.NameRequest{
					Name: testconfig.Bosh.Deployment,
				},
				Group: &selector.NameRequest{
					Name: WorkerVmType,
				},
				ID: &selector.IDRequest{
					Limit: selector.MustNewLimitFromString("1"),
				},
			},
			Tasks: tasks.OptionsSlice{
				tasks.KillOptions{},
			},
		}
		incident := hellRaiser.CreateIncident(killOneWorker)
		incident.Wait()
		Eventually(countRunningWorkers, 600, 20).Should(Equal(2))

		By("Verifying the worker VM has restarted")
		var startingWorkerVms []director.VMInfo
		getStartingWorkerVms := func() []director.VMInfo {
			startingWorkerVms = DeploymentVmsOfType(deployment, WorkerVmType, VmStartingState)
			return startingWorkerVms
		}
		Eventually(getStartingWorkerVms, 600, 20).Should(HaveLen(1))

		By("Verifying that the Worker VM has joined the K8s cluster")
		Eventually(func() bool { return AllBoshWorkersHaveJoinedK8s(deployment, kubectl) }, 600, 20).Should(BeTrue())

		By("Deploying nginx on 3 nodes")
		Eventually(kubectl.RunKubectlCommand("create", "-f", nginxDaemonSetSpec), "30s", "5s").Should(gexec.Exit(0))
		Eventually(kubectl.RunKubectlCommand("rollout", "status", "daemonset/nginx", "-w"), "120s").Should(gexec.Exit(0))

		By("Verifying nginx got deployed on new node")
		nodeNames := GetNodeNamesForRunningPods(kubectl)
		Expect(nodeNames).To(HaveLen(3))

		By("Ensuring a new worker VM has joined the bosh deployment")
		var runningWorkerVms []director.VMInfo
		getRunningWorkerVms := func() []director.VMInfo {
			runningWorkerVms = DeploymentVmsOfType(deployment, WorkerVmType, VmRunningState)
			return runningWorkerVms
		}
		Eventually(getRunningWorkerVms).Should(HaveLen(3))

		_, err := GetNewVmId(runningWorkerVms, nodeNames)
		Expect(err).ToNot(HaveOccurred())
	})

})
