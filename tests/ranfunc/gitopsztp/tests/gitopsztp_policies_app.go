package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/gitopsztp/internal/gitopsztpparams"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncparams"
)

var _ = Describe("ZTP Argocd policies Tests", Label("ztp-argocd-policies"), func() {

	// These tests use the hub and spoke arch
	var (
		clusterList      []*clients.Settings
		defaultGitRepo   = ""
		defaultGitBranch = ""
		defaultGitPath   = ""
	)

	BeforeAll(func() {
		// Initialize cluster list
		clusterList = gitopsztphelper.GetAllTestClients()
	})

	BeforeEach(func() {
		// Check that the required clusters are present
		err := cluster.CheckClustersPresent(clusterList)
		if err != nil {
			Skip(fmt.Sprintf("error occurred validating required clusters are present: %s", err.Error()))
		}

		// Check for minimum ztp version
		By("Checking the ZTP version", func() {
			if !ranfunchelper.IsVersionStringInRange(
				ranfunchelper.ZtpVersion,
				"4.10",
				"",
			) {
				Skip(fmt.Sprintf(
					"unable to run test on ztp version '%s' as it is less than minimum '%s",
					ranfunchelper.ZtpVersion,
					"4.10",
				))
			}
		})

		// Collecting and storing default Git values of policies app before the test
		By("Collecting original Git values of policies app", func() {
			defaultGitRepo, defaultGitBranch, defaultGitPath, err = gitopsztphelper.GetGitDetailsFromArgocd(
				gitopsztpparams.ArgocdPoliciesAppName, ranfuncparams.OpenshiftGitops)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	AfterEach(func() {
		// Reset the policies app back to default after each test
		By("Resetting the policies app back to the original settings", func() {
			err := gitopsztphelper.SetGitDetailsInArgocd(
				defaultGitRepo, defaultGitBranch, defaultGitPath,
				gitopsztpparams.ArgocdPoliciesAppName,
				true,
				false,
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Apply and Validate custom source CRs on the du policies", Label("ztp-custom-source-crs"), func() {

		var (
			// customSourceCrPolicyName = "custom-source-cr-policy-config" // enforce policy
			testCrName       = "custom-source-cr"
			testNs           = "default"
			crServiceAccount = serviceaccount.NewBuilder(ranfuncinittools.SpokeAPIClient, testCrName, testNs)
		)

		It("verifies new CR kind that does not exist in ztp "+
			"container image can be created via custom source-cr", polarion.ID("61978"), func() {
			// The ztp test data is stored in a nested directory within the ztp repo
			testGitPath := gitopsztphelper.JoinGitPaths(
				[]string{
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdPoliciesAppName].Path,
					"ztp-test/custom-source-crs/new-cr",
				},
			)

			By("Checking if the git path exists", func() {
				if !gitopsztphelper.DoesGitPathExist(
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdPoliciesAppName].Repo,
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdPoliciesAppName].Branch,
					testGitPath+"/kustomization.yaml",
				) {
					Skip(fmt.Sprintf("git path '%s' could not be found", testGitPath))
				}
			})

			By("Checking Service Account custom resource does NOT exist on spoke", func() {
				SAExists := crServiceAccount.Exists()
				Expect(SAExists).To(BeFalse())
			})

			By("Updating the Argocd app", func() {
				// Update the Argo app to point to the new test kustomization
				err := gitopsztphelper.SetGitDetailsInArgocd(
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdPoliciesAppName].Repo,
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdPoliciesAppName].Branch,
					testGitPath,
					gitopsztpparams.ArgocdPoliciesAppName,
					true,
					true,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			By("Checking new Service Account custom resource created and applied to spoke", func() {
				newCrExists := crServiceAccount.Exists()
				Expect(newCrExists).To(BeTrue())
			})
		})

		AfterEach(func() {
			// Delete leftovers from the custom source-cr test
			By("Deleting created Service Account custom resource from spoke if exists", func() {
				err := crServiceAccount.Delete()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
