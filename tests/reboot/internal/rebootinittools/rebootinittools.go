package rebootinittools

import (
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	"github.com/openshift-kni/eco-gosystem/tests/reboot/internal/rebootconfig"
)

var (
	// APIClient provides API access to cluster.
	APIClient *clients.Settings
	// RebootTestConfig provides access to tests configuration parameters.
	RebootTestConfig *rebootconfig.RebootConfig
)

// init loads all variables automatically when this package is imported. Once package is imported a user has full
// access to all vars within init function. It is recommended to import this package using dot import.
func init() {
	RebootTestConfig = rebootconfig.NewRebootConfig()
	APIClient = inittools.APIClient
}
