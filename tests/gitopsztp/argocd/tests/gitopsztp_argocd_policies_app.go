package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/serviceaccount"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/argocd/internal/argocdhelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/argocd/internal/argocdparams"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
)

var _ = Describe("ZTP Argocd policies Tests", Label("ztp-argocd-policies"), func() {

	// These tests use the hub and spoke arch
	var clusterList []*clients.Settings

	BeforeAll(func() {
		// Initialize cluster list
		clusterList = argocdhelper.GetAllTestClients()
	})

	BeforeEach(func() {
		// Check that the required clusters are present
		err := cluster.CheckClustersPresent(clusterList)
		if err != nil {
			Skip(fmt.Sprintf("error occurred validating required clusters are present: %s", err.Error()))
		}

		// Check for minimum ztp version
		By("Checking the ZTP version", func() {
			if !gitopsztphelper.IsVersionStringInRange(
				gitopsztphelper.ZtpVersion,
				"4.10",
				"",
			) {
				Skip(fmt.Sprintf(
					"unable to run test on ztp version '%s' as it is less than minimum '%s",
					gitopsztphelper.ZtpVersion,
					"4.10",
				))
			}
		})
	})

	Context("Apply and Validate custom source CRs on the du policies", Label("ztp-custom-source-crs"), func() {

		var (
			// customSourceCrPolicyName = "custom-source-cr-policy-config" // enforce policy
			testCrName = "custom-source-cr"
			testNs     = "default"
		)

		It("verifies new CR kind that does not exist in ztp "+
			"container image can be created via custom source-cr", polarion.ID("61978"), func() {
			// The ztp test data is stored in a nested directory within the ztp repo
			testGitPath := argocdhelper.JoinGitPaths(
				[]string{
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Path,
					"ztp-test/custom-source-crs/new-cr",
				},
			)

			By("Checking if the git path exists", func() {
				if !argocdhelper.DoesGitPathExist(
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Repo,
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Branch,
					testGitPath+"/kustomization.yaml",
				) {
					Skip(fmt.Sprintf("git path '%s' could not be found", testGitPath))
				}
			})

			By("Checking Service Account does NOT exist on spoke", func() {
				err := serviceaccount.NewBuilder(gitopsztpinittools.SpokeAPIClient, testCrName, testNs).Exists()
				Expect(err).ToNot(HaveOccurred())
				Expect(err).To(BeFalse())
			})

			By("Updating the Argocd app", func() {
				// Update the Argo app to point to the new test kustomization
				err := argocdhelper.SetGitDetailsInArgocd(
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Repo,
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Branch,
					testGitPath,
					argocdparams.ArgocdPoliciesAppName,
					true,
					true,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			/*
				By("Waiting for policy to be created and exist", func() {
					err := argocdhelper.WaitForPolicyToExist(
						customSourceCrPolicyName,
						argocdparams.ZtpTestNamespace,
						argocdparams.ArgocdChangeTimeout,
					)
					Expect(err).ToNot(HaveOccurred())
				})

				By("Waiting for the policy to be in the Compliant state", func() {
					err := argocdhelper.WaitForPolicyToHaveComplianceState(
						customSourceCrPolicyName,
						argocdparams.ZtpTestNamespace,
						policiesv1.Compliant,
						argocdparams.ArgocdChangeTimeout,
					)
					Expect(err).ToNot(HaveOccurred())
				})
			*/

			By("Checking new custom Service Account created and applied to spoke", func() {
				err := serviceaccount.NewBuilder(gitopsztpinittools.SpokeAPIClient, testCrName, testNs).Exists()
				Expect(err).ToNot(HaveOccurred())
				Expect(err).To(BeTrue())
			})
		})

		It("should validate the same source cr file name", polarion.ID("62260"), func() {
			// Check for minimum ztp version to run this test case
			By("Checking the ZTP version", func() {
				if !gitopsztphelper.IsVersionStringInRange(
					gitopsztphelper.ZtpVersion,
					"4.14",
					"",
				) {
					Skip(fmt.Sprintf(
						"unable to run test on ztp version '%s' as it is less than minimum '%s",
						gitopsztphelper.ZtpVersion,
						"4.14",
					))
				}
			})

			// The ztp test data is stored in a nested directory within the ztp repo
			testGitPath := argocdhelper.JoinGitPaths(
				[]string{
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Path,
					"ztp-test/custom-source-crs/replace-existing",
				},
			)

			By("Checking if the git path exists", func() {
				if !argocdhelper.DoesGitPathExist(
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Repo,
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Branch,
					testGitPath+"/kustomization.yaml",
				) {
					Skip(fmt.Sprintf("git path '%s' could not be found", testGitPath))
				}
			})

			By("Updating the Argocd app", func() {
				// Adding 30s sleep to avoid race condition before updating policies app within short time
				time.Sleep(30 * time.Second)
				// Update the Argo app to point to the new test kustomization
				err := argocdhelper.SetGitDetailsInArgocd(
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Repo,
					argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Branch,
					testGitPath,
					argocdparams.ArgocdPoliciesAppName,
					true,
					true,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			/*
				By("Waiting for policy to be created and exist", func() {
					err := argocdhelper.WaitForPolicyToExist(
						customSourceCrPolicyName,
						argocdparams.ZtpTestNamespace,
						argocdparams.ArgocdChangeTimeout,
					)
					Expect(err).ToNot(HaveOccurred())
				})

				By("Waiting for the policy to be in the Compliant state", func() {
					err := argocdhelper.WaitForPolicyToHaveComplianceState(
						customSourceCrPolicyName,
						argocdparams.ZtpTestNamespace,
						policiesv1.Compliant,
						argocdparams.ArgocdChangeTimeout,
					)
					Expect(err).ToNot(HaveOccurred())
				})
			*/

			By("Checking the custom namespace created and applied to spoke", func() {
				// Adding 20s sleep to avoid race condition before checking custom namespace existence
				time.Sleep(20 * time.Second)
				err := namespace.NewBuilder(gitopsztpinittools.SpokeAPIClient, testCrName).Exists()
				Expect(err).To(BeTrue())
			})
		})

		AfterEach(func() {
			/*
				// Deleting the policy created for custom source-cr test
				By("Deleting the policy if it exists", func() {
					err := argocdhelper.DeletePolicyAndWait(gitopsztpinittools.SpokeAPIClient, customSourceCrPolicyName,
						argocdparams.ZtpTestNamespace)
					Expect(err).ToNot(HaveOccurred())
				})
			*/

			// Delete leftovers from the custom source-crs test
			By("Deleting created Service Account from spoke if exists", func() {
				err := serviceaccount.NewBuilder(gitopsztpinittools.SpokeAPIClient, testCrName, testNs).Delete()
				Expect(err).ToNot(HaveOccurred())
			})

			// Delete leftovers from the same source cr file name test
			By("Deleting created custom namespace from spoke if it exists", func() {
				if namespace.NewBuilder(gitopsztpinittools.SpokeAPIClient, testCrName).Exists() {
					err := namespace.NewBuilder(gitopsztpinittools.SpokeAPIClient, testCrName).DeleteAndWait(3 * time.Minute)
					Expect(err).ToNot(HaveOccurred())
				}
			})
		})
	})

	AfterEach(func() {
		// Reset the policies app back to default after each test
		By("Resetting the policies app back to the original settings", func() {
			err := argocdhelper.SetGitDetailsInArgocd(
				argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Repo,
				argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Branch,
				argocdhelper.ArgocdApps[argocdparams.ArgocdPoliciesAppName].Path,
				argocdparams.ArgocdPoliciesAppName,
				true,
				false,
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
