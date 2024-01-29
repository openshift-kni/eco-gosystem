package talmhelper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/cluster-group-upgrades-operator/pkg/api/clustergroupupgrades/v1alpha1"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"

	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	configurationPolicyv1 "open-cluster-management.io/config-policy-controller/api/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	policiesv1beta1 "open-cluster-management.io/governance-policy-propagator/api/v1beta1"

	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmparams"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
)

var (
	// Spoke1Name name of the first spoke.
	Spoke1Name string
	// Spoke2APIClient api client of the second spoke.
	Spoke2APIClient *clients.Settings
	// Spoke2Name name of the second spoke.
	Spoke2Name string
	// TalmHubVersion talm hub version on hub.
	TalmHubVersion string
)

// talm consts.
const (
	CguName                         string = "talm-cgu"
	Namespace                       string = "ztp-install"
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
	talmPods, err := pod.List(ranfuncinittools.HubAPIClient,
		talmparams.OpenshiftOperatorNamespace,
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
		if !ranfunchelper.IsPodHealthy(talmPod) {
			return fmt.Errorf("talm pod %s is not healthy", talmPod.Definition.Name)
		}

		if !ranfunchelper.IsContainerExistInPod(talmPod, talmparams.TalmContainerName) {
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

		err := namespace.NewBuilder(ranfuncinittools.HubAPIClient,
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

		err := namespace.NewBuilder(ranfuncinittools.SpokeAPIClient,
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

		_, err := namespace.NewBuilder(ranfuncinittools.HubAPIClient,
			talmparams.TalmTestNamespace).Create()
		if err != nil {
			return err
		}
	}

	// Spoke1 is the default kubeconfig
	if os.Getenv(talmparams.Spoke1KubeEnvKey) != "" {
		glog.V(100).Info("Creating talm namespace on spoke1")

		_, err := namespace.NewBuilder(ranfuncinittools.SpokeAPIClient,
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
		ranfuncinittools.HubAPIClient,
		ranfuncinittools.SpokeAPIClient,
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
	policy, err := ocm.PullPolicy(client, policyName, nsName)
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

// GetCguDefinition is used to get a CGU with simplified parameters.
func GetCguDefinition(
	cguName string,
	clusterList []string,
	canaryList []string,
	managedPolicies []string,
	namespace string,
	maxConcurrency int,
	timeout int) cgu.CguBuilder {
	customResource := cgu.CguBuilder{
		Definition: &v1alpha1.ClusterGroupUpgrade{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterGroupUpgrade",
				APIVersion: v1alpha1.SchemeGroupVersion.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cguName,
				Namespace: namespace,
			},
			Spec: v1alpha1.ClusterGroupUpgradeSpec{
				Backup:                false,
				PreCaching:            false,
				Enable:                BoolAddr(true),
				Clusters:              clusterList,
				ClusterLabelSelectors: nil,
				ManagedPolicies:       managedPolicies,
				RemediationStrategy: &v1alpha1.RemediationStrategySpec{
					MaxConcurrency: maxConcurrency,
					Timeout:        timeout,
					Canaries:       canaryList,
				},
			},
		},
	}

	return customResource
}

// BoolAddr is used to convert a boolean to a boolean pointer.
func BoolAddr(b bool) *bool {
	boolVar := b

	return &boolVar
}

// IsClusterStartedInCgu can be used to check if a particular cluster has started
// being remediated in the provided cgu and namespace.
func IsClusterStartedInCgu(
	client *clients.Settings,
	cguName string,
	clusterName string,
	namespace string) (bool, error) {
	cgu, err := cgu.Pull(client, cguName, namespace)
	if err != nil {
		return false, err
	}

	clusterStatus := cgu.Object.Status.Status.CurrentBatchRemediationProgress[clusterName]
	if clusterStatus != nil {
		if clusterStatus.State != "NotStarted" {
			return true, nil
		}
	}

	return false, nil
}

// WaitForCguToFinishSuccessfully waits until the provided CGU reaches the succeeded state
// and all clusters were successful message.
func WaitForCguToFinishSuccessfully(cguName string, namespace string, timeout time.Duration) error {
	// Wait for the cgu to finish
	glog.V(100).Info("waiting for CGU to finish successfully")

	// TALM uses different conditions starting in 4.12
	conditionType := SucceededType
	conditionReason := ConditionReasonCompleted

	if !ranfunchelper.IsVersionStringInRange(
		TalmHubVersion,
		talmparams.TalmUpdatedConditionsVersion,
		"",
	) {
		conditionType = ReadyType
		conditionReason = ConditionReasonUpgradeCompleted
	}

	return WaitForCguInCondition(
		ranfuncinittools.HubAPIClient,
		cguName,
		namespace,
		conditionType,
		"",
		metav1.ConditionTrue,
		conditionReason,
		timeout,
	)
}

// WaitForCguInCondition waits until a specified condition type
// matches the provided message and/or status.
func WaitForCguInCondition(
	client *clients.Settings,
	cguName string,
	namespace string,
	conditionType string,
	expectedMessage string,
	expectedStatus metav1.ConditionStatus,
	expectedReason string,
	timeout time.Duration) error {
	// This will be used inside the wait to keep track of the current status
	// Since we can't return an error outside the poll we need to save it outside the loop
	// and then return it after timeout occurs.
	var lastStatus error

	msg := fmt.Sprintf("Waiting for cgu %s in namespace %s to be in condition: %s=%s",
		cguName, namespace, conditionType, expectedStatus)
	if expectedReason != "" {
		msg = msg + ", with reason: " + expectedReason
	}

	glog.V(100).Info(msg)
	// Use a poll to check the cgu condition
	_ = wait.PollUntilContextTimeout(
		context.TODO(), talmparams.TalmTestPollInterval, timeout, true, func(context.Context) (done bool, err error) {
			// Get the CGU
			clusterGroupUpgrade, err := cgu.Pull(client, cguName, namespace)
			if err != nil {
				lastStatus = err

				return false, err
			}

			// Get the condition
			condition := meta.FindStatusCondition(clusterGroupUpgrade.Object.Status.Conditions, conditionType)

			// If the condition does not exist that is not considered a match
			if condition == nil {
				lastStatus = fmt.Errorf("condition for type '%s' was nil", conditionType)

				return false, nil
			}

			// Check the status if it was defined
			if expectedStatus != "" {
				if condition.Status != expectedStatus {
					lastStatus = fmt.Errorf(
						"actual status '%s' did not match expected status '%s'",
						condition.Status,
						expectedStatus,
					)

					return false, nil
				}
			}

			// Check the message if it was defined
			if expectedMessage != "" {
				if !strings.Contains(condition.Message, expectedMessage) {
					lastStatus = fmt.Errorf(
						"actual message '%s' did not contain expected message '%s'",
						condition.Message,
						expectedMessage,
					)

					return false, nil
				}
			}

			// Check the reason if it was defined
			if expectedReason != "" {
				if condition.Reason != expectedReason {
					lastStatus = fmt.Errorf(
						"actual reason '%s' did not match expected reason '%s'",
						condition.Reason,
						expectedReason,
					)

					return false, nil
				}
			}

			// If it did match we will return nil here to exit the eventually
			lastStatus = nil

			return true, nil
		},
	)

	return lastStatus
}

// WaitForClusterInProgressInCgu can be used to wait until the provided cluster is actively
// being remediated in the provided cgu and namespace.
func WaitForClusterInProgressInCgu(
	client *clients.Settings,
	cguName string,
	clusterName string,
	namespace string,
	timeout time.Duration) error {
	// Print the current check
	glog.V(100).Infof("Waiting until cluster '%s' in progress in cgu '%s'",
		clusterName,
		cguName,
	)

	err := wait.PollUntilContextTimeout(
		context.TODO(), 15*time.Second, timeout, true, func(context.Context) (bool, error) {
			ok, err := IsClusterInProgressInCgu(client, cguName, clusterName, namespace)
			if err != nil {
				return true, err
			}

			return ok, nil
		},
	)

	return err
}

// IsClusterInProgressInCgu can be used to check if a particular cluster is actively
// being remediated in the provided cgu and namespace.
func IsClusterInProgressInCgu(
	client *clients.Settings,
	cguName string,
	clusterName string,
	namespace string) (bool, error) {
	cgu, err := cgu.Pull(client, cguName, namespace)
	if err != nil {
		return false, err
	}

	clusterStatus := cgu.Object.Status.Status.CurrentBatchRemediationProgress[clusterName]
	if clusterStatus != nil {
		if clusterStatus.State == "InProgress" {
			return true, nil
		}
	}

	return false, nil
}

// CreateCguAndWait is used to create a CGU with the provided clusters list and managed policies.
// No policies, placements, bindings, policysets, etc will be created.
func CreateCguAndWait(
	client *clients.Settings,
	clusterGroupUpgrade cgu.CguBuilder) error {
	if client == nil {
		return errors.New("provided nil client")
	}

	if len(clusterGroupUpgrade.Definition.Spec.Clusters) > 0 {
		for _, cluster := range clusterGroupUpgrade.Definition.Spec.Clusters {
			if cluster == "" {
				return errors.New("provided empty cluster in clustersList")
			}
		}
	}

	if len(clusterGroupUpgrade.Definition.Spec.ManagedPolicies) > 0 {
		for _, policy := range clusterGroupUpgrade.Definition.Spec.ManagedPolicies {
			if policy == "" {
				return errors.New("provided empty policy in managedPolicies")
			}
			// If the fully generated name of the talm enforce policy is > 63 characters then they will just not work.
			// There is some wiggle room here since there is an additional identifier on the end of the policy.
			// So instead of hard erroring just print a warning if the length is possibly an issue.
			if len(policy)+len(clusterGroupUpgrade.Definition.Name) > 50 {
				glog.V(100).Info("Warning: Length of generated TALM policies may exceed character limit and not work")
			}
		}
	}

	if clusterGroupUpgrade.Definition.Name == "" {
		return errors.New("provided empty cguName")
	}

	_, err := clusterGroupUpgrade.Create()
	if err != nil {
		return err
	}

	err = WaitUntilObjectExists(
		client,
		clusterGroupUpgrade.Definition.Name,
		clusterGroupUpgrade.Definition.Namespace,
		IsCguExist,
	)

	return err
}

// IsCguExist can be used to check if a specific cgu exists.
func IsCguExist(client *clients.Settings, cguName string, namespace string) (bool, error) {
	glog.V(100).Infof("Checking for existence of cgu '%s' in namespace '%s'\n", cguName, namespace)

	_, err := cgu.Pull(client, cguName, namespace)

	// Filter errors that don't matter
	filtered := FilterMissingResourceErrors(err)

	// If err was defined but filtered is nil, then the resource no longer exists
	if err != nil && filtered == nil {
		return false, nil
	}

	// If filtered it not nil then an actual error occurred
	if filtered != nil {
		return false, filtered
	}

	// Else assume it exists
	return true, nil
}

// WaitUntilObjectExists can be called to wait until a specified resource is present.
// This is called by all of the CreateXAndWait functions in this file.
func WaitUntilObjectExists(
	client *clients.Settings,
	objectName string,
	namespace string,
	getStatus func(client *clients.Settings, objectName string, namespace string) (bool, error)) error {
	// Wait for it to exist
	err := wait.PollUntilContextTimeout(
		context.TODO(), 15*time.Second, 5*time.Minute, true,
		func(context.Context) (bool, error) {
			status, err := getStatus(client, objectName, namespace)

			// Wait until it definitely exists
			if err == nil && status {
				return true, nil
			}

			// Assume it does not exist otherwise
			return false, nil
		},
	)

	return err
}

// FilterMissingResourceErrors takes an input error and checks it for a few specific types of errors.
// If it matches any of the errors that are considered to be a resource not found, then it returns nil.
// Otherwise it returns the original error.
func FilterMissingResourceErrors(err error) error {
	// If the error was nil just immediately return
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "server could not find the requested resource") {
		return nil
	}

	if strings.Contains(err.Error(), "no matches for kind") {
		return nil
	}

	if strings.Contains(err.Error(), "not found") {
		return nil
	}

	return err
}

// CreatePolicyAndWait is used to create a policy and wait for it to exist.
func CreatePolicyAndWait(client *clients.Settings, policy ocm.PolicyBuilder) error {
	// Create the policy
	_, err := policy.Create()
	if err != nil {
		return err
	}

	// Wait for it to exist
	err = WaitUntilObjectExists(
		client,
		policy.Definition.Name,
		policy.Definition.Namespace,
		IsPolicyExist,
	)

	return err
}

// IsPolicyExist can be used to check if a specific policy exists.
func IsPolicyExist(client *clients.Settings, policyName string, namespace string) (bool, error) {
	glog.V(100).Info("Checking for existence of policy '%s' in namespace '%s'\n", policyName, namespace)

	// We can use another helper to get the object
	policy, err := ocm.PullPolicy(client, policyName, namespace)

	// Filter any missing resource errors before checking the result
	err = FilterMissingResourceErrors(err)
	if err != nil {
		return false, err
	}

	return policyName == policy.Object.Name, nil
}

// CreatePolicyAndCgu is used to create a simplified CGU to cover the most common use case.
// This will automatically create a single policy enforcing the compliance type on the provided object.
// If you require multiple policies to be managed then consider CreateCgu() instead.
func CreatePolicyAndCgu(
	client *clients.Settings,
	object runtime.Object,
	complianceType configurationPolicyv1.ComplianceType,
	remediationAction configurationPolicyv1.RemediationAction,
	policyName string,
	policySetName string,
	placementBindingName string,
	placementRule string,
	namespace string,
	clusterSelector metav1.LabelSelector,
	clusterGroupUpgrade cgu.CguBuilder) error {
	// Step 1 - Create simple policy with all required components
	err := CreatePolicyWithAllComponents(
		client,
		object,
		complianceType,
		remediationAction,
		policyName,
		policySetName,
		placementBindingName,
		placementRule,
		namespace,
		clusterGroupUpgrade.Definition.Spec.Clusters,
		clusterSelector,
	)
	if err != nil {
		return err
	}

	// Step 2 - Create the cgu
	err = CreateCguAndWait(
		client,
		clusterGroupUpgrade,
	)
	if err != nil {
		return err
	}

	return nil
}

// CreatePolicyWithAllComponents is used to create a policy and all the requireed components for
// applying that policy such as policyset, placementrule, placement binding, etc.
func CreatePolicyWithAllComponents(
	client *clients.Settings,
	object runtime.Object,
	complianceType configurationPolicyv1.ComplianceType,
	remediationAction configurationPolicyv1.RemediationAction,
	policyName string,
	policySetName string,
	placementBindingName string,
	placementRule string,
	namespace string,
	clusters []string,
	clusterSelector metav1.LabelSelector,
) error {
	// Step 0 - Validate inputs
	if client == nil {
		return errors.New("provided nil client")
	}

	if object == nil {
		return errors.New("provided nil object")
	}

	if policyName == "" {
		return errors.New("provided empty policyName")
	}

	if policySetName == "" {
		return errors.New("provided empty policySetName")
	}

	if placementBindingName == "" {
		return errors.New("provided empty placementBindingName")
	}

	if placementRule == "" {
		return errors.New("provided empty placementRule")
	}

	if namespace == "" {
		return errors.New("provided empty namespace")
	}

	// Step 1 - Create the policy
	glog.V(100).Info("create the policy that the cgu will apply")

	configurationPolicy := GetConfigurationPolicyDefinition(policyName, complianceType, remediationAction, object)

	cguPolicy := GetPolicyDefinition(policyName, namespace, &configurationPolicy, remediationAction)

	return ApplyPolicyAndCreateAllComponents(client,
		cguPolicy,
		policySetName,
		placementBindingName,
		placementRule,
		namespace,
		clusters,
		clusterSelector)
}

// GetConfigurationPolicyDefinition is used to get a configuration policy that contains the provided object.
func GetConfigurationPolicyDefinition(
	policyName string,
	complianceType configurationPolicyv1.ComplianceType,
	remediationAction configurationPolicyv1.RemediationAction,
	object runtime.Object) configurationPolicyv1.ConfigurationPolicy {
	customResource := configurationPolicyv1.ConfigurationPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigurationPolicy",
			APIVersion: "policy.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-config", policyName),
		},
		Spec: &configurationPolicyv1.ConfigurationPolicySpec{
			Severity:          "low",
			RemediationAction: remediationAction,
			NamespaceSelector: configurationPolicyv1.Target{
				Include: []configurationPolicyv1.NonEmptyString{"kube-*"},
				Exclude: []configurationPolicyv1.NonEmptyString{"*"},
			},
			ObjectTemplates: []*configurationPolicyv1.ObjectTemplate{
				{
					ComplianceType: complianceType,
					ObjectDefinition: runtime.RawExtension{
						Object: object,
					},
				},
			},
			EvaluationInterval: configurationPolicyv1.EvaluationInterval{
				Compliant:    "10s",
				NonCompliant: "10s",
			},
		},
	}

	return customResource
}

// GetPolicyDefinition is used to get a policy that can be used with a CGU.
func GetPolicyDefinition(
	policyName string,
	namespace string,
	object runtime.Object,
	remediationAction configurationPolicyv1.RemediationAction) ocm.PolicyBuilder {
	customResource := ocm.PolicyBuilder{
		Definition: &policiesv1.Policy{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Policy",
				APIVersion: policiesv1.SchemeGroupVersion.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      policyName,
				Namespace: namespace,
			},
			Spec: policiesv1.PolicySpec{
				Disabled: false,
				PolicyTemplates: []*policiesv1.PolicyTemplate{
					{
						ObjectDefinition: runtime.RawExtension{
							Object: object,
						},
					},
				},
				RemediationAction: policiesv1.RemediationAction(remediationAction),
			},
		},
	}

	return customResource
}

