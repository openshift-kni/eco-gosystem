package powermanagementhelper

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/powermanagement/internal/powermanagementparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	. "github.com/onsi/gomega"
)

// Config type keeps general configuration.
type Config struct {
	General struct {
		ReportDirAbsPath              string `yaml:"report" envconfig:"REPORT_DIR_NAME"`
		CnfNodeLabel                  string `yaml:"cnf_worker_label" envconfig:"ROLE_WORKER_CNF"`
		DumpFailedTestsReportLocation string `envconfig:"REPORTER_ERROR_OUTPUT"`
		PolarionReport                bool   `yaml:"polarion_report" envconfig:"POLARION_REPORT"`
	} `yaml:"general"`
	Network struct {
		TestContainerImage      string `yaml:"test_container_image" envconfig:"NETWORK_TEST_CONTAINER_IMAGE"`
		SriovInterfaces         string `envconfig:"CNF_INTERFACES_LIST"`
		MetalLBAddressPoolIP    string `envconfig:"METALLB_ADDR_LIST"`
		MetalLBSwitchInterfaces string `envconfig:"METALLB_SWITCH_INTERFACES"`
		MetalLBVlanIDs          string `envconfig:"METALLB_VLANS"`
		FrrImage                string `yaml:"frr_image" envconfig:"FRR_IMAGE"`
		SwitchUser              string `envconfig:"SWITCH_USER"`
		SwitchPass              string `envconfig:"SWITCH_PASS"`
		SwitchIP                string `envconfig:"SWITCH_IP"`
		SwitchInterfaces        string `envconfig:"SWITCH_INTERFACES"`
	} `yaml:"network"`
	Ran struct {
		CnfTestImage              string   `yaml:"cnf_test_image" envconfig:"CNF_TEST_IMAGE"`
		StressngTestImage         string   `yaml:"stressng_test_image" envconfig:"STRESSNG_TEST_IMAGE"`
		OslatTestImage            string   `yaml:"oslat_test_image" envconfig:"OSLAT_TEST_IMAGE"`
		ProcessExporterImage      string   `yaml:"process_exporter_image" envconfig:"PROCESS_EXPORTER_IMAGE"`
		ProcessExporterConfigsDir string   `yaml:"process_exporter_resources"`
		BmcHosts                  string   `envconfig:"BMC_HOSTS"`
		BmcUser                   string   `yaml:"bmc_user" envconfig:"BMC_USER"`
		BmcPassword               string   `yaml:"bmc_password" envconfig:"BMC_PASSWORD"`
		PduAddr                   string   `envconfig:"PDU_ADDR"`
		PduSocket                 string   `envconfig:"PDU_SOCKET"`
		RanEventTestDebug         string   `envconfig:"RAN_EVENT_TEST_DEBUG"`
		ConsumerImage             string   `yaml:"consumer_image" envconfig:"CLOUD_EVENT_CONSUMER_IMAGE"`
		BmerTestDebug             string   `envconfig:"BMER_TEST_DEBUG"`
		BmerConfigsDir            string   `yaml:"bmer_consumer_manifests"`
		PtpConfigsDir             string   `yaml:"ptp_consumer_manifests"`
		KubeconfigHub             string   `envconfig:"KUBECONFIG_HUB"`
		OcpUpgradeUpstreamURL     string   `yaml:"ocp_upgrade_upstream_url" envconfig:"OCP_UPGRADE_UPSTREAM_URL"`
		TalmPrecachePolicies      []string `envconfig:"TALM_PRECACHE_POLICIES" yaml:"talm_precache_policies"`
		KubeconfigSpoke2          string   `envconfig:"KUBECONFIG_SPOKE2"`
		ZtpGitRepo                string   `envconfig:"ZTP_GIT_REPO"`
		ZtpGitBranch              string   `envconfig:"ZTP_GIT_BRANCH"`
		ZtpGitDir                 string   `envconfig:"ZTP_GIT_DIR"`
	} `yaml:"ran"`
}

// DeployProcessExporter deploys process exporter and returns the daemonset and error if any.
func DeployProcessExporter() *daemonset.Builder {
	daemonSet, err := daemonset.Pull(gitopsztpinittools.HubAPIClient, powermanagementparams.PromNamespace,
		powermanagementparams.ProcessExporterPodName)
	Expect(err).ShouldNot(HaveOccurred())

	Eventually(func() error {
		daemonSet, err = daemonset.Pull(gitopsztpinittools.HubAPIClient, powermanagementparams.PromNamespace,
			powermanagementparams.ProcessExporterPodName)

		if err != nil {
			glog.V(100).Infof("Failed to retrieve status for daemonset: ", daemonSet.Object.Name)

			return err
		}
		if daemonSet.Object.Spec.Template.Spec.Containers[0].Image != powermanagementparams.ProcessExporterImage {
			// Update dummy helper.Config.Ran.ProcessExporterImage to configured value
			daemonSet.Object.Spec.Template.Spec.Containers[0].Image = powermanagementparams.ProcessExporterImage
			daemonSet, err = daemonSet.Update()

			return fmt.Errorf("image updated, retrieve updated daemonset in next round")
		}
		if daemonSet.Object.Status.NumberReady == daemonSet.Object.Status.DesiredNumberScheduled {
			glog.V(100).Infof("All pods are ready in daemonset: %s\n", daemonSet.Object.Name)

			return nil
		}

		return fmt.Errorf("not all daemon pods are ready in daemonset: %s", daemonSet.Object.Name)
	}, 5*time.Minute, 5*time.Second).ShouldNot(HaveOccurred())

	return daemonSet
}

