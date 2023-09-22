package csv

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides a struct for csv object from the cluster and a csv definition.
type Builder struct {
	// csv definition, used to create the csv object.
	Definition *v1alpha1.ClusterServiceVersion
	// Created csv object.
	Object *v1alpha1.ClusterServiceVersion
	// Used in functions that define or mutate csv definition. errorMsg is processed
	// before the csv object is created
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewBuilder creates new instance of Builder.
func NewBuilder(apiClient *clients.Settings, name, nsname string) *Builder {
	glog.V(100).Infof("Initializing new %s clusterVersion structure in %s namespace", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1alpha1.ClusterServiceVersion{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the csv is empty")

		builder.errorMsg = "csv 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The nsname of the csv is empty")

		builder.errorMsg = "csv 'nsname' cannot be empty"
	}

	return &builder
}

// Exists checks whether the given csv exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if csv %s exists in %s namespace",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.ClusterServiceVersions(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// IsSucceeded check if the csv is Succeeded.
func (builder *Builder) IsSucceeded() (bool, error) {
	if valid, err := builder.validate(); !valid {
		return false, err
	}

	glog.V(100).Infof("Verify %s csv in %s namespace is Succeeded",
		builder.Definition.Name, builder.Definition.Namespace)

	csvFound := builder.Exists()
	if !csvFound {
		return false, fmt.Errorf("%s csv not found in %s namespace",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	phase, err := builder.GetPhase()

	if err != nil {
		return false, fmt.Errorf("failed to get phase value for %s clusterserviceversion in %s namespace due to %w",
			builder.Definition.Name, builder.Definition.Namespace, err)
	}

	if phase == "Succeeded" {
		return true, nil
	}

	return false, fmt.Errorf("bad %s csv in %s namespace phase: %s",
		builder.Definition.Name, builder.Definition.Namespace, phase)
}

// GetPhase get current csv phase.
func (builder *Builder) GetPhase() (v1alpha1.ClusterServiceVersionPhase, error) {
	if valid, err := builder.validate(); !valid {
		return "", err
	}

	glog.V(100).Infof("Get %s csv in %s namespace phase",
		builder.Definition.Name, builder.Definition.Namespace)

	cvFound := builder.Exists()
	if !cvFound {
		return "", fmt.Errorf("%s csv not found in %s namespace",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	phase := builder.Object.Status.Phase

	return phase, nil
}

// GetCSVBuilder returns a csvBuilder of a csv based on provided label (first from the list).
func GetCSVBuilder(apiClient *clients.Settings, namePattern, nsname string) (*Builder, error) {
	if apiClient == nil {
		return nil, fmt.Errorf("apiClient is nil")
	}

	csvList, err := ListAllInNamespaceWithNamePattern(apiClient, namePattern, nsname)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s csv in %s namespace on cluster: %w", namePattern, nsname, err)
	}

	if len(csvList) == 0 {
		return nil, fmt.Errorf("csv with suitable for the %s name-pattern not currently running in %s namespace",
			namePattern, nsname)
	}

	return csvList[0], nil
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "ClusterServiceVersion"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		return false, fmt.Errorf(msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	return true, nil
}
