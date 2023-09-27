package tests

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/assisted"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpparams"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
)

var _ = Describe("ZTP Argocd clusters Tests", Ordered, Label("ztp-argocd-clusters"), func() {

	// These tests use the hub and spoke arch
	var clusterList []*clients.Settings

	BeforeAll(func() {
		// Initialize cluster list
		clusterList = gitopsztphelper.GetAllTestClients()
	})

	BeforeEach(func() {
		err := cluster.CheckClustersPresent(clusterList)
		if err != nil {
			Skip(fmt.Sprintf("error occurred validating required clusters are present: %s", err.Error()))
		}

		// Check for minimum ztp version
		By("Checking the ZTP version", func() {
			if !gitopsztphelper.IsVersionStringInRange(
				gitopsztphelper.ZtpVersion,
				"4.11",
				"",
			) {
				Skip(fmt.Sprintf(
					"unable to run test on ztp version '%s' as it is less than minimum '%s",
					gitopsztphelper.ZtpVersion,
					"4.11",
				))
			}
		})
	})

	Context("override the klusterlet addon configuration", Label("ztp-klusterlet"), func() {
		It("Should not have nmstateconfig cr when nodeNetwork section does not exist on siteConfig", func() {
			testGitPath := gitopsztphelper.JoinGitPaths(
				[]string{
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdClustersAppName].Path,
					"ztp-test/remove-nmstate",
				},
			)

			By("Checking if the git path exists", func() {
				if !gitopsztphelper.DoesGitPathExist(
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdClustersAppName].Repo,
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdClustersAppName].Branch,
					testGitPath+"/kustomization.yaml",
				) {
					Skip(fmt.Sprintf("git path '%s' could not be found", testGitPath))
				}
			})

			By("Check nmstateConfig cr exists", func() {
				nmStateConfigList, err := assisted.ListNmStateConfigsInAllNamespaces(gitopsztphelper.HubAPIClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(nmStateConfigList).ToNot(BeEmpty(), "No NMstateConfig found before test begins")
			})

			By("Reconfigure clusters app to set the ztp directory to the ztp-tests/remove-nmstate dir", func() {
				err := gitopsztphelper.SetGitDetailsInArgocd(
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdClustersAppName].Repo,
					gitopsztphelper.ArgocdApps[gitopsztpparams.ArgocdClustersAppName].Branch,
					testGitPath,
					gitopsztpparams.ArgocdClustersAppName,
					true,
					true,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			By("Check nmstateConfig CR is gone under spoke cluster NS on hub", func() {
				nmStateConfigList, err := assisted.ListNmStateConfigsInAllNamespaces(gitopsztphelper.HubAPIClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(nmStateConfigList).To(BeEmpty(), "NMstateconfig was found")
			})
		})
	})

})
