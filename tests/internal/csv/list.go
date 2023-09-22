package csv

import (
	"context"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListAllInNamespaceWithNamePattern returns a cluster-wide csv inventory.
func ListAllInNamespaceWithNamePattern(apiClient *clients.Settings, namePattern, nsname string) ([]*Builder, error) {
	glog.V(100).Infof("Listing all csv with the name pattern %s found in %s namespace", namePattern, nsname)

	csvList, err := apiClient.ClusterServiceVersions(nsname).List(context.Background(), v1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("Failed to list all clusterserviceversions in %s namespace due to %s",
			nsname, err.Error())

		return nil, err
	}

	var csvObjects []*Builder

	for _, foundCsv := range csvList.Items {
		if strings.Contains(foundCsv.Name, namePattern) {
			copiedCsv := foundCsv
			csvBuilder := &Builder{
				apiClient:  apiClient,
				Object:     &copiedCsv,
				Definition: &copiedCsv,
			}

			csvObjects = append(csvObjects, csvBuilder)
		}
	}

	return csvObjects, nil
}
