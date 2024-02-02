package imagebasedupgradeparams

// ClusterStruct is a struct that holds the cluster version and id.
type ClusterStruct struct {
	Version string
	ID      string
}

var (
	// PreUpgradeClusterInfo holds the cluster info pre upgrade.
	PreUpgradeClusterInfo = ClusterStruct{}

	// PostUpgradeClusterInfo holds the cluster info post upgrade.
	PostUpgradeClusterInfo = ClusterStruct{}
)
