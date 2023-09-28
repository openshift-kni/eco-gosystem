package talmparams

import (
	"time"
)

// talm related vars.
const (
	TalmUpdatedConditionsVersion = "4.12"
	OpenshiftOperatorNamespace   = "openshift-operators"
	OperatorHubTalmNamespace     = "topology-aware-lifecycle-manager"
	Spoke1KubeEnvKey             = "KUBECONFIG"
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
