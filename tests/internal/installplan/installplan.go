package installplan

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides a struct for installplan object from the cluster and a installplan definition.
type Builder struct {
	// installplan definition, used to create the installplan object.
	Definition *v1alpha1.InstallPlan
	// Created installplan object.
	Object *v1alpha1.InstallPlan
	// Used in functions that define or mutate installplan definition. errorMsg is processed
	// before the installplan object is created
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof("Initializing new %s installplan structure", name)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1alpha1.InstallPlan{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the installplan is empty")

		builder.errorMsg = "installplan 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The nsname of the installplan is empty")

		builder.errorMsg = "installplan 'nsname' cannot be empty"
	}

	return &builder
}

// GetInstallPlanBuilder returns installplan (first from the list).
func GetInstallPlanBuilder(apiClient *clients.Settings, nsname string) (*Builder, error) {
	if apiClient == nil {
		return nil, fmt.Errorf("apiClient is nil")
	}

	installPlanList, err := apiClient.InstallPlans(nsname).List(context.Background(), metaV1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("Failed to list all installplan in %s namespace due to %s",
			nsname, err.Error())

		return nil, err
	}

	var installPlanObjects []*Builder

	for _, foundCsv := range installPlanList.Items {
		copiedCsv := foundCsv
		csvBuilder := &Builder{
			apiClient:  apiClient,
			Object:     &copiedCsv,
			Definition: &copiedCsv,
		}

		installPlanObjects = append(installPlanObjects, csvBuilder)
	}

	if len(installPlanObjects) == 0 {
		return nil, fmt.Errorf("installplan not found in %s namespace", nsname)
	}

	return installPlanObjects[0], nil
}
