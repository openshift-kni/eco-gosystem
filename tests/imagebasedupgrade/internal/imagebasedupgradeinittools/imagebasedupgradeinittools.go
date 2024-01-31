package imagebasedupgradeinittools

import (
	"os"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
)

var (
	// SeedHubAPIClient is the api client to the hub cluster.
	SeedHubAPIClient *clients.Settings
	// TargetHubAPIClient is the api client to the hub cluster.
	TargetHubAPIClient *clients.Settings
	// SeedSNOAPIClient is the api client to the seed SNO cluster.
	SeedSNOAPIClient *clients.Settings
	// TargetSNOAPIClient is the api client to the target SNO cluster.
	TargetSNOAPIClient *clients.Settings
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	SeedHubAPIClient = inittools.APIClient
	TargetHubAPIClient = inittools.APIClient
	SeedSNOAPIClient = DefineAPIClient(imagebasedupgradeparams.SeedSNOKubeEnvKey)
	TargetSNOAPIClient = DefineAPIClient(imagebasedupgradeparams.TargetSNOKubeEnvKey)
}

// DefineAPIClient creates new api client instance connected to given cluster.
func DefineAPIClient(kubeconfigEnvVar string) *clients.Settings {
	kubeFilePath, present := os.LookupEnv(kubeconfigEnvVar)
	if !present {
		glog.Fatalf("can not load api client. Please check %s env var", kubeconfigEnvVar)

		return nil
	}

	client := clients.New(kubeFilePath)
	if client == nil {
		glog.Fatalf("client is not set please check %s env variable", kubeconfigEnvVar)

		return nil
	}

	return client
}
