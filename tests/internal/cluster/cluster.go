package cluster

import (
	"errors"
	"log"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
)

// CheckClustersPresent can be used to check for the presence of specific clusters.
func CheckClustersPresent(clients []*clients.Settings) error {
	// Log the cluster list
	log.Println(clients)

	for _, client := range clients {
		if client == nil {
			return errors.New("provided nil client in cluster list")
		}
	}

	return nil
}
