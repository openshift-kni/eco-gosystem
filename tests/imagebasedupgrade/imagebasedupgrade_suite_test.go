package imagebasedupgrade_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/eco-gosystem/tests/imagebasedupgrade/tests"
)

func TestImageBasedUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ImageBasedUpgrade Suite")
}
