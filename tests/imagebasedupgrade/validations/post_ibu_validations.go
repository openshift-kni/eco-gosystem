package ibuvalidations

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeparams"
)

// PostUpgradeValidations is a dedicated func to run post upgrade test validations.
func PostUpgradeValidations() {
	Describe(
		"PostIBUValidations",
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
		})
}
