package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmhelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
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

	Context("where all the canaries are successful", func() {
		// 47947
		It("should complete the CGU", polarion.ID("47947"), func() {

			By("creating the cgu and associated resources", func() {
				// cgu := talmhelper.GetCguDefinition(
				// 	talmhelper.CguName,
				// 	[]string{talmhelper.Spoke2Name, talmhelper.Spoke2Name},
				// 	[]string{talmhelper.Spoke2Name},
				// 	[]string{talmhelper.PolicyName},
				// 	talmhelper.Namespace, 1, 9)

				// namespace, err := namespace.Pull(gitopsztpinittools.HubAPIClient, talmhelper.TemporaryNamespaceName)
				// Expect(err).ToNot(HaveOccurred())

				// create policy and cgu
			})

			By("making sure the canary cluster (spoke2) starts first", func() {
				err := talmhelper.WaitForClusterInProgressInCgu(
					gitopsztpinittools.HubAPIClient,
					talmhelper.CguName,
					talmhelper.Spoke2Name,
					talmhelper.Namespace,
					2*talmparams.TalmDefaultReconcileTime,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			By("making sure the non-canary cluster (spoke1) has not started yet", func() {
				started, err := talmhelper.IsClusterStartedInCgu(
					gitopsztpinittools.HubAPIClient,
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
