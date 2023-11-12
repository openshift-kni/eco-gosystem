package talmhelper

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Spoke2APIClient api client of the second spoke.
	Spoke2APIClient *clients.Settings
	// Spoke2Name name of the second spoke.
	Spoke2Name string
)

// talm consts.
const (
	CguName                         string = "talm-cgu"
	Namespace                       string = "talm-namespace"
	PlacementBindingName            string = "talm-placement-binding"
	PlacementRule                   string = "talm-placement-rule"
	PolicyName                      string = "talm-policy"
	PolicySetName                   string = "talm-policyset"
	CatalogSourceName               string = "talm-catsrc"
	Talm411TimeoutMessage           string = "The ClusterGroupUpgrade CR policies are taking too long to complete"
	Talm412TimeoutMessage           string = "Policy remediation took too long"
	TemporaryNamespaceName          string = Namespace + "-temp"
	ProgressingType                 string = "Progressing"
	ReadyType                       string = "Ready"
	SucceededType                   string = "Succeeded"
	ValidatedType                   string = "Validated"
	ConditionReasonCompleted        string = "Completed"
	ConditionReasonUpgradeCompleted string = "UpgradeCompleted"
)

// VerifyTalmIsInstalled checks that talm pod+container is present and that CGUs can be fetched.
func VerifyTalmIsInstalled() error {
	// Check for talm pods
	talmPods, err := pod.List(gitopsztpinittools.HubAPIClient,
		talmparams.TalmOperatorNamespace,
		metav1.ListOptions{
			LabelSelector: talmparams.TalmPodLabelSelector})
	if err != nil {
		return err
	}

	// Check if any pods exist
	if len(talmPods) == 0 {
		return errors.New("unable to find talm pod")
	}

	// Check each pod for the talm container
	for _, talmPod := range talmPods {
		if !gitopsztphelper.IsPodHealthy(talmPod) {
			return fmt.Errorf("talm pod %s is not healthy", talmPod.Definition.Name)
		}

		if !gitopsztphelper.IsContainerExistInPod(talmPod, talmparams.TalmContainerName) {
			return fmt.Errorf("talm pod defined but talm container does not exist")
		}
	}

	return nil
}

// DeleteTalmTestNamespace deletes the TALM test namespace.
func DeleteTalmTestNamespace(allowNotFound bool) error {
	glog.V(100).Info("Deleting talm namespace")

	// Hub may be optional depending on what tests are running
	if os.Getenv(talmparams.HubKubeEnvKey) != "" {
		glog.V(100).Info("Deleting talm namespace on hub")

		err := namespace.NewBuilder(gitopsztpinittools.HubAPIClient,
			talmparams.TalmTestNamespace).DeleteAndWait(5 * time.Minute)
		if err != nil {
			if !allowNotFound || (allowNotFound && !k8sErr.IsNotFound(err)) {
				return err
			}
		}
	}

	// Spoke1 is the default kubeconfig
	if os.Getenv(talmparams.Spoke1KubeEnvKey) != "" {
		glog.V(100).Info("Deleting talm namespace on spoke1")

		err := namespace.NewBuilder(gitopsztpinittools.SpokeAPIClient,
			talmparams.TalmTestNamespace).DeleteAndWait(5 * time.Minute)
		if err != nil {
			if !allowNotFound || (allowNotFound && !k8sErr.IsNotFound(err)) {
				return err
			}
		}
	}

	return nil
}

// CreateTalmTestNamespace deletes the TALM test namespace.
func CreateTalmTestNamespace() error {
	glog.V(100).Info("Creating talm namespace")

	// Hub may be optional depending on what tests are running
	if os.Getenv(talmparams.HubKubeEnvKey) != "" {
		glog.V(100).Info("Creating talm namespace on hub")

		_, err := namespace.NewBuilder(gitopsztpinittools.HubAPIClient,
			talmparams.TalmTestNamespace).Create()
		if err != nil {
			return err
		}
	}

	// Spoke1 is the default kubeconfig
	if os.Getenv(talmparams.Spoke1KubeEnvKey) != "" {
		glog.V(100).Info("Creating talm namespace on spoke1")

		_, err := namespace.NewBuilder(gitopsztpinittools.SpokeAPIClient,
			talmparams.TalmTestNamespace).Create()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAllTestClients is used to quickly obtain a list of all the test clients.
func GetAllTestClients() []*clients.Settings {
	return []*clients.Settings{
		gitopsztpinittools.HubAPIClient,
		gitopsztpinittools.SpokeAPIClient,
		Spoke2APIClient,
	}
}

// CleanupTestResourcesOnClients is used to delete all references to specified cgu,
// policy, and namespace on all clusters specified in the client list.
func CleanupTestResourcesOnClients(
	clients []*clients.Settings,
	cguName string,
	policyName string,
	createdNamespace string,
	placementBinding string,
	placementRule string,
	policySet string,
	catsrcName string) []error {
	// Create a list of cleanupErrors
	var cleanupErrors []error

	// Loop over all clients for cleanup
	for _, client := range clients {
		// Perform the deletes
		glog.V(100).Infof("cleaning up resources on client '%s'", client.Config.Host)

		cleanupErr := CleanupTestResourcesOnClient(
			client,
			cguName,
			policyName,
			createdNamespace,
			placementBinding,
			placementRule,
			policySet,
			catsrcName,
			true,
		)
		if len(cleanupErr) != 0 {
			cleanupErrors = append(cleanupErrors, cleanupErr...)
		}
	}

	return cleanupErrors
}

// CleanupTestResourcesOnClient is used to delete everything on a specific cluster.
func CleanupTestResourcesOnClient(
	client *clients.Settings,
	cguName string,
	policyName string,
	nsName string,
	placementBinding string,
	placementRule string,
	policySet string,
	catsrcName string,
	deleteNs bool,
) []error {
	// Create a list of errorList
	var errorList []error

	// Check for nil client first
	if client == nil {
		errorList = append(errorList, errors.New("provided nil client"))

		return errorList
	}

	// Attempt to delete cgu
	glog.V(100).Infof("Deleting cgu '%s'", cguName)

	// delete cgu
	cgu, err := cgu.Pull(client, cguName, nsName)
	if err != nil {
		errorList = append(errorList, err)
	}

	_, err = cgu.Delete()
	if err != nil {
		errorList = append(errorList, err)
	}

	// delete policy
	policy, err := ocm.Pull(client, policyName, nsName)
	if err != nil {
		errorList = append(errorList, err)
	}

	_, err = policy.Delete()
	if err != nil {
		errorList = append(errorList, err)
	}

	// delete namespace
	if nsName != "" && deleteNs {
		glog.V(100).Infof("Deleting namespace '%s'", nsName)

		ns := namespace.NewBuilder(client, nsName)
		if ns.Exists() {
			err := ns.DeleteAndWait(5 * time.Minute)
			if err != nil {
				errorList = append(errorList, err)
			}
		}
	}

	return errorList
}

// CleanupNamespace is used to cleanup a namespace on multiple clients.
func CleanupNamespace(clients []*clients.Settings, ns string) error {
	for _, client := range clients {
		namespace := namespace.NewBuilder(client, ns)
		if namespace.Exists() {
			err := namespace.DeleteAndWait(5 * time.Minute)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
