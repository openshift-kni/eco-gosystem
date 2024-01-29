package talm

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmhelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmparams"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configurationPolicyv1 "open-cluster-management.io/config-policy-controller/api/v1"
)

var _ = Describe("Talm Canary Tests", Ordered, Label("talmcanary"), func() {

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

		glog.V(100).Info("gitopsztpinittools.SpokeAPIClient.Name", ranfuncinittools.SpokeAPIClient.Name)
		// Cleanup state to make it consistent
		for _, client := range clusterList {

			fmt.Printf("working on cluster : %s", client.Name)
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

			glog.V(100).Info("creating namespace")
			// Create namespace
			_, err := namespace.NewBuilder(client, talmhelper.Namespace).Create()
			Expect(err).ToNot(HaveOccurred())
		}

		// Cleanup the temporary namespace
		err = talmhelper.CleanupNamespace(clusterList, talmhelper.TemporaryNamespaceName)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		// Cleanup everything
		errList := talmhelper.CleanupTestResourcesOnClients(
			clusterList,
			talmhelper.CguName,
			talmhelper.PolicyName,
			talmhelper.Namespace,
			talmhelper.PlacementBindingName,
			talmhelper.PlacementRule,
			talmhelper.PolicySetName,
			talmhelper.CatalogSourceName)
		Expect(errList).To(BeEmpty())

		// Cleanup the temporary namespace
		err := talmhelper.CleanupNamespace(clusterList, talmhelper.TemporaryNamespaceName)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("where first canary fails", func() {
		// 47954
		It("should stop the CGU", func() {
			By("verifying the temporary namespace does not exist", func() {
				namespaceExists := namespace.NewBuilder(ranfuncinittools.SpokeAPIClient,
					talmhelper.TemporaryNamespaceName).Exists()
				Expect(namespaceExists).To(BeFalse())

				namespaceExists = namespace.NewBuilder(talmhelper.Spoke2APIClient,
					talmhelper.TemporaryNamespaceName).Exists()
				Expect(namespaceExists).To(BeFalse())

			})

			By("creating the cgu and associated resources", func() {
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

				clusterGroupUpgrade := talmhelper.GetCguDefinition(
					talmhelper.CguName,
					[]string{talmhelper.Spoke1Name, talmhelper.Spoke2Name},
					[]string{talmhelper.Spoke2Name},
					[]string{talmhelper.PolicyName},
					talmhelper.Namespace, 1, 9)

				clusterGroupUpgrade.Definition.Spec.Enable = talmhelper.BoolAddr(false)

				err := talmhelper.CreatePolicyAndCgu(
					ranfuncinittools.HubAPIClient,
					&catsrc,
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

					err = talmhelper.EnableCgu(
						ranfuncinittools.HubAPIClient,
						clusterGroupUpgrade,
					)
					Expect(err).ToNot(HaveOccurred())
				})

				By("making sure the canary cluster (spoke2) starts first", func() {
					err := talmhelper.WaitForClusterInProgressInCgu(
						ranfuncinittools.HubAPIClient,
						talmhelper.CguName,
						talmhelper.Spoke2Name,
						talmhelper.Namespace,
						3*talmparams.TalmDefaultReconcileTime,
					)
					Expect(err).ToNot(HaveOccurred())
				})

				By("making sure the non-canary cluster (spoke1) has not started yet", func() {
					started, err := talmhelper.IsClusterStartedInCgu(
						ranfuncinittools.HubAPIClient,
						talmhelper.CguName,
						talmhelper.Spoke1Name,
						talmhelper.Namespace,
					)
					Expect(err).ToNot(HaveOccurred())
					Expect(started).To(BeFalse())
				})

				By("validating that the timeout was due to canary failure", func() {
					// Validation here depends on the TALM version

					conditionType := talmhelper.SucceededType
					conditionMessage := "Policy remediation took too long on canary clusters"

					if !ranfunchelper.IsVersionStringInRange(
						talmhelper.TalmHubVersion,
						talmparams.TalmUpdatedConditionsVersion,
						"",
					) {
						conditionType = talmhelper.ReadyType
						conditionMessage = talmhelper.Talm411TimeoutMessage
					}

					err := talmhelper.WaitForCguInCondition(
						ranfuncinittools.HubAPIClient,
						talmhelper.CguName,
						talmhelper.Namespace,
						conditionType,
						conditionMessage,
						"",
						"",
						11*time.Minute,
					)
					Expect(err).ToNot(HaveOccurred())
				})

			})

		})
	})

	Context("where all the canaries are successful", func() {

		It("should complete the CGU", polarion.ID("47947"), func() {

			By("creating the cgu and associated resources", func() {
				cgu := talmhelper.GetCguDefinition(
					talmhelper.CguName,
					[]string{talmhelper.Spoke2Name, talmhelper.Spoke2Name},
					[]string{talmhelper.Spoke2Name},
					[]string{talmhelper.PolicyName},
					talmhelper.Namespace, 1, 9)

				namespace, err := namespace.Pull(ranfuncinittools.HubAPIClient, talmhelper.TemporaryNamespaceName)
				Expect(err).ToNot(HaveOccurred())

				err = talmhelper.CreatePolicyAndCgu(
					ranfuncinittools.HubAPIClient,
					namespace.Definition,
					configurationPolicyv1.MustHave,
					configurationPolicyv1.Inform,
					talmhelper.PolicyName,
					talmhelper.PolicySetName,
					talmhelper.PlacementBindingName,
					talmhelper.PlacementRule,
					talmhelper.Namespace,
					metav1.LabelSelector{},
					cgu,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			By("making sure the canary cluster (spoke2) starts first", func() {
				err := talmhelper.WaitForClusterInProgressInCgu(
					ranfuncinittools.HubAPIClient,
					talmhelper.CguName,
					talmhelper.Spoke2Name,
					talmhelper.Namespace,
					2*talmparams.TalmDefaultReconcileTime,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			By("making sure the non-canary cluster (spoke1) has not started yet", func() {
				started, err := talmhelper.IsClusterStartedInCgu(
					ranfuncinittools.HubAPIClient,
					talmhelper.CguName,
					talmhelper.Spoke1Name,
					talmhelper.Namespace,
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(started).To(BeFalse())
			})

			By("waiting for the cgu to finish successfully", func() {
				err := talmhelper.WaitForCguToFinishSuccessfully(talmhelper.CguName, talmhelper.Namespace, 10*time.Minute)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
