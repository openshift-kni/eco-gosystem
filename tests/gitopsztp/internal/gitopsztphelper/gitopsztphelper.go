package gitopsztphelper

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
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

// IsVersionStringInRange can be used to check if a version string is between a specified min and max value.
// All the string inputs to this function should be dot separated positive intergers, e.g. "1.0.0" or "4.10".
// Each string inputs must be at least two dot separarted integers but may also be 3 or more.
func IsVersionStringInRange(version, minimum, maximum string) bool {
	// First we will define a helper to validate that a string is at least two dot separated positive integers
	ValidateInputString := func(input string) (bool, []int) {
		// Split the input string on the dot character
		versionSplits := strings.Split(input, ".")

		// We need at least two digits to verify
		if len(versionSplits) < 2 {
			return false, []int{}
		}

		// Prepare a list of digits to return later
		digits := []int{}

		// Check if the first two splits are valid integers
		for i := 0; i < 2; i++ {
			// Attempt to convert the string to an integer
			digit, err := strconv.Atoi(versionSplits[i])
			if err != nil {
				return false, []int{}
			}

			// Save the digit
			digits = append(digits, digit)
		}

		// Otherwise if we've passed all the checks it is a valid version string
		return true, digits
	}

	versionValid, versionDigits := ValidateInputString(version)
	minimumValid, minimumDigits := ValidateInputString(minimum)
	maximumValid, maximumDigits := ValidateInputString(maximum)

	if !minimumValid {
		// We don't want any bad minimums to be passed at all, so panic if minimum wasn't an empty string
		if minimum != "" {
			panic(fmt.Errorf("invalid minimum provided: '%s'", minimum))
		}

		// Assume the minimum digits are [0,0] for later comparison
		minimumDigits = []int{0, 0}
	}

	if !maximumValid {
		// We don't want any bad maximums to be passed at all, so panic if maximum wasn't an empty string
		if maximum != "" {
			panic(fmt.Errorf("invalid maximum provided: '%s'", maximum))
		}

		// Assume the maximum digits are [math.MaxInt, math.MaxInt] for later comparison
		maximumDigits = []int{math.MaxInt, math.MaxInt}
	}

	// If the version was not valid then we need to check the min and max
	if !versionValid {
		// If no min or max was defined then return true
		if !minimumValid && !maximumValid {
			return true
		}

		// Otherwise return whether the input maximum was an empty string or not
		return maximum == ""
	}

	// Otherwise the versions were valid so compare the digits
	for i := 0; i < 2; i++ {
		// The version bit should be between the minimum and maximum
		if versionDigits[i] < minimumDigits[i] || versionDigits[i] > maximumDigits[i] {
			return false
		}
	}

	// At the end if we never returned then all the digits were in valid range
	return true
}

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
