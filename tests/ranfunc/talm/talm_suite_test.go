package talm

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/reporter"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmhelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmparams"
	_ "github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/tests"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestTalm(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	// Stop ginkgo complaining about slow tests

	RunSpecs(t, "TALM Suite", reporterConfig)
}

var _ = BeforeSuite(func() {
	err := ranfunchelper.InitializeClients()
	Expect(err).ToNot(HaveOccurred())

	// Make sure TALM is present
	err = talmhelper.VerifyTalmIsInstalled()
	Expect(err).ToNot(HaveOccurred())

	// Delete the namespace before creating it to ensure it is in a consistent blank state
	err = talmhelper.DeleteTalmTestNamespace(true)
	Expect(err).ToNot(HaveOccurred())
	err = talmhelper.CreateTalmTestNamespace()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	// Deleting the namespace after the suite finishes ensures all the CGUs created are deleted
	err := talmhelper.DeleteTalmTestNamespace(false)
	Expect(err).ToNot(HaveOccurred())
})

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), GeneralConfig.GetDumpFailedTestReportLocation(currentFile),
		GeneralConfig.ReportsDirAbsPath, talmparams.ReporterNamespacesToDump,
		talmparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})