// ApplyPolicyAndCreateAllComponents is used only apply already created policy and all the required components for
// applying that policy such as policyset, placementrule, placement binding, etc.
func ApplyPolicyAndCreateAllComponents(
	client *clients.Settings,
	policy ocm.PolicyBuilder,
	policySetName string,
	placementBindingName string,
	placementRuleName string,
	namespace string,
	clusters []string,
	clusterSelector metav1.LabelSelector,
) error {
	err := CreatePolicyAndWait(client, policy)

	if err != nil {
		return err
	}

	// Step 2 - Create the policy set
	glog.V(100).Info("creating the policyset")

	var nonEmptyStringList []policiesv1beta1.NonEmptyString
	nonEmptyStringList = append(nonEmptyStringList, policiesv1beta1.NonEmptyString(policy.Definition.Name))

	policySet := GetPolicySetDefinition(policySetName, nonEmptyStringList, namespace)

	_, err = policySet.Create()
	if err != nil {
		return err
	}

	// Step 3 - Get a placement field
	glog.V(100).Info("creating the generic placement fields")

	fields := GetPlacementFieldDefinition(clusters, clusterSelector)

	// Step 4 - Create the placement rule
	glog.V(100).Info("creating the placementrule")

	placementRule := GetPlacementRuleDefinition(placementRuleName, namespace, fields)

	_, err = placementRule.Create()
	if err != nil {
		return err
	}

	// Step 5 - Create the placement binding
	glog.V(100).Info("creating the placementbinding")

	placementBinding := GetPlacementBindingDefinition(
		placementBindingName,
		policySetName,
		placementRuleName,
		namespace,
	)

	_, err = placementBinding.Create()
	if err != nil {
		return err
	}

	return nil
}

