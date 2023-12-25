package powermanagementparams

import "time"

// RAN Power Measurement metric names/prefixes.
const (
	RanPowerMetricTotalSamples            = "ranmetrics_power_total_samples"
	RanPowerMetricSamplingIntervalSeconds = "ranmetrics_power_sampling_interval_seconds"
	RanPowerMetricMinInstantPower         = "ranmetrics_power_min_instantaneous"
	RanPowerMetricMaxInstantPower         = "ranmetrics_power_max_instantaneous"
	RanPowerMetricMeanInstantPower        = "ranmetrics_power_mean_instantaneous"
	RanPowerMetricStdDevInstantPower      = "ranmetrics_power_standard_deviation_instantaneous"
	RanPowerMetricMedianInstantPower      = "ranmetrics_power_median_instantaneous"
)

// Power State Configurations.
const (
	PowerSavingMode     = "powersaving"
	PerformanceMode     = "performance"
	HighPerformanceMode = "highperformance"
)

// Ipmitool power metrics.
const (
	IpmiDcmiPowerMinimumDuringSampling = "minPower"
	IpmiDcmiPowerMaximumDuringSampling = "maxPower"
	IpmiDcmiPowerAverageDuringSampling = "avgPower"
	IpmiDcmiPowerInstantaneous         = "instantaneousPower"
)

// Default sampling values.
const (
	DefaultRanMetricSamplingInterval = "30s"
	DefaultRanNoWorkloadDuration     = "5m"
	DefaultRanSteadyWorkloadDuration = "10m"
)

const (
	// NamespaceTesting is the tests namespace.
	NamespaceTesting = "ran-test"
	// PromNamespace is the Prometheus namespace.
	PromNamespace = "openshift-monitoring"
	// ProcessExporterPodName is the name of the process-exporter pod.
	ProcessExporterPodName = "process-exporter"
	// ProcessExporterImage is the image of the process-exporter pod.
	ProcessExporterImage = "quay.io/ocp-edge-qe/process-exporter:ppid-2"
	// Timeout is the timeout being used in powersave tests.
	Timeout = 15 * time.Minute
	// CnfTestImage is the test image used by pods.
	CnfTestImage = "quay.io/openshift-kni/cnf-tests:4.8"
	// PrivPodNamespace is the priv pod namespace.
	PrivPodNamespace = "cnfgotestpriv"
)
