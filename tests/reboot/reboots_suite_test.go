package spk_system_test

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/polarion"
	"github.com/openshift-kni/eco-goinfra/pkg/reporter"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"

	"github.com/openshift-kni/eco-gosystem/tests/reboot/internal/rebootparams"
	_ "github.com/openshift-kni/eco-gosystem/tests/reboot/tests"
)

var _, currentFile, _, _ = runtime.Caller(0)

func TestClusterReboot(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "SystemTests Reboot Suite", Label(rebootparams.Labels...), reporterConfig)
}

var _ = JustAfterEach(func() {
	reporter.ReportIfFailed(
		CurrentSpecReport(), GeneralConfig.GetDumpFailedTestReportLocation(currentFile), GeneralConfig.ReportsDirAbsPath,
		rebootparams.ReporterNamespacesToDump, rebootparams.ReporterCRDsToDump, clients.SetScheme)
})

var _ = ReportAfterSuite("", func(report Report) {
	polarion.CreateReport(
		report, GeneralConfig.GetPolarionReportPath(), GeneralConfig.PolarionTCPrefix)
})
