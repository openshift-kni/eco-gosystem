package argocd

import (
	"log"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/reporter"
	"github.com/openshift-kni/eco-gosystem/tests/argocd/internal/argocdhelper"
	"github.com/openshift-kni/eco-gosystem/tests/argocd/internal/argocdparams"
	_ "github.com/openshift-kni/eco-gosystem/tests/argocd/tests"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	"github.com/openshift-kni/eco-gosystem/tests/talm/talmparams"

	. "github.com/onsi/gomega"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestArgocd(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	// Stop ginkgo complaining about slow tests

	RunSpecs(t, "Argocd Suite", reporterConfig)
}

var _ = BeforeSuite(func() {
	// API clients must be initialized first since they will be used later
	err := InitializeClients()
	Expect(err).ToNot(HaveOccurred())

	namespace := namespace.NewBuilder(argocdhelper.HubAPIClient, argocdparams.ZtpTestNamespace)

	// Delete and re-create the namespace to start with a clean state
	if namespace.Exists() {
		err = namespace.DeleteAndWait(argocdparams.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred())
	}

	_, err = namespace.Create()
	Expect(err).ToNot(HaveOccurred())

	// create a privileged pod to run commands on nodes
	glog.V(100).Infof("Setup initiated: creating a privileged pod")

	_, err = argocdhelper.CreatePrivilegedPods("")
	Expect(err).ToNot(HaveOccurred())

})

var _ = AfterSuite(func() {
	// Restore the original Argocd configuration after the tests are completed
	err := argocdhelper.ResetArgocdGitDetails()
	Expect(err).ToNot(HaveOccurred())

	// Delete the ztp namespace
	err = namespace.NewBuilder(argocdhelper.HubAPIClient, argocdparams.ZtpTestNamespace).Delete()
	Expect(err).ToNot(HaveOccurred())

	// delete the privileged pod
	glog.V(100).Infof("Teardown initiated: deleting privileged pod")
	err = namespace.NewBuilder(argocdhelper.HubAPIClient, argocdparams.PrivPodNamespace).Delete()
	Expect(err).ToNot(HaveOccurred())
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), GeneralConfig.GetDumpFailedTestReportLocation(currentFile),
		GeneralConfig.ReportsDirAbsPath, argocdhelper.ReporterNamespacesToDump,
		argocdhelper.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})

// InitializeClients initializes hub & spoke clients.
func InitializeClients() error {
	if os.Getenv(argocdparams.HubKubeEnvKey) != "" {
		var err error
		// Define all the hub information
		argocdhelper.HubAPIClient, err = argocdhelper.DefineAPIClient(argocdparams.HubKubeEnvKey)
		if err != nil {
			return err
		}

		argocdhelper.HubName, err = cluster.GetClusterName(argocdparams.HubKubeEnvKey)
		if err != nil {
			return err
		}

		ocpVersion, err := cluster.GetClusterVersion(argocdhelper.HubAPIClient)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has OCP version '%s'\n", argocdhelper.HubName, ocpVersion)

		argocdhelper.AcmVersion, err = argocdhelper.GetOperatorVersionFromCSV(
			argocdhelper.HubAPIClient,
			argocdparams.AcmOperatorName,
			argocdparams.AcmOperatorNamespace,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has ACM version '%s'\n", argocdhelper.HubName, argocdhelper.AcmVersion)

		argocdhelper.ZtpVersion, err = argocdhelper.GetZtpVersionFromArgocd(
			argocdhelper.HubAPIClient,
			argocdparams.OpenshiftGitopsRepoServer,
			argocdparams.OpenshiftGitops,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has ZTP version '%s'\n", argocdhelper.HubName, argocdhelper.ZtpVersion)

		argocdhelper.TalmVersion, err = argocdhelper.GetOperatorVersionFromCSV(
			argocdhelper.HubAPIClient,
			talmparams.OperatorHubTalmNamespace,
			talmparams.OpenshiftOperatorNamespace,
		)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has TALM version '%s'\n", argocdhelper.HubName, argocdhelper.TalmVersion)
	}

	// Spoke is the default kubeconfig
	if os.Getenv(argocdparams.SpokeKubeEnvKey) != "" {
		var err error
		// Define all the spoke information
		argocdhelper.SpokeAPIClient, err = argocdhelper.DefineAPIClient(argocdparams.SpokeKubeEnvKey)
		if err != nil {
			return err
		}

		argocdhelper.SpokeName, err = cluster.GetClusterName(argocdparams.SpokeKubeEnvKey)
		if err != nil {
			return err
		}

		ocpVersion, err := cluster.GetClusterVersion(argocdhelper.SpokeAPIClient)
		if err != nil {
			return err
		}

		log.Printf("cluster '%s' has OCP version '%s'\n", argocdhelper.SpokeName, ocpVersion)
	}

	return nil
}
