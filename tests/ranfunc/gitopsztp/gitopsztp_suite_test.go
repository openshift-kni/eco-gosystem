package gitopsztp

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/reporter"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/gitopsztp/internal/gitopsztpparams"

	_ "github.com/openshift-kni/eco-gosystem/tests/ranfunc/gitopsztp/tests"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"

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
	err := ranfunchelper.InitializeClients()
	Expect(err).ToNot(HaveOccurred())

	namespace := namespace.NewBuilder(ranfuncinittools.HubAPIClient, gitopsztpparams.ZtpTestNamespace)

	// Delete and re-create the namespace to start with a clean state
	if namespace.Exists() {
		err = namespace.DeleteAndWait(gitopsztpparams.DefaultTimeout)
		Expect(err).ToNot(HaveOccurred())
	}

	_, err = namespace.Create()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Restore the original Argocd configuration after the tests are completed
	err := gitopsztphelper.ResetArgocdGitDetails()
	Expect(err).ToNot(HaveOccurred())

	// Delete the ztp namespace
	err = namespace.NewBuilder(ranfuncinittools.HubAPIClient, gitopsztpparams.ZtpTestNamespace).Delete()
	Expect(err).ToNot(HaveOccurred())
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), GeneralConfig.GetDumpFailedTestReportLocation(currentFile),
		GeneralConfig.ReportsDirAbsPath, gitopsztphelper.ReporterNamespacesToDump,
		gitopsztphelper.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})
