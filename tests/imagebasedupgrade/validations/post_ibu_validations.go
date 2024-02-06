package ibuvalidations

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeinittools"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeparams"
)

// PostUpgradeValidations is a dedicated func to run post upgrade test validations.
func PostUpgradeValidations() {
	Describe(
		"PostIBUValidations",
		Ordered,
		Label("PostIBUValidations"), func() {
			It("Validate Cluster version", polarion.ID("99998"), Label("ValidateClusterVersion"), func() {
				By("Validate upgraded cluster version reports correct version", func() {
					Expect(imagebasedupgradeparams.PreUpgradeClusterInfo.Version).
						ToNot(Equal(imagebasedupgradeparams.PostUpgradeClusterInfo.Version), "Cluster version hasn't changed")
				})
			})

			It("Validate Cluster ID", polarion.ID("99999"), Label("ValidateClusterID"), func() {
				By("Validate cluster ID remains the same", func() {
					Expect(imagebasedupgradeparams.PreUpgradeClusterInfo.ID).
						To(Equal(imagebasedupgradeparams.PostUpgradeClusterInfo.ID), "Cluster ID has changed")
				})
			})

			It("Validate CSV versions", polarion.ID("99999"), Label("ValidateCSV"), func() {
				By("Validate CSVs were upgraded", func() {
					for _, preUpgradeOperator := range imagebasedupgradeparams.PreUpgradeClusterInfo.Operators {
						for _, postUpgradeOperator := range imagebasedupgradeparams.PostUpgradeClusterInfo.Operators {
							Expect(preUpgradeOperator).ToNot(Equal(postUpgradeOperator), "Operator %s was not upgraded", preUpgradeOperator)
						}
					}
				})
			})

			It("Validate no pods using seed name", polarion.ID("99999"), Label("ValidatePodsSeedName"), func() {
				By("Validate no pods are using seed's name", func() {
					podList, err := pod.ListInAllNamespaces(TargetSNOAPIClient, v1.ListOptions{})
					Expect(err).ToNot(HaveOccurred(), "Failed to list pods")

					for _, pod := range podList {
						Expect(pod.Object.Name).ToNot(ContainSubstring(imagebasedupgradeparams.PreUpgradeClusterInfo.NodeName),
							"Pod %s is using seed's name", pod.Object.Name)
					}
				})
			})
		})
}
