package talm

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmhelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmparams"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configurationPolicyv1 "open-cluster-management.io/config-policy-controller/api/v1"
)

var _ = Describe("Talm Batching Tests", Ordered, Label("talmbatching"), func() {

	// These tests only use the hub and spoke1
	var clusterList []*clients.Settings

	BeforeAll(func() {
		// Initialize cluster list
		clusterList = talmhelper.GetAllTestClients()
	})

	BeforeEach(func() {
		// Check that the required clusters are present
		err := cluster.CheckClustersPresent(clusterList)
		if err != nil {
			Skip(fmt.Sprintf("error occurred validating required clusters are present: %s", err.Error()))
		}

		// Cleanup state to make it consistent
		for _, client := range clusterList {

			// Cleanup everything
			errList := talmhelper.CleanupTestResourcesOnClients(
				[]*clients.Settings{
					client,
				},
				talmhelper.CguName,
				talmhelper.PolicyName,
				talmhelper.Namespace,
				talmhelper.PlacementBindingName,
				talmhelper.PlacementRule,
				talmhelper.PolicySetName,
				talmhelper.CatalogSourceName)
			Expect(errList).To(BeEmpty())

			// Create namespace
			_, err := namespace.NewBuilder(client, talmhelper.Namespace).Create()
			Expect(err).ToNot(HaveOccurred())
		}
	})

	Context("with a single spoke that is missing", Label("talmmissingspoke"), func() {
		// 47949
		It("should report the missing spoke", func() {
			By("validating the talm version meets the test minimum", func() {
				// TALM 4.11 does not set any conditions for a non managed cluster error
				// We are unable to verify the state in 4.11 therefore we cannot run this test
				if !ranfunchelper.IsVersionStringInRange(
					talmhelper.TalmHubVersion,
					talmparams.TalmUpdatedConditionsVersion,
					"",
				) {
					Skip("test requires talm 4.12 or higher")
				}
			})
			By("creating the cgu", func() {
				cgu := talmhelper.GetCguDefinition(
					talmhelper.CguName,
					[]string{"non-existent-cluster"},
					[]string{},
					[]string{"non-existent-policy"},
					talmhelper.Namespace, 1, 1)

				err := talmhelper.CreateCguAndWait(
					ranfuncinittools.HubAPIClient,
					cgu,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			// Wait for the cgu condition to show the expected error message
			By("waiting for the error condition to match", func() {
				err := talmhelper.WaitForCguInCondition(
					ranfuncinittools.HubAPIClient,
					talmhelper.CguName,
					talmhelper.Namespace,
					"ClustersSelected",
					"Unable to select clusters: cluster non-existent-cluster is not a ManagedCluster",
					"",
					"",
					talmparams.TalmDefaultReconcileTime*3,
				)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("with a missing policy", Label("talmmissingpolicy"), func() {
		// 49755
		It("should report the missing policy", func() {
			By("create and enable a cgu with a managed policy that does not exist", func() {

				cgu := talmhelper.GetCguDefinition(
					talmhelper.CguName,
					[]string{talmhelper.Spoke1Name},
					[]string{},
					[]string{"non-existent-policy"},
					talmhelper.Namespace, 1, 1)

				err := talmhelper.CreateCguAndWait(
					ranfuncinittools.HubAPIClient,
					cgu,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			By("waiting for the cgu status to report the missing policy", func() {
				// Validation here depends on the TALM version

				conditionType := talmhelper.ValidatedType
				conditionMessage := "Missing managed policies: [non-existent-policy]"

				if !ranfunchelper.IsVersionStringInRange(
					talmhelper.TalmHubVersion,
					talmparams.TalmUpdatedConditionsVersion,
					"",
				) {
					conditionType = talmhelper.ReadyType
					conditionMessage = "The ClusterGroupUpgrade CR has: missing managed policies: [non-existent-policy]"
				}

				// This should immediately error out so we don't need a long timeout
				err := talmhelper.WaitForCguInCondition(
					ranfuncinittools.HubAPIClient,
					talmhelper.CguName,
					talmhelper.Namespace,
					conditionType,
					conditionMessage,
					"",
					"",
					1*time.Minute,
				)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		// 47952
		It("should report the failed spoke when one spoke in a batch times out", func() {
			namespaceBuilder := namespace.NewBuilder(ranfuncinittools.SpokeAPIClient, talmhelper.TemporaryNamespaceName)

			By("creating the temporary namespace on spoke1 only", func() {
				_, err := namespaceBuilder.Create()
				Expect(err).ToNot(HaveOccurred())
			})
			By("verifying the temporary namespace exists on spoke1", func() {
				result := namespaceBuilder.Exists()
				Expect(result).To(BeTrue())
			})
			By("verifying the temporary namespace does not exist on spoke2", func() {
				nsExist := namespace.NewBuilder(talmhelper.Spoke2APIClient, talmhelper.TemporaryNamespaceName).Exists()
				Expect(nsExist).To(BeFalse())
			})
			By("creating the cgu and associated resources", func() {
				// This test uses a max concurrency of 2
				// This way both spokes are in the same batch
				clusterGroupUpgrade := talmhelper.GetCguDefinition(
					talmhelper.CguName,
					[]string{
						talmhelper.Spoke1Name,
						talmhelper.Spoke2Name,
					},
					[]string{},
					[]string{
						talmhelper.PolicyName,
					},
					talmhelper.Namespace,
					2,
					9,
				)
				clusterGroupUpgrade.Definition.Spec.Enable = talmhelper.BoolAddr(false)

				catsrc := talmhelper.GetCatsrcDefinition(
					talmhelper.CatalogSourceName,
					talmhelper.TemporaryNamespaceName,
					operatorsv1alpha1.SourceTypeInternal,
					1,
					"",
					"",
					"",
					talmhelper.CatalogSourceName,
				)

				err := talmhelper.CreatePolicyAndCgu(
					ranfuncinittools.HubAPIClient,
					catsrc.Definition,
					configurationPolicyv1.MustHave,
					configurationPolicyv1.Inform,
					talmhelper.PolicyName,
					talmhelper.PolicySetName,
					talmhelper.PlacementBindingName,
					talmhelper.PlacementRule,
					talmhelper.Namespace,
					metav1.LabelSelector{},
					clusterGroupUpgrade,
				)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for the system to settle", func() {
					time.Sleep(talmparams.TalmSystemStablizationTime)
				})

				By("enabling the CGU", func() {
					clusterGroupUpgrade, err := cgu.Pull(ranfuncinittools.HubAPIClient, talmhelper.CguName, talmhelper.Namespace)
					Expect(err).ToNot(HaveOccurred())

					err = talmhelper.EnableCgu(ranfuncinittools.HubAPIClient, clusterGroupUpgrade)
					Expect(err).ToNot(HaveOccurred())
				})

				By("waiting for the cgu to timeout", func() {
					err := talmhelper.WaitForCguToTimeout(talmhelper.CguName, talmhelper.Namespace, 16*time.Minute)
					Expect(err).ToNot(HaveOccurred())
				})

				By("validating that the policy was successful on spoke2", func() {
					catSrcExists := olm.NewCatalogSourceBuilder(talmhelper.Spoke2APIClient,
						talmhelper.CatalogSourceName, talmhelper.TemporaryNamespaceName).Exists()
					Expect(catSrcExists).To(BeTrue())
				})

				By("validating that the policy was not successful on spoke1", func() {
					catSrcExists := olm.NewCatalogSourceBuilder(ranfuncinittools.SpokeAPIClient,
						talmhelper.CatalogSourceName, talmhelper.TemporaryNamespaceName).Exists()
					err = talmhelper.FilterMissingResourceErrors(err)
					Expect(err).ToNot(HaveOccurred())
					Expect(catSrcExists).To(BeFalse())
				})
			})
		})

	})
})
