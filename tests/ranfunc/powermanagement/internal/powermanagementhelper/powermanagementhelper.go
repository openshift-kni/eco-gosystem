package powermanagementhelper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"
	"github.com/openshift-kni/eco-gosystem/tests/internal/config"
	"github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/powermanagement/internal/powermanagementparams"
	performancev2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
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
	daemonSet, err := daemonset.Pull(ranfuncinittools.HubAPIClient, powermanagementparams.PromNamespace,
		powermanagementparams.ProcessExporterPodName)
	Expect(err).ShouldNot(HaveOccurred())

	Eventually(func() error {
		daemonSet, err = daemonset.Pull(ranfuncinittools.HubAPIClient, powermanagementparams.PromNamespace,
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
	profilesBuilder, err := nto.ListProfiles(ranfuncinittools.HubAPIClient)
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

	pod := pod.NewBuilder(ranfuncinittools.HubAPIClient, "", namespace.Definition.Name, image)
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

	mcp, err := mco.Pull(ranfuncinittools.HubAPIClient, "master")
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

// CollectPowerMetricsWithNoWorkload collects metrics with no workload.
func CollectPowerMetricsWithNoWorkload(duration, samplingInterval time.Duration,
	tag, nodeName string) (map[string]string, error) {
	scenario := "noworkload"
	glog.V(100).Infof("Wait for %s for %s scenario\n", duration.String(), scenario)

	return CollectPowerUsageMetrics(duration, samplingInterval, scenario, tag, nodeName)
}

// CollectPowerUsageMetrics collects power usage metrics.
func CollectPowerUsageMetrics(duration, samplingInterval time.Duration, scenario,
	tag, nodeName string) (map[string]string, error) {
	startTime := time.Now()
	expectedEndTime := startTime.Add(duration)
	powerMeasurements := make(map[time.Time]map[string]float64)

	stopSampling := func(t time.Time) bool {
		if t.After(startTime) && t.Before(expectedEndTime) {
			return false
		}

		return true
	}

	var sampleGroup sync.WaitGroup

	err := wait.PollUntilContextTimeout(
		context.TODO(), samplingInterval, duration, true, func(context.Context) (bool, error) {
			sampleGroup.Add(1)
			timestamp := time.Now()
			go func(t time.Time) {
				defer sampleGroup.Done()
				out, err := GetHostPowerUsage(nodeName)
				if err == nil {
					powerMeasurements[t] = out
				}
			}(timestamp)

			return stopSampling(timestamp.Add(samplingInterval)), nil
		})

	// Wait for all the tasks to complete.
	sampleGroup.Wait()

	if err != nil {
		return nil, err
	}

	log.Printf("Power usage test started: %v\nPower usage test ended: %v\n", startTime, time.Now())

	if len(powerMeasurements) < 1 {
		return nil, errors.New("no power usage metrics were retrieved")
	}

	// Compute power metrics.
	return computePowerUsageStatistics(powerMeasurements, samplingInterval, scenario, tag)
}

// GetHostPowerUsage retrieve host power utilization metrics queried via ipmitool command.
func GetHostPowerUsage(nodeName string) (map[string]float64, error) {
	user, password, hosts := ParseBmcInfo(inittools.GeneralConfig)
	if len(hosts) > 1 {
		log.Printf("multiple hosts detected, only using %s\n", hosts[0])
	}

	subcommands := []string{"dcmi", "power", "reading"}
	args := []string{"ipmitool", "-I", "lanplus", "-U", user, "-P", password, "-H", hosts[0]}
	args = append(args, subcommands...)

	output, err := cmd.ExecCmd(args, nodeName)
	if err != nil {
		return nil, err
	}

	// Parse the ipmitool string output and return the result.
	return parseIpmiPowerOutput(output)
}

// ParseBmcInfo returns bmc username, password, and hosts from environment variables if exist.
func ParseBmcInfo(conf *config.GeneralConfig) (bmcUser, bmcPassword string, bmcHosts []string) {
	hostsEnvVar := conf.BmcHosts
	Expect(hostsEnvVar).ToNot(BeEmpty(), "Please set BMC_HOSTS environment variable.")
	hosts := strings.Split(hostsEnvVar, ",")

	return conf.BmcUser, conf.BmcPassword, hosts
}

// computePowerUsageStatistics computes the power usage summary statistics.
func computePowerUsageStatistics(powerMeasurements map[time.Time]map[string]float64,
	samplingInterval time.Duration, scenario, tag string) (map[string]string, error) {
	/*
		Compute power measurement statistics

		Sample power measurement data:
		map[
		2023-03-08 10:17:46.629599 -0500 EST m=+132.341222733:map[avgPower:251 instantaneousPower:326 maxPower:503 minPower:8]
		2023-03-08 10:18:46.630737 -0500 EST m=+192.341245075:map[avgPower:251 instantaneousPower:324 maxPower:503 minPower:8]
		2023-03-08 10:19:46.563857 -0500 EST m=+252.341201729:map[avgPower:251 instantaneousPower:329 maxPower:503 minPower:8]
		2023-03-08 10:20:46.563313 -0500 EST m=+312.340308977:map[avgPower:251 instantaneousPower:332 maxPower:503 minPower:8]
		2023-03-08 10:21:46.564469 -0500 EST m=+372.341065314:map[avgPower:251 instantaneousPower:329 maxPower:503 minPower:8]
		]

		The following power measurement summary statistics are computed:

		numberSamples: count(powerMeasurements)
		samplingInterval: <samplingInterval>
		minInstantaneousPower: min(instantaneousPower)
		maxInstantaneousPower: max(instantaneousPower)
		meanInstantaneousPower: mean(instantaneousPower)
		stdDevInstantaneousPower: standard-deviation(instantaneousPower)
		medianInstantaneousPower: median(instantaneousPower)
	*/
	glog.V(100).Infof("Power usage measurements for %s:\n%v\n", scenario, powerMeasurements)

	compMap := make(map[string]string)

	numberSamples := len(powerMeasurements)

	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricTotalSamples, scenario, tag)] =
		fmt.Sprintf("%d", numberSamples)
	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricSamplingIntervalSeconds, scenario, tag)] =
		fmt.Sprintf("%.0f", samplingInterval.Seconds())

	instantPowerData := make([]float64, numberSamples)

	index := -1
	for _, row := range powerMeasurements {
		index++

		instantPowerData[index] = row[powermanagementparams.IpmiDcmiPowerInstantaneous]
	}

	minInstantaneousPower, err := Min(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricMinInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", minInstantaneousPower)

	maxInstantaneousPower, err := Max(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricMaxInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", maxInstantaneousPower)

	meanInstantaneousPower, err := Mean(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricMeanInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", meanInstantaneousPower)

	stdDevInstantaneousPower, err := StdDev(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricStdDevInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", stdDevInstantaneousPower)

	medianInstantaneousPower, err := Median(instantPowerData)
	if err != nil {
		return compMap, err
	}

	compMap[fmt.Sprintf("%s_%s_%s", powermanagementparams.RanPowerMetricMedianInstantPower, scenario, tag)] =
		fmt.Sprintf("%.7f", medianInstantaneousPower)

	return compMap, nil
}

// parseIpmiPowerOutput parses the ipmitool host power usage and returns a map of corresponding float values.
func parseIpmiPowerOutput(result string) (map[string]float64, error) {
	powerMeasurements := make(map[string]float64)
	powerMeasurementExpr := make(map[string]string)

	powerMeasurementExpr[powermanagementparams.IpmiDcmiPowerInstantaneous] =
		`Instantaneous power reading: (\s*[0-9]+) Watts`
	powerMeasurementExpr[powermanagementparams.IpmiDcmiPowerMinimumDuringSampling] =
		`Minimum during sampling period: (\s*[0-9]+) Watts`
	powerMeasurementExpr[powermanagementparams.IpmiDcmiPowerMaximumDuringSampling] =
		`Maximum during sampling period: (\s*[0-9]+) Watts`
	powerMeasurementExpr[powermanagementparams.IpmiDcmiPowerAverageDuringSampling] =
		`Average power reading over sample period: (\s*[0-9]+) Watts`

	// Extract power measurements.
	for key, pattern := range powerMeasurementExpr {
		re := regexp.MustCompile(pattern)
		res := re.FindStringSubmatch(result)

		if len(res) > 0 {
			var value float64
			_, err := fmt.Sscan(res[1], &value)

			if err != nil {
				return nil, err
			}

			powerMeasurements[key] = value
		}
	}

	return powerMeasurements, nil
}
