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

// LVDBuilder provides a struct for localVolumeDiscovery object from the cluster and a localVolumeDiscovery definition.
type LVDBuilder struct {
	// localVolumeDiscovery definition, used to create the localVolumeDiscovery object.
	Definition *lsoV1alpha1.LocalVolumeDiscovery
	// Created localVolumeDiscovery object.
	Object *lsoV1alpha1.LocalVolumeDiscovery
	// Used in functions that define or mutate localVolumeDiscovery definition. errorMsg is processed
	// before the localVolumeDiscovery object is created
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// NewLVDBuilder creates new instance of LVDBuilder.
func NewLVDBuilder(apiClient *clients.Settings, name, nsname string) *LVDBuilder {
	glog.V(100).Infof("Initializing new %s localVolumeDiscovery structure in %s namespace", name, nsname)

	builder := LVDBuilder{
		apiClient: apiClient,
		Definition: &lsoV1alpha1.LocalVolumeDiscovery{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the localVolumeDiscovery is empty")

		builder.errorMsg = "localVolumeDiscovery 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The nsname of the localVolumeDiscovery is empty")

		builder.errorMsg = "localVolumeDiscovery 'nsname' cannot be empty"
	}

	return &builder
}

// Discover fetches existing localVolumeDiscovery from cluster.
func (builder *LVDBuilder) Discover() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Pulling existing localVolumeDiscovery with name %s under namespace %s from cluster",
		builder.Definition.Name, builder.Definition.Namespace)

	lvd := &lsoV1alpha1.LocalVolumeDiscovery{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, lvd)

	builder.Object = lvd

	return err
}

// Exists checks whether the given localVolumeDiscovery exists.
func (builder *LVDBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if localVolumeDiscovery %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.Discover()

	return err == nil || !k8serrors.IsNotFound(err)
}

// IsDiscovering check if the localVolumeDiscovery is Discovering.
func (builder *LVDBuilder) IsDiscovering() (bool, error) {
	if valid, err := builder.validate(); !valid {
		return false, err
	}

	glog.V(100).Infof("Verify %s localVolumeDiscovery in %s namespace is Succeeded",
		builder.Definition.Name, builder.Definition.Namespace)

	csvFound := builder.Exists()
	if !csvFound {
		return false, fmt.Errorf("%s localVolumeDiscovery not found in %s namespace",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	phase, err := builder.GetPhase()

	if err != nil {
		return false, fmt.Errorf("failed to get phase value for %s localVolumeDiscovery in %s namespace due to %w",
			builder.Definition.Name, builder.Definition.Namespace, err)
	}

	if phase == "Discovering" {
		return true, nil
	}

	return false, fmt.Errorf("bad %s localVolumeDiscovery in %s namespace phase: %s",
		builder.Definition.Name, builder.Definition.Namespace, phase)
}

// GetPhase get current localVolumeDiscovery phase.
func (builder *LVDBuilder) GetPhase() (lsoV1alpha1.DiscoveryPhase, error) {
	if valid, err := builder.validate(); !valid {
		return "", err
	}

	glog.V(100).Infof("Get %s localVolumeDiscovery in %s namespace phase",
		builder.Definition.Name, builder.Definition.Namespace)

	lvdFound := builder.Exists()
	if !lvdFound {
		return "", fmt.Errorf("%s localVolumeDiscovery not found in %s namespace",
			builder.Definition.Name, builder.Definition.Namespace)
	}

	phase := builder.Object.Status.Phase

	return phase, nil
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *LVDBuilder) validate() (bool, error) {
	resourceCRD := "LocalVolumeDiscovery"

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
