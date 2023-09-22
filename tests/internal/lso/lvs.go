package lso

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	lsoV1alpha1 "github.com/openshift/local-storage-operator/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// LVSBuilder provides a struct for localVolumeSet object from the cluster and a localVolumeSet definition.
type LVSBuilder struct {
	// localVolumeSet definition, used to create the localVolumeSet object.
	Definition *lsoV1alpha1.LocalVolumeSet
	// Created localVolumeDiscovery object.
	Object *lsoV1alpha1.LocalVolumeSet
	// Used in functions that define or mutate localVolumeSet definition. errorMsg is processed
	// before the localVolumeSet object is created
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewLVSBuilder creates new instance of LVSBuilder.
func NewLVSBuilder(apiClient *clients.Settings, name, nsname string) *LVSBuilder {
	glog.V(100).Infof("Initializing new %s localVolumeSet structure in %s namespace", name, nsname)

	builder := LVSBuilder{
		apiClient: apiClient,
		Definition: &lsoV1alpha1.LocalVolumeSet{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the localVolumeSet is empty")

		builder.errorMsg = "localVolumeSet 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The nsname of the localVolumeSet is empty")

		builder.errorMsg = "localVolumeSet 'nsname' cannot be empty"
	}

	return &builder
}

// Discover fetches existing localVolumeSet from cluster.
func (builder *LVSBuilder) Discover() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Pulling existing localVolumeSet with name %s under namespace %s from cluster",
		builder.Definition.Name, builder.Definition.Namespace)

	lvs := &lsoV1alpha1.LocalVolumeSet{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, lvs)

	builder.Object = lvs

	return err
}

// Exists checks whether the given localVolumeSet exists.
func (builder *LVSBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if localVolumeSet %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.Discover()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *LVSBuilder) validate() (bool, error) {
	resourceCRD := "LocalVolumeSet"

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
