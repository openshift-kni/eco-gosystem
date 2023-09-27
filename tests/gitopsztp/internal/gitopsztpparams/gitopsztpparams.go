package gitopsztpparams

import "time"

// ArgocdGitDetails stores argocd git details.
type ArgocdGitDetails struct {
	Repo   string
	Branch string
	Path   string
}

// ZTP and Argocd vars.
const (
	// Namespaces matching '^ztp*' are special
	// Argocd will only let us use such namespaces for policy gen templates on the SNO nodes.
	ZtpTestNamespace          string        = "ztp-test"
	HubKubeEnvKey             string        = "KUBECONFIG_HUB"
	SpokeKubeEnvKey           string        = "KUBECONFIG"
	OpenshiftGitops           string        = "openshift-gitops"
	OpenshiftGitopsRepoServer string        = "openshift-gitops-repo-server"
	ArgocdPoliciesAppName     string        = "policies"
	ArgocdClustersAppName     string        = "clusters"
	ArgocdChangeTimeout       time.Duration = 10 * time.Minute
	ArgocdChangeInterval      time.Duration = 10 * time.Second
	DefaultTimeout            time.Duration = 5 * time.Minute
	ZtpSiteGenerateImageName  string        = "registry-proxy.engineering.redhat.com/rh-osbs/openshift4-ztp-site-generate"
	AcmOperatorName           string        = "advanced-cluster-management"
	AcmOperatorNamespace      string        = "rhacm"
	AcmArgocdInitContainer    string        = "multicluster-operators-subscription"
	AcmPolicyGeneratorName    string        = "acm-policy-generator"
	ImageRegistryNamespace    string        = "openshift-image-registry"
	MulticlusterhubOperator   string        = "multiclusterhub-operator"
	PerfProfileName           string        = "openshift-node-performance-profile"
	MachineConfigName         string        = "02-master-workload-partitioning"
	TunedPatchName            string        = "performance-patch"
	TunedNamespace            string        = "openshift-cluster-node-tuning-operator"
	MCPname                   string        = "master"
	PrivPodNamespace          string        = "cnfgotestpriv"
)

// ArgocdApps is a list of the argocd app names that are defined above.
var ArgocdApps = []string{
	ArgocdClustersAppName,
	ArgocdPoliciesAppName,
}
