package imagebasedupgradeparams

const (
	// SeedHubKubeEnvKey is the hub's kubeconfig env var.
	SeedHubKubeEnvKey string = "KUBECONFIG_SEED_HUB"
	// TargetHubKubeEnvKey is the hub's kubeconfig env var.
	TargetHubKubeEnvKey string = "KUBECONFIG_TARGET_HUB"
	// SeedSNOKubeEnvKey is the seed kubeconfig env var.
	SeedSNOKubeEnvKey string = "KUBECONFIG_SEED_SNO"
	// TargetSNOKubeEnvKey is the target kubeconfig env var.
	TargetSNOKubeEnvKey string = "KUBECONFIG_TARGET_SNO"
	// ImagebasedupgradeCrName is the Imagebasedupgrade CR name.
	ImagebasedupgradeCrName string = "upgrade"
	// ImagebasedupgradeCrNamespace is the Imagebasedupgrade CR namespace.
	ImagebasedupgradeCrNamespace string = "openshift-lifecycle-agent"
)
