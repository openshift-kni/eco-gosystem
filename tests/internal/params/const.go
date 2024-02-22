package systemtestsparams

import (
	"encoding/json"
	"fmt"
	"log"
)

const (
	// Label represents label that can be used for test cases selection.
	Label = "system"
)

// BMCDetails structure to hold BMC details.
type BMCDetails struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	BMCAddress string `json:"bmc"`
}

// NodesBMCMap holds info about BMC connection for a specific node.
type NodesBMCMap map[string]BMCDetails

// Decode - method for envconfig package to parse JSON encoded environment variables.
func (nad *NodesBMCMap) Decode(value string) error {
	nodesAuthMap := new(map[string]BMCDetails)

	err := json.Unmarshal([]byte(value), nodesAuthMap)

	if err != nil {
		log.Printf("Error to parse data %v", err)

		return fmt.Errorf("invalid map json: %w", err)
	}

	*nad = *nodesAuthMap

	return nil
}
