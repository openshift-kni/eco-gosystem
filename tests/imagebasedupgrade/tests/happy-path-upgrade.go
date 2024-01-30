package tests

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/ibuclusterinfo"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeinittools"
	"github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/internal/imagebasedupgradeparams"
	ibuvalidations "github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/validations"
)

// IbuCr is a dedicated var to use and act on it.
var IbuCr = lca.NewImageBasedUpgradeBuilder(
	imagebasedupgradeinittools.TargetSNOAPIClient, imagebasedupgradeparams.ImagebasedupgradeCrName)

var _ = Describe(
	"HappyPathUpgrade",
	Ordered,
	ContinueOnFailure,
	Label("HappyPathUpgrade"), func() {
		BeforeAll(func() {
			By("Generating seed image", func() {
				// Test Automation Code Implementation is to be done.
			})

			By("Saving pre upgrade cluster info", func() {
				err := ibuclusterinfo.SaveClusterInfo(&imagebasedupgradeparams.PreUpgradeClusterInfo)
				Expect(err).ToNot(HaveOccurred(), "Failed to save pre upgrade cluster info")
			})
		})

		AfterAll(func() {
			// Rollback to pre-upgrade state and reset PolicyGenTemplate.
		})

		It("End to end upgrade happy path", polarion.ID("68954"), Label("HappyPathUpgrade"), func() {

			By("Checking ImageBasedUpgrade CR exists with Idle stage in Target SNO", func() {
				crExists := IbuCr.Exists()
				Expect(crExists).To(BeTrue())
			})

			By("Updating ImageBasedUpgrade CR with Prep stage in Target SNO", func() {
				IbuCr.WithStage("Prep")
			})

			By("Updating ImageBasedUpgrade CR with Upgrade stage in Target SNO", func() {
				IbuCr.WithStage("Upgrade")
			})

			By(" Verifying target SNO cluster ACM registration post upgrade", func() {

			})

			By("Updating ImageBasedUpgrade CR with Idle stage in Target SNO", func() {
				IbuCr.WithStage("Idle")

			})
		})

		err := ibuclusterinfo.SaveClusterInfo(&imagebasedupgradeparams.PostUpgradeClusterInfo)
		Expect(err).ToNot(HaveOccurred(), "Failed to save post upgrade cluster info")

		ibuvalidations.PostUpgradeValidations()

	})
