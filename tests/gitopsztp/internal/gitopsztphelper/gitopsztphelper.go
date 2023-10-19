package gitopsztphelper

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// HubName hub cluster name.
	HubName string
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
	var err error

	if os.Getenv(gitopsztpparams.HubKubeEnvKey) != "" {
		HubName, err = cluster.GetClusterName(gitopsztpparams.HubKubeEnvKey)
		if err != nil {
			return err
		}

		ocpVersion, err := cluster.GetClusterVersion(gitopsztpinittools.HubAPIClient)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has OCP version '%s'\n", HubName, ocpVersion)

		AcmVersion, err = GetOperatorVersionFromCSV(
			gitopsztpinittools.HubAPIClient,
			gitopsztpparams.AcmOperatorName,
			gitopsztpparams.AcmOperatorNamespace,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has ACM version '%s'\n", HubName, AcmVersion)

		ZtpVersion, err = GetZtpVersionFromArgocd(
			gitopsztpinittools.HubAPIClient,
			gitopsztpparams.OpenshiftGitopsRepoServer,
			gitopsztpparams.OpenshiftGitops,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has ZTP version '%s'\n", HubName, ZtpVersion)

		TalmVersion, err = GetOperatorVersionFromCSV(
			gitopsztpinittools.HubAPIClient,
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
		SpokeName, err = cluster.GetClusterName(gitopsztpparams.SpokeKubeEnvKey)
		if err != nil {
			return err
		}

		ocpVersion, err := cluster.GetClusterVersion(gitopsztpinittools.SpokeAPIClient)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has OCP version '%s'\n", SpokeName, ocpVersion)
	}

	return nil
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

// IsPodHealthy returns true if a given pod is healthy, otherwise false.
func IsPodHealthy(pod *pod.Builder) bool {
	if pod.Object.Status.Phase == v1.PodRunning {
		if !isPodInCondition(pod, v1.PodReady) {
			glog.Fatalf("pod condition is not Ready. Message: %s", pod.Object.Status.Message)

			return false
		} else if pod.Object.Status.Phase != v1.PodSucceeded {
			// Pod is not running or completed.
			glog.Fatalf("pod phase is %s. Message: %s", pod.Object.Status.Phase, pod.Object.Status.Message)

			return false
		}
	}

	return true
}

// isPodInCondition returns true if a given pod is in expected condition, otherwise false.
func isPodInCondition(pod *pod.Builder, condition v1.PodConditionType) bool {
	for _, c := range pod.Object.Status.Conditions {
		if c.Type == condition && c.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}

// IsContainerExistInPod check if a given container exists in a given pod.
func IsContainerExistInPod(pod *pod.Builder, containerName string) bool {
	containers := pod.Object.Status.ContainerStatuses

	for _, container := range containers {
		if container.Name == containerName {
			glog.V(100).Infof("found %s container\n", containerName)

			return true
		}
	}

	return false
}
