package powermanagement

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell

	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/powermanagement/internal/powermanagementhelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/powermanagement/internal/powermanagementparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"

	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	"github.com/openshift/cluster-node-tuning-operator/pkg/performanceprofile/controller/performanceprofile/components"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/utils/ptr"
)

var _ = Describe("Per-Core Runtime Tuning of power states - CRI-O", Ordered, func() {
	var (
		nodeList                     []*nodes.Builder
		snoNode                      *corev1.Node
		err                          error
		isSNO                        bool
		perfProfile                  *nto.Builder
		workloadHints                *performancev2.WorkloadHints
		originPerformanceProfileSpec performancev2.PerformanceProfileSpec
	)

	var timeout = 15 * time.Minute

	BeforeAll(func() {
		// Get nodes for connection host
		nodeList, err = nodes.List(gitopsztpinittools.HubAPIClient)
		Expect(err).ToNot(HaveOccurred())

		isSNO = powermanagementhelper.IsSingleNodeCluster(gitopsztpinittools.HubAPIClient)
		Expect(isSNO).Should(BeTrue(), "Currently only SNO nodes are supported by this test")

		perfProfile, err = powermanagementhelper.GetPerformanceProfileWithCPUSet()
		Expect(err).ToNot(HaveOccurred())
		snoNode = nodeList[0].Object

		originPerformanceProfileSpec = perfProfile.Object.Spec
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func() {
		// Restore performance profile to original spec after each test
		glog.V(100).Infof("Restore performance profile to original specs")
		perfProfile.Definition.Spec = originPerformanceProfileSpec

		_, err := perfProfile.Update(true)
		Expect(err).ToNot(HaveOccurred())
		mcp, err := mco.Pull(gitopsztpinittools.HubAPIClient, "master")
		Expect(err).ToNot(HaveOccurred())

		err = mcp.WaitForUpdate(timeout)
		Expect(err).ToNot(HaveOccurred())
	})

	// OCP-54571 - Install SNO node with standard DU profile that does not include WorkloadHints
	It("verifies expected kernel parameters with no workload hints specified in PerformanceProfile", func() {

		// Verify no workload hints in performanceprofile
		workloadHints = perfProfile.Definition.Spec.WorkloadHints
		// By(fmt.Sprintf("DEBUG WorkloadHints = %v\n\n%+v", workloadHints, workloadHints))
		if workloadHints != nil {
			Skip("WorkloadHints already present in perfProfile.Spec")
		}

		By("Checking for expected kernel parameters")

		// Expected default set of kernel parameters when no WorkloadHints are specified in PerformanceProfile
		requiredKernelParms := []string{
			"nohz_full=[0-9,-]+",
			"tsc=nowatchdog",
			"nosoftlockup",
			"nmi_watchdog=0",
			"mce=off",
			"skew_tick=1",
			"intel_pstate=disable",
		}
		output, err := cmd.ExecCmd([]string{"chroot", "rootfs", "cat", "/proc/cmdline"}, snoNode.Name)
		Expect(err).ToNot(HaveOccurred(), "Unable to cat /proc/cmdline")
		for _, parameter := range requiredKernelParms {
			By(fmt.Sprintf("Checking /proc/cmdline for %s", parameter))
			rePattern := regexp.MustCompile(parameter)
			Expect(rePattern.FindStringIndex(output)).
				ToNot(BeNil(), fmt.Sprintf("Kernel parameter %s is missing from cmdline", parameter))

		}

	})

	// OCP-54572 - Enable powersave at node level and then enable performance at node level
	It("Enable powersave at node level and then enable performance at node level", func() {
		By("Patching the performance profile with the workload hints")
		err := setPowerMode(perfProfile, true, false, true)
		Expect(err).ToNot(HaveOccurred(), "Unable to set power mode")

		cmdline, err := cmd.ExecCmd([]string{"chroot", "rootfs", "cat", "/proc/cmdline"}, snoNode.Name)
		Expect(err).ToNot(HaveOccurred(), "Unable to cat /proc/cmdline")
		Expect(cmdline).To(ContainSubstring("intel_pstate=passive"))
		Expect(cmdline).ToNot(ContainSubstring("intel_pstate=disable"))
	})

	// OCP-54574 - Telco_Case: Enable powersave at node level and then enable high performance
	// at node level, check power consumption with no workload pods.
	It("Enable powersave, and then enable high performance at node level, "+
		"check power consumption with no workload pods.", func() {

		testPodAnnotations := map[string]string{
			"cpu-load-balancing.crio.io": "disable",
			"cpu-quota.crio.io":          "disable",
			"irq-load-balancing.crio.io": "disable",
			"cpu-c-states.crio.io":       "disable",
			"cpu-freq-governor.crio.io":  "performance",
		}

		cpuLimit := resource.MustParse("2")
		memLimit := resource.MustParse("100Mi")

		By("Patching the performance profile with the workload hints")
		err := setPowerMode(perfProfile, true, false, true)
		Expect(err).ToNot(HaveOccurred(), "Unable to set power mode")

		By("Define test pod")
		ns := namespace.NewBuilder(gitopsztpinittools.HubAPIClient, powermanagementparams.PrivPodNamespace)
		testpod := powermanagementhelper.DefineQoSTestPod(*ns, snoNode.Name, cpuLimit.String(),
			cpuLimit.String(), memLimit.String(), memLimit.String())
		testpod.Definition.Annotations = testPodAnnotations
		runtimeClass := getComponentName(perfProfile.Definition.Name, components.ComponentNamePrefix)
		testpod.Definition.Spec.RuntimeClassName = &runtimeClass

		By("Create test pod")
		testpod, err = testpod.CreateAndWaitUntilRunning(10 * time.Minute)
		Expect(err).ToNot(HaveOccurred(), "Unable to create pod")
		Expect(testpod.Object.Status.QOSClass).To(Equal(corev1.PodQOSGuaranteed),
			"Test pod does not have QoS class of Guaranteed")

		defer func() {
			// delete the pod only if the tc had failed and the pod still exists
			By("Delete pod in case of a failure")
			if testpod.Exists() {
				err = testpod.WaitUntilDeleted(5 * time.Minute)
				Expect(err).ToNot(HaveOccurred())
			}
		}()

		output, err := testpod.ExecCommand([]string{"cat", "/sys/fs/cgroup/cpuset/cpuset.cpus"})
		Expect(err).ToNot(HaveOccurred())

		By("Verify powersetting of cpus used by the pod")
		trimmedOutput := strings.Trim(output.String(), "\r\n")
		cpusUsed, err := cpuset.Parse(trimmedOutput)
		Expect(err).ToNot(HaveOccurred())

		targetCpus := cpusUsed.List()
		err = checkCPUGovernorsAndResumeLatency(targetCpus, snoNode.Name, "n/a", "performance")
		Expect(err).ToNot(HaveOccurred())

		By("Verify the rest of cpus have default power setting")
		allCpus := snoNode.Status.Capacity.Cpu()
		cpus, err := cpuset.Parse(fmt.Sprintf("0-%d", allCpus.Value()-1))
		Expect(err).ToNot(HaveOccurred())

		otherCPUs := cpus.Difference(cpusUsed)
		// Verify cpus not assigned to the pod have default power settings.
		err = checkCPUGovernorsAndResumeLatency(otherCPUs.List(), snoNode.Name, "0", "performance")
		Expect(err).ToNot(HaveOccurred())

		By("Delete the pod")
		err = testpod.WaitUntilDeleted(5 * time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By("Verify after pod was deleted cpus assigned to container have default powersave settings")
		err = checkCPUGovernorsAndResumeLatency(targetCpus, snoNode.Name, "0", "performance")
		Expect(err).ToNot(HaveOccurred())

	})

})

// setPowerMode updates the performance profile with the given workload hints, and waits for the mcp update.
func setPowerMode(perfProfile *nto.Builder, perPodPowerManagement, highPowerConsumption, realTime bool) error {
	glog.V(100).Infof("Set powersave mode on performance profile")

	perfProfile.Definition.Spec.WorkloadHints = &performancev2.WorkloadHints{
		PerPodPowerManagement: ptr.To[bool](perPodPowerManagement),
		HighPowerConsumption:  ptr.To[bool](highPowerConsumption),
		RealTime:              ptr.To[bool](realTime),
	}

	_, err := perfProfile.Update(true)
	if err != nil {
		return err
	}

	mcp, err := mco.Pull(gitopsztpinittools.HubAPIClient, "master")
	Expect(err).ToNot(HaveOccurred())

	err = mcp.WaitForUpdate(powermanagementparams.Timeout)

	return err
}

// getComponentName returns the component name for the specific performance profile.
func getComponentName(profileName string, prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, profileName)
}

// checkCPUGovernorsAndResumeLatency  Checks power and latency settings of the cpus.
func checkCPUGovernorsAndResumeLatency(cpus []int, nodeName, pmQos, governor string) error {
	for _, cpu := range cpus {
		command := []string{"/bin/bash", "-c", fmt.Sprintf(
			"cat /sys/devices/system/cpu/cpu%d/power/pm_qos_resume_latency_us", cpu)}

		output, err := cmd.ExecCmd(command, nodeName)
		if err != nil {
			return err
		}

		Expect(strings.Trim(output, "\r")).To(Equal(pmQos))

		command = []string{"/bin/bash", "-c", fmt.Sprintf(
			"cat /sys/devices/system/cpu/cpu%d/cpufreq/scaling_governor", cpu)}

		output, err = cmd.ExecCmd(command, nodeName)
		if err != nil {
			return err
		}

		Expect(strings.Trim(output, "\r")).To(Equal(governor))
	}

	return nil
}
