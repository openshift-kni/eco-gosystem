package reboot

import (
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"
)

// SoftRebootNode executes systemctl reboot on a node.
func SoftRebootNode(nodeName string) error {
	cmdToExec := []string{"chroot", "/rootfs", "systemctl", "reboot"}

	_, err := cmd.ExecCmd(cmdToExec, nodeName)

	if err != nil {
		return err
	}

	return nil
}
