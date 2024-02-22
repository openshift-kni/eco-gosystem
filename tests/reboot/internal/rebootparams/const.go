package rebootparams

const (
	// Label is used for 'reboot' test cases selection.
	Label = "reboot"

	// LabelValidateReboot is used to select tests that reboot cluster.
	LabelValidateReboot = "validate_reboots"

	// RebootLogLevel configures logging level for reboot related tests.
	RebootLogLevel = 90
)

// BMCDetails structure to hold BMC details.
type BMCDetails struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	BMCAddress string `json:"bmc"`
}

// NodesBMCMap holds info about BMC connection for a specific node.
type NodesBMCMap map[string]BMCDetails
