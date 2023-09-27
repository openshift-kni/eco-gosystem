package cluster

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/clusterversion"

	"k8s.io/client-go/tools/clientcmd"
)

// CheckClustersPresent can be used to check for the presence of specific clusters.
func CheckClustersPresent(clients []*clients.Settings) error {
	// Log the cluster list
	log.Println(clients)

	for _, client := range clients {
		if client == nil {
			return errors.New("provided nil client in cluster list")
		}
	}

	return nil
}

// GetClusterName extracts the cluster name from provided kubeconfig. It assumes the there's exactly 1 cluster.
func GetClusterName(kubeconfigEnvVar string) (string, error) {
	kubeFilePath, present := os.LookupEnv(kubeconfigEnvVar)
	if !present {
		return "", fmt.Errorf("can not load api client. Please check '%s' env var", kubeconfigEnvVar)
	}

	rawConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeFilePath},
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		}).RawConfig()

	for _, cluster := range rawConfig.Clusters {
		// The cluster data looks like this:
		/*
			    "clusters": [
					{
						"name": "local-cluster",
						"cluster": {
							"server": "https://api.local-cluster.karmalabs.local:6443",
							"certificate-authority-data": "DATA+OMITTED"
						}
					}
				],
		*/
		// However the "name" field is not always correct so it is more consistent to instead
		// parse the name out from the server url
		splits := strings.Split(cluster.Server, ".")

		clusterName := splits[1]

		log.Println("cluster name: ", clusterName)

		return clusterName, nil
	}

	return "", fmt.Errorf("can not load api client. Please check '%s' env var", kubeconfigEnvVar)
}

// GetClusterVersion can be used to get the Openshift version from the provided cluster.
func GetClusterVersion(apiClient *clients.Settings) (string, error) {
	// Check if the client was even defined first
	if apiClient == nil {
		return "", fmt.Errorf("provided client was not defined")
	}

	clusterVersion, err := clusterversion.Pull(apiClient)
	if err != nil {
		return "", err
	}

	histories := clusterVersion.Object.Status.History
	for i := len(histories) - 1; i >= 0; i-- {
		history := histories[i]
		if history.State == "Completed" {
			return history.Version, nil
		}
	}

	log.Println("Warning: No completed version found in clusterversion. Returning desired version")

	return clusterVersion.Object.Status.Desired.Version, nil
}
