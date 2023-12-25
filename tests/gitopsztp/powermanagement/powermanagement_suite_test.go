package powermanagement

import (
	"runtime"
	"testing"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/powermanagement/internal/powermanagementparams"
	_ "github.com/openshift-kni/eco-gosystem/tests/gitopsztp/powermanagement/tests"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
)

var _, currentFile, _, _ = runtime.Caller(0)

const (
	timeout = 10 * time.Minute
)

func TestPowerSave(t *testing.T) {
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = GeneralConfig.GetJunitReportPath(currentFile)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Power Management Test Suite", reporterConfig)
}

var _ = BeforeSuite(func() {
	// Cleanup and create test namespace
	namespace := namespace.NewBuilder(gitopsztpinittools.HubAPIClient, powermanagementparams.NamespaceTesting)
	if namespace.Exists() {
		glog.V(100).Infof("Deleting test namespace", powermanagementparams.NamespaceTesting)
		_ = namespace.DeleteAndWait(5 * time.Minute)
	}
	glog.V(100).Infof("Creating test namespace", powermanagementparams.NamespaceTesting)
	_, err := namespace.Create()
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	namespace := namespace.NewBuilder(gitopsztpinittools.HubAPIClient, powermanagementparams.NamespaceTesting)
	if namespace.Exists() {
		glog.V(100).Infof("Deleting test namespace", powermanagementparams.NamespaceTesting)
		err := namespace.DeleteAndWait(timeout)
		Expect(err).ToNot(HaveOccurred())
	}
})
