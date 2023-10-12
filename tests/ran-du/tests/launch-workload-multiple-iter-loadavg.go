package ran_du_system_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-gosystem/tests/internal/await"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"
	"github.com/openshift-kni/eco-gosystem/tests/internal/shell"
	. "github.com/openshift-kni/eco-gosystem/tests/ran-du/internal/randuinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ran-du/internal/randuparams"
	"github.com/openshift-kni/eco-gosystem/tests/ran-du/internal/randutestworkload"
	"gonum.org/v1/gonum/floats"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe(
	"LaunchWorkloadMultipleIterations",
	Ordered,
	ContinueOnFailure,
	Label("LaunchWorkloadMultipleIterations"), func() {
		It("Launch workload multiple times", polarion.ID("45698"), Label("LaunchWorkloadMultipleIterations"), func() {

			launchIterations, err := strconv.Atoi(RanDuTestConfig.LaunchWorkloadIterations)
			if err != nil {
				fmt.Println(err)
			}

			By("Launch workload")
			for iter := 0; iter < launchIterations; iter++ {
				fmt.Printf("Launch workload iteration no. %d\n", iter)

				By("Clean up workload namespace")
				if namespace.NewBuilder(APIClient, RanDuTestConfig.TestWorkload.Namespace).Exists() {
					err := randutestworkload.CleanNameSpace(randuparams.DefaultTimeout, RanDuTestConfig.TestWorkload.Namespace)
					Expect(err).ToNot(HaveOccurred(), "Failed to clean workload test namespace objects")
				}

				if RanDuTestConfig.TestWorkload.CreateMethod == randuparams.TestWorkloadShellLaunchMethod {
					By("Launching workload using shell method")
					_, err := shell.ExecuteCmd(RanDuTestConfig.TestWorkload.CreateShellCmd)
					Expect(err).ToNot(HaveOccurred(), "Failed to launch workload")
				}

				By("Waiting for deployment replicas to become ready")
				_, err := await.WaitUntilAllDeploymentsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace,
					randuparams.DefaultTimeout)
				Expect(err).ToNot(HaveOccurred(), "error while waiting for deployment to become ready")

				By("Waiting for statefulset replicas to become ready")
				_, err = await.WaitUntilAllStatefulSetsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace,
					randuparams.DefaultTimeout)
				Expect(err).ToNot(HaveOccurred(), "error while waiting for statefulsets to become ready")

				By("Waiting for all pods to become ready")
				_, err = await.WaitUntilAllPodsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace, 10*time.Second)
				Expect(err).ToNot(HaveOccurred(), "pod not ready: %s", err)
			}
			By("Retrieve nodes list")
			nodeList, err := nodes.List(
				APIClient,
				metav1.ListOptions{},
			)
			Expect(err).ToNot(HaveOccurred(), "Error listing nodes.")

			By("Observe node load average while workload is running")
			cmdToExec := []string{"awk", "{print $1}", "/proc/loadavg"}
			var observedLoadAverage []float64

			for n := 0; n < 30; n++ {
				buf, err := cmd.ExecCmd(cmdToExec, nodeList[0].Definition.Name)
				Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

				floatBuf, err := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(buf, "\r", ""), "\n", ""), 32)
				if err != nil {
					fmt.Println(err)
				}

				observedLoadAverage = append(observedLoadAverage, floatBuf)
				time.Sleep(10 * time.Second)
			}

			Expect(floats.Max(observedLoadAverage)).To(BeNumerically("<", randuparams.TestMultipleLaunchWorkloadLoadAvg),
				"error: node load average detected above 100")

		})
		AfterAll(func() {
			By("Cleaning up test workload resources")
			err := randutestworkload.CleanNameSpace(randuparams.DefaultTimeout, RanDuTestConfig.TestWorkload.Namespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to clean workload test namespace objects")
		})
	})
