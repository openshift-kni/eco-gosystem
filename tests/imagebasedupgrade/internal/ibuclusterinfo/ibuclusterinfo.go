package ibuclusterinfo

import (
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	. "github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeinittools"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
)

// itemInArray is a dedicated func to check if an item is in an array.
func itemInArray(item string, array []string) bool {
	for _, v := range array {
		if v == item {
			return true
		}
	}

	return false
}

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

	csvList, err := olm.ListClusterServiceVersionInAllNamespaces(TargetSNOAPIClient)

	if err != nil {
		glog.V(100).Infof("Could not retrieve csv list")

		return err
	}

	var installedCSV []string

	for _, csv := range csvList {
		if itemInArray(csv.Object.Name, installedCSV) {
			// Do nothing
		} else {
			installedCSV = append(installedCSV, csv.Object.Name)
		}
	}

	node, err := nodes.List(TargetSNOAPIClient)

	if err != nil {
		glog.V(100).Infof("Could not retrieve node list")

		return err
	}

	*upgradeVar = imagebasedupgradeparams.ClusterStruct{
		Version:   clusterVersion,
		ID:        clusterID,
		Operators: installedCSV,
		NodeName:  node[0].Object.Name,
	}

	return nil
}
