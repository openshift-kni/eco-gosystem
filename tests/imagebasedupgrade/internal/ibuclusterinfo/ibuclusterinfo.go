package ibuclusterinfo

import (
	"github.com/golang/glog"
	. "github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeinittools"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
)

// SaveClusterInfo is a dedicated func to save cluster info.
func SaveClusterInfo(upgradeVar *imagebasedupgradeparams.ClusterStruct) error {
	clusterVersion, err := cluster.GetClusterVersion(TargetSNOAPIClient)

	if err != nil {
		glog.V(100).Infof("Could not retrieve cluster version")

		return err
	}

	clusterID, err := cluster.GetClusterID(TargetSNOAPIClient)

	if err != nil {
		glog.V(100).Infof("Could not retrieve cluster id")

		return err
	}

	*upgradeVar = imagebasedupgradeparams.ClusterStruct{
		Version: clusterVersion,
		ID:      clusterID,
	}

	return nil
}
