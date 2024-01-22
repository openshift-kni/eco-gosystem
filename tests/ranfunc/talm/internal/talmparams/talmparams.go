package talmparams

import (
	"time"

	"github.com/openshift-kni/k8sreporter"
	v1 "k8s.io/api/core/v1"
)

var (
	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		TalmTestNamespace: "",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}
)

// talm related vars.
const (
	HubKubeEnvKey                = "KUBECONFIG"
	TalmUpdatedConditionsVersion = "4.12"
	OpenshiftOperatorNamespace   = "openshift-operators"
	OperatorHubTalmNamespace     = "topology-aware-lifecycle-manager"
	Spoke1KubeEnvKey             = "KUBECONFIG_SPOKE1"
	Spoke2KubeEnvKey             = "KUBECONFIG_SPOKE2"
	TalmContainerName            = "manager"
	TalmDefaultReconcileTime     = 5 * time.Minute
	TalmOperatorNamespace        = "openshift-cluster-group-upgrades"
	TalmPodLabelSelector         = "pod-template-hash"
	TalmPodNameHub               = "cluster-group-upgrades-controller-manager"
	TalmSystemStablizationTime   = 15 * time.Second
	TalmTestNamespace            = "talm-test"
	TalmTestPollInterval         = 5 * time.Second
	// common prefix.
	PolicyNameCommonName       = "generated-policy"
	PolicySetNameCommonName    = "generated-policyset"
	PlacementRuleCommonName    = "generated-placementrule"
	PlacementBindingCommonName = "generated-placementbinding"
	CguCommonName              = "generated-cgu"
	NsCommonName               = "generated-namespace"
)

var (
	// TalmNamespaces contains a list of all the talm related namespaces.
	TalmNamespaces = map[string]string{
		TalmOperatorNamespace: "talm",
	}
)
