package gitopsztphelper

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// HubAPIClient is the api client to the hub.
	HubAPIClient *clients.Settings
	// HubName hub cluster name.
	HubName string
	// SpokeAPIClient is the api client to the spoke.
	SpokeAPIClient *clients.Settings
	// SpokeName spoke cluster name.
	SpokeName string
	// ZtpVersion ztp version.
	ZtpVersion string
	// AcmVersion Advanced cluster management version.
	AcmVersion string
	// TalmVersion talm version.
	TalmVersion string
)

// InitializeClients initializes hub & spoke clients.
func InitializeClients() error {
	if os.Getenv(gitopsztpparams.HubKubeEnvKey) != "" {
		var err error
		// Define all the hub information
		HubAPIClient, err = DefineAPIClient(gitopsztpparams.HubKubeEnvKey)
		if err != nil {
			return err
		}

		HubName, err = cluster.GetClusterName(gitopsztpparams.HubKubeEnvKey)
		if err != nil {
			return err
		}

		ocpVersion, err := cluster.GetClusterVersion(HubAPIClient)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has OCP version '%s'\n", HubName, ocpVersion)

		AcmVersion, err = GetOperatorVersionFromCSV(
			HubAPIClient,
			gitopsztpparams.AcmOperatorName,
			gitopsztpparams.AcmOperatorNamespace,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has ACM version '%s'\n", HubName, AcmVersion)

		ZtpVersion, err = GetZtpVersionFromArgocd(
			HubAPIClient,
			gitopsztpparams.OpenshiftGitopsRepoServer,
			gitopsztpparams.OpenshiftGitops,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has ZTP version '%s'\n", HubName, ZtpVersion)

		TalmVersion, err = GetOperatorVersionFromCSV(
			HubAPIClient,
			gitopsztpparams.OperatorHubTalmNamespace,
			gitopsztpparams.OpenshiftOperatorNamespace,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has TALM version '%s'\n", HubName, TalmVersion)
	}

	// Spoke is the default kubeconfig
	if os.Getenv(gitopsztpparams.SpokeKubeEnvKey) != "" {
		var err error
		// Define all the spoke information
		SpokeAPIClient, err = DefineAPIClient(gitopsztpparams.SpokeKubeEnvKey)
		if err != nil {
			return err
		}

		SpokeName, err = cluster.GetClusterName(gitopsztpparams.SpokeKubeEnvKey)
		if err != nil {
			return err
		}

		ocpVersion, err := cluster.GetClusterVersion(SpokeAPIClient)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has OCP version '%s'\n", SpokeName, ocpVersion)
	}

	return nil
}

// DefineAPIClient creates new api client instance connected to given cluster.
func DefineAPIClient(kubeconfigEnvVar string) (*clients.Settings, error) {
	kubeFilePath, present := os.LookupEnv(kubeconfigEnvVar)
	if !present {
		return nil, fmt.Errorf("can not load api client. Please check %s env var", kubeconfigEnvVar)
	}

	clients := clients.New(kubeFilePath)
	if clients == nil {
		return nil, fmt.Errorf("client is not set please check %s env variable", kubeconfigEnvVar)
	}

	return clients, nil
}

// GetOperatorVersionFromCSV returns operator version from csv.
func GetOperatorVersionFromCSV(client *clients.Settings, operatorName, operatorNamespace string) (string, error) {
	// Check if the client is valid
	if client == nil {
		return "", fmt.Errorf("provided nil client")
	}

	// Get the CSV objects
	csv, err := olm.ListClusterServiceVersion(client, operatorNamespace, metav1.ListOptions{})

	// Check for any error getting the CSVs
	if err != nil {
		return "", err
	}

	// Find the CSV that matches the operator
	for _, csv := range csv {
		if strings.Contains(csv.Object.Name, operatorName) {
			return csv.Object.Spec.Version.String(), nil
		}
	}

	return "", nil
}

// GetZtpVersionFromArgocd is used to fetch the version of the ztp-site-generator init container.
func GetZtpVersionFromArgocd(client *clients.Settings, name string, namespace string) (string, error) {
	deployment, err := deployment.Pull(client, name, namespace)
	if err != nil {
		return "", err
	}

	for _, container := range deployment.Object.Spec.Template.Spec.InitContainers {
		// Legacy 4.11 uses the image name as `ztp-site-generator`
		// While 4.12+ uses the image name as `ztp-site-generate`
		// So just check for `ztp-site-gen` to cover both
		if strings.Contains(container.Image, "ztp-site-gen") {
			arr := strings.Split(container.Image, ":")
			// Get the image tag which is the last element
			ztpVersion := arr[len(arr)-1]

			if ztpVersion == "latest" {
				log.Println("Site generator version tag was 'latest', so returning empty version")

				return "", nil
			}

			// The format here will be like vX.Y.Z so we need to remove the v at the start
			fmt.Println("ztpVersion = ", ztpVersion)

			return ztpVersion[1:], nil
		}
	}

	return "", fmt.Errorf("unable to identify ztp version")
}
