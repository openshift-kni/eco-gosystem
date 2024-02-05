package rebootparams

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
		"openshift-machine-api": "openshift-machine-api",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}
)
