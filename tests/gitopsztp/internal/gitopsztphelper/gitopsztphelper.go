package gitopsztphelper

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	"github.com/openshift-kni/k8sreporter"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-kni/eco-goinfra/pkg/argocd"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpparams"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/talmparams"
)

// ztp related vars.
var (
	// test image.
	CnfTestImage   = "quay.io/openshift-kni/cnf-tests:4.8"
	HubAPIClient   *clients.Settings
	HubName        string
	SpokeAPIClient *clients.Settings
	SpokeName      string
	ArgocdApps     = map[string]gitopsztpparams.ArgocdGitDetails{}
	ZtpVersion     string
	AcmVersion     string
	TalmVersion    string
	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		gitopsztpparams.ZtpTestNamespace: "",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}
)

// SetGitDetailsInArgocd is used to update the git repo, branch, and path in the Argocd app.
func SetGitDetailsInArgocd(gitRepo, gitBranch, gitPath, argocdApp string, waitForSync, syncMustBeValid bool) error {
	app, err := argocd.PullApplication(HubAPIClient, argocdApp, gitopsztpparams.OpenshiftGitops)
	if err != nil {
		return err
	}

	if app.Object.Spec.Source.RepoURL == gitRepo &&
		app.Object.Spec.Source.TargetRevision == gitBranch &&
		app.Object.Spec.Source.Path == gitPath {
		log.Println("Provided git details are the already configured details in Argocd. No change required.")

		return nil
	}

	app.WithGitDetails(gitRepo, gitBranch, gitPath)

	_, err = app.Update(true)
	if err != nil {
		return err
	}

	if waitForSync {
		err = WaitUntilArgocdChangeIsCompleted(argocdApp, syncMustBeValid, gitopsztpparams.ArgocdChangeTimeout)
		if err != nil {
			return err
		}
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
			talmparams.OperatorHubTalmNamespace,
			talmparams.OpenshiftOperatorNamespace,
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

// GetZtpContext is used to get the context for the Ztp test client interactions.
func GetZtpContext() context.Context {
	return context.Background()
}

// GetAllTestClients is used to quickly obtain a list of all the test clients.
func GetAllTestClients() []*clients.Settings {
	return []*clients.Settings{
		HubAPIClient,
		SpokeAPIClient,
	}
}

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

// JoinGitPaths is used to join any combination of git strings but also avoiding double slashes.
func JoinGitPaths(inputs []string) string {
	// We want to preserve any existing double slashes but we don't want to add any between the input elements
	// To work around this we will use a special join character and a couple replacements
	special := "<<join>>"

	// Combine the inputs with the special join character
	result := strings.Join(
		inputs,
		special,
	)

	// Replace any special joins that have a slash prefix
	result = strings.ReplaceAll(result, "/"+special, "/")

	// Replace any special joins that have a slash suffix
	result = strings.ReplaceAll(result, special+"/", "/")

	// Finally replace any remaining special joins
	result = strings.ReplaceAll(result, special, "/")

	// The final result should never have double slashes between the joined elements
	// However if they already had any double slashes, e.g. "http://", they will be preserved
	return result
}

// DoesGitPathExist checks if the specified git url exists.
func DoesGitPathExist(gitURL, gitBranch, gitPath string) bool {
	// Combine the separate pieces to get the url
	// We also need to remove the ".git" from the end of the url
	url := JoinGitPaths(
		[]string{
			strings.Replace(gitURL, ".git", "", 1),
			"raw",
			gitBranch,
			gitPath,
		},
	)

	// Log the url we are trying
	log.Printf("Checking if git url '%s' exists\n", url)

	// Create a custom http.Transport with insecure TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Create a custom http.Client using the insecure transport
	client := &http.Client{Transport: transport}

	// Make a request using the custom client
	resp, err := client.Get(url)

	// Check if we got a valid response
	if err == nil && resp.StatusCode == 200 {
		log.Printf("found valid git url for '%s'\n", gitPath)

		return true
	}

	// If we got here then none of the urls could be found.
	log.Printf("could not find valid url for '%s'\n", gitPath)

	return false
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

// WaitUntilArgocdChangeIsCompleted is used to wait until Argocd has updated its configuration.
func WaitUntilArgocdChangeIsCompleted(appName string, syncMustBeValid bool, timeout time.Duration) error {
	log.Println("Waiting for Argocd change to finish syncing")

	err := wait.PollImmediate(gitopsztpparams.ArgocdChangeInterval, timeout, func() (bool, error) {
		log.Println("Checking if argo change is complete...")

		app, err := argocd.PullApplication(HubAPIClient, appName, gitopsztpparams.OpenshiftGitops)
		if err != nil {
			return false, err
		}

		if app != nil {
			// If there are any conditions then it probably means theres a problem
			// By printing them here we can make diagnosing a failing test easier
			for index, condition := range app.Object.Status.Conditions {
				log.Printf("Condition #%d: '%s'\n", index, condition)
			}

			// The sync result may also have helpful information in the event of an error
			if app.Object.Status.OperationState != nil && app.Object.Status.OperationState.SyncResult != nil {
				for index, resource := range app.Object.Status.OperationState.SyncResult.Resources {
					if resource != nil {
						log.Printf("Sync resource #%d: '%s\n", index, resource)
					}
				}
			}

			// Check if all the git details match
			if app.Object.Status.Sync.ComparedTo.Source.RepoURL == app.Object.Spec.Source.RepoURL &&
				app.Object.Status.Sync.ComparedTo.Source.TargetRevision == app.Object.Spec.Source.TargetRevision &&
				app.Object.Status.Sync.ComparedTo.Source.Path == app.Object.Spec.Source.Path {
				// If we expect the sync to be successful then we also need to check that the status is 'Synced'
				if syncMustBeValid {
					return app.Object.Status.Sync.Status == "Synced", nil
				}

				return true, nil
			}
		}

		return false, nil
	})

	return err
}

// GetGitDetailsFromArgocd is used to get the current git repo, branch, and path in the Argocd app.
func GetGitDetailsFromArgocd(appName, namespace string) (string, string, string, error) {
	app, err := argocd.PullApplication(HubAPIClient, appName, namespace)
	if err != nil {
		return "", "", "", err
	}

	return app.Object.Spec.Source.RepoURL, app.Object.Spec.Source.TargetRevision, app.Object.Spec.Source.Path, nil
}

// GetArgocdAppGitDetails is used to check the environment variables for any ztp test configuration.
// If any are undefined then the default values are used instead.
func GetArgocdAppGitDetails() error {
	// Check if the hub is defined
	if os.Getenv(gitopsztpparams.HubKubeEnvKey) != "" {
		// Loop over the apps and save the git details
		for _, app := range gitopsztpparams.ArgocdApps {
			repo, branch, dir, err := GetGitDetailsFromArgocd(app, gitopsztpparams.OpenshiftGitops)
			if err != nil {
				return err
			}

			// Save the git details to the map
			ArgocdApps[app] = gitopsztpparams.ArgocdGitDetails{
				Repo:   repo,
				Branch: branch,
				Path:   dir,
			}
		}
	}

	return nil
}

// ResetArgocdGitDetails is used to configure Argocd back to the values it had before the tests started.
func ResetArgocdGitDetails() error {
	if os.Getenv(gitopsztpparams.HubKubeEnvKey) != "" {
		// Loop over the apps and restore the git details
		for _, app := range gitopsztpparams.ArgocdApps {
			// Restore the app's git details
			err := SetGitDetailsInArgocd(
				ArgocdApps[app].Repo,
				ArgocdApps[app].Branch,
				ArgocdApps[app].Path,
				app,
				false,
				false,
			)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CreatePrivilegedPods creates privileged test pods on worker nodes to assist testing
// Returns a map with nodeName as key and pod pointer as value, and error if occurred.
func CreatePrivilegedPods(image string) (map[string]*pod.Builder, error) {
	if image == "" {
		image = CnfTestImage
	}
	// Create cnfgotestpriv namespace if not already created
	namespace := namespace.NewBuilder(HubAPIClient, gitopsztpparams.PrivPodNamespace)
	if !namespace.Exists() {
		log.Println("Creating namespace:", gitopsztpparams.PrivPodNamespace)

		_, err := namespace.Create()
		if err != nil {
			return nil, err
		}
	}
	// Launch priv pods on nodes with worker role so it can be successfully scheduled.
	workerNodesList, err := nodes.List(HubAPIClient, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return nil, err
	}

	privPods := make(map[string]*pod.Builder)

	for _, workerNode := range workerNodesList {
		podName := fmt.Sprintf("%s-%s", gitopsztpparams.PrivPodNamespace, workerNode.Object.Name)
		privilegedPod := pod.NewBuilder(HubAPIClient, podName, gitopsztpparams.PrivPodNamespace, image)

		_, err := privilegedPod.WithPrivilegedFlag().DefineOnNode(workerNode.Object.Name).WithLocalVolume(
			"rootfs", "/rootfs").WithHostPid(true).CreateAndWaitUntilRunning(
			10 * time.Minute)
		if err != nil {
			return nil, err
		}

		privPods[workerNode.Object.Name] = privilegedPod
	}

	return privPods, nil
}
