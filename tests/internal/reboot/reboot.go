package reboot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bmc-toolbox/bmclib/v2"
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"
	. "github.com/openshift-kni/eco-gosystem/tests/internal/inittools"
	systemtestsparams "github.com/openshift-kni/eco-gosystem/tests/internal/params"
	systemtestsscc "github.com/openshift-kni/eco-gosystem/tests/internal/scc"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
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

// powerCycleNode is used to inovke 'power cycle' command over BMC session.
func powerCycleNode(
	waitingGroup *sync.WaitGroup,
	nodeName string,
	client *bmclib.Client,
	lock *sync.Mutex,
	resMap map[string][]string) {
	glog.V(90).Infof(
		fmt.Sprintf("Starting go routine for %s", nodeName))

	defer waitingGroup.Done()

	glog.V(90).Infof(
		fmt.Sprintf("[%s] Setting timeout for context", nodeName))

	bmcCtx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	glog.V(90).Infof(
		fmt.Sprintf("[%s] Starting BMC session", nodeName))

	err := client.Open(bmcCtx)

	if err != nil {
		glog.V(90).Infof("Failed to open BMC session to %s", nodeName)
		lock.Lock()
		if value, ok := resMap["error"]; !ok {
			resMap["error"] = []string{fmt.Sprintf("Failed to open BMC session to %s", nodeName)}
		} else {
			value = append(value, fmt.Sprintf("Failed to open BMC session to %s", nodeName))
			resMap["error"] = value
		}
		lock.Unlock()

		return
	}

	defer client.Close(bmcCtx)

	glog.V(90).Infof(fmt.Sprintf("Checking power state on %s", nodeName))

	err = wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 5*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			if _, err := client.SetPowerState(bmcCtx, "cycle"); err != nil {
				glog.V(90).Infof(
					fmt.Sprintf("Failed to power cycle %s -> %v", nodeName, err))

				return false, err
			}

			glog.V(90).Infof(
				fmt.Sprintf("Successfully powered cycle %s", nodeName))

			return true, nil
		})

	if err != nil {
		glog.V(90).Infof("Failed to reboot node %s due to %v", nodeName, err)
		lock.Lock()
		if value, ok := resMap["error"]; !ok {
			resMap["error"] = []string{fmt.Sprintf("Failed to reboot node %s due to %v", nodeName, err)}
		} else {
			value = append(value, fmt.Sprintf("Failed to reboot node %s due to %v", nodeName, err))
			resMap["error"] = value
		}
		lock.Unlock()

		return
	}
}

// HardRebootNodeBMC reboots node(s) via BMC
// returns false if there were issues power cycling node(s).
func HardRebootNodeBMC(nodesBMCMap systemtestsparams.NodesBMCMap) error {
	clientOpts := []bmclib.Option{}

	glog.V(90).Infof(fmt.Sprintf("BMC options %v", clientOpts))

	glog.V(90).Infof(
		fmt.Sprintf("NodesCredentialsMap:\n\t%#v", nodesBMCMap))

	var bmcMap = make(map[string]*bmclib.Client)

	for node, auth := range nodesBMCMap {
		glog.V(90).Infof(
			fmt.Sprintf("Creating BMC client for node %s", node))
		glog.V(90).Infof(
			fmt.Sprintf("BMC Auth %#v", auth))

		bmcClient := bmclib.NewClient(auth.BMCAddress, auth.Username, auth.Password, clientOpts...)
		bmcMap[node] = bmcClient
	}

	var (
		waitGroup sync.WaitGroup
		mapMutex  sync.Mutex
		resultMap = make(map[string][]string)
	)

	for node, client := range bmcMap {
		waitGroup.Add(1)

		go powerCycleNode(&waitGroup, node, client, &mapMutex, resultMap)
	}

	glog.V(90).Infof("Wait for all reboots to finish")
	waitGroup.Wait()

	glog.V(90).Infof("Finished waiting for go routines to finish")

	if len(resultMap) != 0 {
		glog.V(90).Infof("There were errors power cycling nodes")

		for _, msg := range resultMap["error"] {
			glog.V(90).Infof("\tError: %v", msg)
		}

		return fmt.Errorf("there were errors power cycling nodes")
	}

	return nil
}

// HardRebootNode executes ipmitool chassis power cycle on a node.
func HardRebootNode(nodeName string, nsName string) error {
	err := systemtestsscc.AddPrivilegedSCCtoDefaultSA(nsName)
	if err != nil {
		return err
	}

	deployContainer := pod.NewContainerBuilder(systemtestsparams.HardRebootDeploymentName,
		GeneralConfig.IpmiToolImage, []string{"sleep", "86400"})

	trueVar := true
	deployContainer = deployContainer.WithSecurityContext(&v1.SecurityContext{Privileged: &trueVar})

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		return err
	}

	createDeploy := deployment.NewBuilder(APIClient, systemtestsparams.HardRebootDeploymentName, nsName,
		map[string]string{"test": "hardreboot"}, deployContainerCfg)
	createDeploy = createDeploy.WithNodeSelector(map[string]string{"kubernetes.io/hostname": nodeName})

	_, err = createDeploy.CreateAndWaitUntilReady(300 * time.Second)
	if err != nil {
		return err
	}

	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
		LabelSelector: labels.SelectorFromSet(labels.Set{"test": "hardreboot"}).String(),
	}
	ipmiPods, err := pod.List(APIClient, nsName, listOptions)

	if err != nil {
		return err
	}

	// pull openshift apiserver deployment object to wait for after the node reboot.
	openshiftAPIDeploy, err := deployment.Pull(APIClient, "apiserver", "openshift-apiserver")

	if err != nil {
		return err
	}

	cmdToExec := []string{"ipmitool", "chassis", "power", "cycle"}

	glog.V(90).Infof("Exec cmd %v on pod %s", cmdToExec, ipmiPods[0].Definition.Name)
	_, err = ipmiPods[0].ExecCommand(cmdToExec)

	if err != nil {
		return err
	}

	// wait for the openshift apiserver deployment to be available
	err = openshiftAPIDeploy.WaitUntilCondition("Available", 5*time.Minute)

	if err != nil {
		return err
	}

	err = createDeploy.DeleteAndWait(2 * time.Minute)

	if err != nil {
		return err
	}

	return nil
}

// KernelCrashKdump triggers a kernel crash dump which generates a vmcore dump.
func KernelCrashKdump(nodeName string) error {
	// pull openshift apiserver deployment object to wait for after the node reboot.
	openshiftAPIDeploy, err := deployment.Pull(APIClient, "apiserver", "openshift-apiserver")

	if err != nil {
		return err
	}

	cmdToExec := []string{"chroot", "/rootfs", "/bin/sh", "-c", "rm -rf /var/crash/*"}
	glog.V(90).Infof("Remove any existing crash dumps. Exec cmd %v", cmdToExec)
	_, err = cmd.ExecCmd(cmdToExec, nodeName)

	if err != nil {
		return err
	}

	cmdToExec = []string{"/bin/sh", "-c", "echo c > /proc/sysrq-trigger"}

	glog.V(90).Infof("Trigerring kernel crash. Exec cmd %v", cmdToExec)
	_, err = cmd.ExecCmd(cmdToExec, nodeName)

	if err != nil {
		return err
	}

	// wait for the openshift apiserver deployment to be available
	err = openshiftAPIDeploy.WaitUntilCondition("Available", 5*time.Minute)

	if err != nil {
		return err
	}

	return nil
}