// GetPlacementBindingDefinition is used to get a placement binding to use with a cgu.
func GetPlacementBindingDefinition(
	placementBindingName string,
	policySetName string,
	placementRuleName string,
	namespace string) ocm.PlacementBindingBuilder {
	customResource := ocm.PlacementBindingBuilder{
		Definition: &policiesv1.PlacementBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PlacementBinding",
				APIVersion: policiesv1.SchemeGroupVersion.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      placementBindingName,
				Namespace: namespace,
			},
			PlacementRef: policiesv1.PlacementSubject{
				Name:     placementRuleName,
				APIGroup: "apps.open-cluster-management.io",
				Kind:     "PlacementRule",
			},
			Subjects: []policiesv1.Subject{
				{
					Name:     policySetName,
					APIGroup: "policy.open-cluster-management.io",
					Kind:     "PolicySet",
				},
			}},
	}

	return customResource
}

// GetPlacementRuleDefinition is used to get a placement rule to use with a cgu.
func GetPlacementRuleDefinition(
	placementRuleName string,
	namespace string,
	placementFields placementrulev1.GenericPlacementFields) ocm.PlacementRuleBuilder {
	customResource := ocm.PlacementRuleBuilder{
		Definition: &placementrulev1.PlacementRule{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PlacementRule",
				APIVersion: placementrulev1.SchemeGroupVersion.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      placementRuleName,
				Namespace: namespace,
			},
			Spec: placementrulev1.PlacementRuleSpec{
				GenericPlacementFields: placementFields,
			}},
	}

	return customResource
}

