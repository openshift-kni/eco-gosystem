package clusterlogging

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	clov1 "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterLoggingBuilder provides struct for clusterLogging object.
type ClusterLoggingBuilder struct {
	// ClusterLogging definition. Used to create clusterLogging object with minimum set of required elements.
	Definition *clov1.ClusterLogging
	// Created clusterLogging object on the cluster.
	Object *clov1.ClusterLogging
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before clusterLogging object is created.
	errorMsg string
}

// NewClusterLoggingBuilder method creates new instance of builder.
func NewClusterLoggingBuilder(
	apiClient *clients.Settings, name, nsname string) *ClusterLoggingBuilder {
	glog.V(100).Infof("Initializing new clusterLogging structure with the following params: name: %s, namespace: %s",
		name, nsname)

	builder := &ClusterLoggingBuilder{
		apiClient: apiClient,
		Definition: &clov1.ClusterLogging{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterLogging is empty")

		builder.errorMsg = "The clusterLogging 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterLogging is empty")

		builder.errorMsg = "The clusterLogging 'namespace' cannot be empty"
	}

	return builder
}

// Discover fetches existing clusterLogging from cluster.
func (builder *ClusterLoggingBuilder) Discover() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Pulling existing clusterLogging with name %s under namespace %s from cluster",
		builder.Definition.Name, builder.Definition.Namespace)

	clo := &clov1.ClusterLogging{}
	err := builder.apiClient.Get(context.Background(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, clo)

	builder.Object = clo

	return err
}

// Create makes a clusterLogging in the cluster and stores the created object in struct.
func (builder *ClusterLoggingBuilder) Create() (*ClusterLoggingBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the clusterLogging %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Delete removes clusterLogging from a cluster.
func (builder *ClusterLoggingBuilder) Delete() (*ClusterLoggingBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the clusterLogging %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("clusterLogging cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.Background(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("can not delete clusterLogging: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Exists checks whether the given clusterLogging exists.
func (builder *ClusterLoggingBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterLogging %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.Discover()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ClusterLoggingBuilder) validate() (bool, error) {
	resourceCRD := "ClusterLogging"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
