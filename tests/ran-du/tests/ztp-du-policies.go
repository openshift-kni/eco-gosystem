package ran_du_system_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	. "github.com/openshift-kni/eco-gosystem/tests/ran-du/internal/randuinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ran-du/internal/randuparams"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe(
	"ZTPPoliciesCompliance",
	Label("ZTPPoliciesCompliance"),
	Ordered,
	func() {
		var (
			clusterName string
			err         error
		)
		BeforeAll(func() {
			By("Fetching Cluster name")
			clusterName, err = cluster.GetClusterName(randuparams.KubeEnvKey)
			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster name")
		})

		It("All policies compliant", polarion.ID("OCP-48465"),
			func() {
				By("Retrieve policy list")
				policies, err := ocm.ListPoliciesInAllNamespaces(
					APIClient,
					client.ListOptions{Namespace: clusterName},
				)
				Expect(err).NotTo(HaveOccurred(), "Failed to get policy list")

				By("Checking for NonCompliant policies")
				nonCompliantPolicies := make([]string, 0)
				for _, policy := range policies {
					if policy.Definition.Status.ComplianceState == policiesv1.NonCompliant {
						nonCompliantPolicies = append(nonCompliantPolicies, policy.Definition.Name)
					}
				}
				Expect(nonCompliantPolicies).To(
					BeEmpty(),
					fmt.Sprintf("NonCompliant policies found: %s", strings.Join(nonCompliantPolicies, ",")),
				)
			},
		)
	},
)