// GetPerformanceProfileWithCPUSet returns the first performance profile found with reserved and isolated cpuset.
func GetPerformanceProfileWithCPUSet() (*nto.Builder, error) {
	profilesBuilder, err := nto.ListProfiles(gitopsztpinittools.HubAPIClient)
	if err != nil {
		return nil, err
	}

	for _, profileBuilder := range profilesBuilder {
		if profileBuilder.Object.Spec.CPU != nil &&
			profileBuilder.Object.Spec.CPU.Reserved != nil &&
			profileBuilder.Object.Spec.CPU.Isolated != nil {
			return profileBuilder, nil
		}
	}

	return nil, fmt.Errorf("performance profile with Reserved and Isolated CPU set is not found")
}

// IsSingleNodeCluster returns true if the cluster is single-node, otherwise false.
func IsSingleNodeCluster(client *clients.Settings) bool {
	nodeList, err := nodes.List(client)
	Expect(err).ToNot(HaveOccurred())

	return len(nodeList) == 1
}

// DefineQoSTestPod defines test pod with given cpu and memory resources.
func DefineQoSTestPod(namespace namespace.Builder, nodeName, cpuReq, cpuLimit, memReq, memLimit string) *pod.Builder {
	image := powermanagementparams.CnfTestImage
	// Create namespace if not already created
	if !namespace.Exists() {
		glog.V(100).Infof("Creating namespace:", namespace)
		_, err := namespace.Create()
		Expect(err).ShouldNot(HaveOccurred())
	}

	pod := pod.NewBuilder(gitopsztpinittools.HubAPIClient, "", namespace.Definition.Name, image)
	pod = RedefineContainerResources(pod, cpuReq, cpuLimit, memReq, memLimit).DefineOnNode(nodeName)

	return pod
}

// RedefineContainerResources redefines a pod builder with CPU and Memory resources in first container
// Use empty string to skip a resource. e.g., cpuLimit="".
func RedefineContainerResources(
	pod *pod.Builder, cpuRequest string, cpuLimit string, memoryRequest string, memoryLimit string) *pod.Builder {
	pod.Definition.Spec.Containers[0].Resources.Requests = corev1.ResourceList{}
	pod.Definition.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}

	if cpuLimit != "" {
		pod.Definition.Spec.Containers[0].Resources.Limits["cpu"] = resource.MustParse(cpuLimit)
	}

	if cpuRequest != "" {
		pod.Definition.Spec.Containers[0].Resources.Requests["cpu"] = resource.MustParse(cpuRequest)
	}

	if memoryLimit != "" {
		pod.Definition.Spec.Containers[0].Resources.Limits["memory"] = resource.MustParse(memoryLimit)
	}

	if memoryRequest != "" {
		pod.Definition.Spec.Containers[0].Resources.Requests["memory"] = resource.MustParse(memoryRequest)
	}

	return pod
}

// SetPowerMode updates the performance profile with the given workload hints, and waits for the mcp update.
func SetPowerMode(perfProfile *nto.Builder, perPodPowerManagement, highPowerConsumption, realTime bool) error {
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

// GetComponentName returns the component name for the specific performance profile.
func GetComponentName(profileName string, prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, profileName)
}

// CheckCPUGovernorsAndResumeLatency  Checks power and latency settings of the cpus.
func CheckCPUGovernorsAndResumeLatency(cpus []int, nodeName, pmQos, governor string) error {
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

// IsIpmitoolExist returns true if ipmitool is installed on test executor, otherwise false.
func IsIpmitoolExist(nodeName string) bool {
	_, err := cmd.ExecCmd([]string{"which", "ipmitool"}, nodeName)

	return err == nil
}

// GetEnv retrieves the value of the environment variable named by the key.
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

// GetPowerState determines the power state from the workloadHints object of the PerformanceProfile.
func GetPowerState(perfProfile *nto.Builder) (powerState string, err error) {
	workloadHints := perfProfile.Definition.Spec.WorkloadHints

	if workloadHints == nil {
		// No workloadHints object -> default is Performance mode
		powerState = powermanagementparams.PerformanceMode
	} else {
		realTime := *workloadHints.RealTime
		highPowerConsumption := *workloadHints.HighPowerConsumption
		perPodPowerManagement := *workloadHints.PerPodPowerManagement

		switch {
		case realTime && !highPowerConsumption && !perPodPowerManagement:
			powerState = powermanagementparams.PerformanceMode
		case realTime && highPowerConsumption && !perPodPowerManagement:
			powerState = powermanagementparams.HighPerformanceMode
		case realTime && !highPowerConsumption && perPodPowerManagement:
			powerState = powermanagementparams.PowerSavingMode
		default:
			return "", errors.New("unknown workloadHints power state configuration")
		}

	}

	return powerState, nil
}
