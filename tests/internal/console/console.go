package console

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	v1 "github.com/openshift/api/config/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder provides a struct for console object from the cluster and a console definition.
type Builder struct {
	// console definition, used to create the pod object.
	Definition *v1.Console
	// Created console object.
	Object *v1.Console
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// AdditionalOptions additional options for console object.
type AdditionalOptions func(builder *Builder) (*Builder, error)

// NewBuilder creates a new instance of Builder.
func NewBuilder(apiClient *clients.Settings) *Builder {
	glog.V(100).Info("Initializing new console structure")

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Console{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "cluster",
			},
		},
	}

	return &builder
}

// Pull loads an existing console into the Builder struct.
func Pull(apiClient *clients.Settings) (*Builder, error) {
	glog.V(100).Infof("Pulling cluster console")

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Console{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "cluster",
			},
			Spec: v1.ConsoleSpec{
				Authentication: v1.ConsoleAuthentication{},
			},
		},
	}

	if valid, err := builder.validate(); !valid {
		glog.V(100).Info("Bad cluster state, failed to pull cluster console. Object doesn't exist")

		return nil, fmt.Errorf("cluster console object doesn't exist; %w", err)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Update renovates the existing cluster console object with cluster console definition in builder.
func (builder *Builder) Update() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Info("Updating cluster console")

	var err error
	builder.Object, err = builder.apiClient.Consoles().Update(context.Background(), builder.Definition,
		metaV1.UpdateOptions{})

	return builder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Console"

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
