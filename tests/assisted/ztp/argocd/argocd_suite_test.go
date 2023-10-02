package argocd

import (
	"runtime"
	"testing"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/reporter"
	"github.com/openshift-kni/eco-gosystem/tests/assisted/ztp/argocd/internal/argocdhelper"
	"github.com/openshift-kni/eco-gosystem/tests/assisted/ztp/argocd/internal/argocdparams"
	_ "github.com/openshift-kni/eco-gosystem/tests/assisted/ztp/argocd/tests"
	"github.com/openshift-kni/eco-gosystem/tests/assisted/ztp/internal/ztphelper"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"

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
	err := ztphelper.InitializeClients()
	Expect(err).ToNot(HaveOccurred())

	namespace := namespace.NewBuilder(ztphelper.HubAPIClient, argocdparams.ZtpTestNamespace)

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
	err = namespace.NewBuilder(ztphelper.HubAPIClient, argocdparams.ZtpTestNamespace).Delete()
	Expect(err).ToNot(HaveOccurred())

	// delete the privileged pod
	glog.V(100).Infof("Teardown initiated: deleting privileged pod")
	err = namespace.NewBuilder(ztphelper.HubAPIClient, argocdparams.PrivPodNamespace).Delete()
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
