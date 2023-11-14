package argocdhelper

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openshift-kni/k8sreporter"
	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-kni/eco-goinfra/pkg/argocd"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/argocd/internal/argocdparams"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpparams"
)

// ztp related vars.
var (
	// test image.
	CnfTestImage = "quay.io/openshift-kni/cnf-tests:4.8"

	// ArgocdApps represents argocd git details.
	ArgocdApps = map[string]argocdparams.ArgocdGitDetails{}
	// ReporterNamespacesToDump tells to the reporter from where to collect logs.
	ReporterNamespacesToDump = map[string]string{
		argocdparams.ZtpTestNamespace: "",
	}

	// ReporterCRDsToDump tells to the reporter what CRs to dump.
	ReporterCRDsToDump = []k8sreporter.CRData{
		{Cr: &v1.PodList{}},
	}
)

// SetGitDetailsInArgocd is used to update the git repo, branch, and path in the Argocd app.
func SetGitDetailsInArgocd(gitRepo, gitBranch, gitPath, argocdApp string, waitForSync, syncMustBeValid bool) error {
	app, err := argocd.PullApplication(gitopsztpinittools.HubAPIClient, argocdApp, gitopsztpparams.OpenshiftGitops)
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
		err = WaitUntilArgocdChangeIsCompleted(argocdApp, syncMustBeValid, argocdparams.ArgocdChangeTimeout)
		if err != nil {
			return err
		}
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
		gitopsztpinittools.HubAPIClient,
		gitopsztpinittools.SpokeAPIClient,
	}
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

// WaitUntilArgocdChangeIsCompleted is used to wait until Argocd has updated its configuration.
func WaitUntilArgocdChangeIsCompleted(appName string, syncMustBeValid bool, timeout time.Duration) error {
	log.Println("Waiting for Argocd change to finish syncing")

	err := wait.PollImmediate(argocdparams.ArgocdChangeInterval, timeout, func() (bool, error) {
		log.Println("Checking if argo change is complete...")

		app, err := argocd.PullApplication(gitopsztpinittools.HubAPIClient, appName, gitopsztpparams.OpenshiftGitops)
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
	app, err := argocd.PullApplication(gitopsztpinittools.HubAPIClient, appName, namespace)
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
		for _, app := range argocdparams.ArgocdApps {
			repo, branch, dir, err := GetGitDetailsFromArgocd(app, gitopsztpparams.OpenshiftGitops)
			if err != nil {
				return err
			}

			// Save the git details to the map
			ArgocdApps[app] = argocdparams.ArgocdGitDetails{
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
		for _, app := range argocdparams.ArgocdApps {
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