// GetPolicySetDefinition is used to get a policy set for the provided policies.
func GetPolicySetDefinition(
	policySetName string,
	policyList []policiesv1beta1.NonEmptyString,
	namespace string) ocm.PolicySetBuilder {
	customResource := ocm.PolicySetBuilder{
		Definition: &policiesv1beta1.PolicySet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PolicySet",
				APIVersion: policiesv1beta1.GroupVersion.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      policySetName,
				Namespace: namespace,
			},
			Spec: policiesv1beta1.PolicySetSpec{
				Policies: policyList,
			},
		},
	}

	return customResource
}

// GetPlacementFieldDefinition is used to get a generic placement field for use with a placement rule.
func GetPlacementFieldDefinition(
	clusters []string,
	clusterSelector metav1.LabelSelector) placementrulev1.GenericPlacementFields {
	// Build the placement object we need in lieu of a flat string list
	clustersPlacementField := []placementrulev1.GenericClusterReference{}
	for _, cluster := range clusters {
		clustersPlacementField = append(clustersPlacementField, placementrulev1.GenericClusterReference{Name: cluster})
	}

	customResource := placementrulev1.GenericPlacementFields{
		Clusters:        clustersPlacementField,
		ClusterSelector: &clusterSelector,
	}

	return customResource
}

// GetCatsrcDefinition is used to get a catalog source definition for use in a policy.
func GetCatsrcDefinition(
	name string,
	namespace string,
	sourceType operatorsv1alpha1.SourceType,
	priority int,
	configMap string,
	address string,
	image string,
	displayName string) operatorsv1alpha1.CatalogSource {
	return operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CatalogSource",
			APIVersion: operatorsv1alpha1.CatalogSourceCRDAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: operatorsv1alpha1.CatalogSourceSpec{
			SourceType:  sourceType,
			Priority:    priority,
			ConfigMap:   configMap,
			Address:     address,
			Image:       image,
			DisplayName: displayName,
			Description: "a catalog source created by the talm tests",
			Publisher:   "eco-gosystem/tests/ranfunc/talm",
		},
	}
}

// EnableCgu enables cgu.
func EnableCgu(client *clients.Settings, cgu *cgu.CguBuilder) error {
	cgu.Definition.Spec.Enable = ptr.To[bool](true)
	_, err := cgu.Update(true)

	return err
}
