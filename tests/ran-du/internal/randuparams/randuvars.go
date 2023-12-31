package randuparams

import (
	systemtestsparams "github.com/openshift-kni/eco-gosystem/tests/internal/params"
	"github.com/openshift-kni/k8sreporter"
	v1 "k8s.io/api/core/v1"
)

var (
	// Labels represents the range of labels that can be used for test cases selection.
	Labels = []string{systemtestsparams.Label, Label}

	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		"test": "randu-test-workload",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}

	// TestNamespaceName is used for defining the namespace name where test resources are created.
	TestNamespaceName = "ran-du-system-tests"

	// TestMultipleLaunchWorkloadLoadAvg is used for defining the node load average threshold to be
	// used in the LaunchWorkloadMultipleIterations test.
	TestMultipleLaunchWorkloadLoadAvg = 100
)
