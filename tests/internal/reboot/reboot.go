package reboot

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// SoftRebootNode executes systemctl reboot on a node.
func SoftRebootNode(nodeName string) error {
	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
		LabelSelector: labels.SelectorFromSet(labels.Set{"k8s-app": GeneralConfig.MCOConfigDaemonName}).String(),
	}

	mcPodList, err := pod.List(APIClient, GeneralConfig.MCONamespace, listOptions)
	if err != nil {
		return err
	}

	cmdToExec := []string{"chroot", "/rootfs", "systemctl", "reboot"}

	glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, mcPodList[0].Definition.Name)
	buf, err := mcPodList[0].ExecCommand(cmdToExec)

	if err != nil {
		return fmt.Errorf("%w\n%s", err, buf.String())
	}

	return nil
}
