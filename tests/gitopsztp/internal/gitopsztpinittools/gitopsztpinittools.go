package gitopsztpinittools

import (
	"os"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
)

var (
	// HubAPIClient is the api client to the hub.
	HubAPIClient *clients.Settings
	// SpokeAPIClient is the api client to the spoke.
	SpokeAPIClient *clients.Settings
)

func init() {
	HubAPIClient = inittools.APIClient
	SpokeAPIClient = DefineAPIClient(gitopsztpparams.SpokeKubeEnvKey)
}

// DefineAPIClient creates new api client instance connected to given cluster.
func DefineAPIClient(kubeconfigEnvVar string) *clients.Settings {
	kubeFilePath, present := os.LookupEnv(kubeconfigEnvVar)
	if !present {
		glog.Fatalf("can not load api client. Please check %s env var", kubeconfigEnvVar)

		return nil
	}

	clients := clients.New(kubeFilePath)
	if clients == nil {
		glog.Fatalf("client is not set please check %s env variable", kubeconfigEnvVar)

		return nil
	}

	return clients
}
