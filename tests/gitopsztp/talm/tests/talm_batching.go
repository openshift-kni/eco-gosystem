package talm

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmhelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
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
				if !gitopsztphelper.IsVersionStringInRange(
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
					gitopsztpinittools.HubAPIClient,
					cgu,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			// Wait for the cgu condition to show the expected error message
			By("waiting for the error condition to match", func() {
				err := talmhelper.WaitForCguInCondition(
					gitopsztpinittools.HubAPIClient,
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
})
