package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmhelper"
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
})
