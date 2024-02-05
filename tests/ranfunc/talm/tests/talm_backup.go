package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cluster"
	"github.com/openshift-kni/eco-gosystem/tests/internal/cmd"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfunchelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/internal/ranfuncinittools"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmhelper"
	"github.com/openshift-kni/eco-gosystem/tests/ranfunc/talm/internal/talmparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configurationPolicyv1 "open-cluster-management.io/config-policy-controller/api/v1"
)

const (
	backupPath  = "/var/recovery"
	ranTestPath = "/var/ran-test-talm-recovery"
	fsSize      = "100M"
)

var (
	nodeName           string
	loopBackDevicePath string
)

var _ = Describe("Talm Backup Tests with single spoke", func() {

	BeforeEach(func() {
		if !ranfunchelper.IsVersionStringInRange(
			talmhelper.TalmHubVersion,
			"4.11",
			"",
		) {
			Skip("backup tests require talm 4.11 or higher")
		}
	})

	// ocp-50835
	Context("with full disk for spoke1", func() {
		curName := "disk-full-single-spoke"
		cguName := fmt.Sprintf("%s-%s", talmparams.CguCommonName, curName)
		policyName := fmt.Sprintf("%s-%s", talmparams.PolicyNameCommonName, curName)

		BeforeEach(func() {
			By("setting up filesystem to simulate low space")
			nodeList, err := nodes.List(ranfuncinittools.HubAPIClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(nodeList)).To(BeNumerically(">=", 1))

			nodeName := nodeList[0].Object.Name
			loopBackDevicePath = prepareEnvWithSmallMountPoint(nodeName)
		})

		AfterEach(func() {
			glog.V(100).Info("starting disk-full env clean up")
			diskFullEnvCleanup(nodeName, curName, loopBackDevicePath)

			// Delete temporary namespace on spoke cluster.
			spokeClusterList := []*clients.Settings{ranfuncinittools.SpokeAPIClient}
			err := talmhelper.CleanupNamespace(spokeClusterList, talmhelper.TemporaryNamespaceName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have a failed cgu for single spoke", func() {
			By("applying all the required CRs for backup")
			// prep cgu
			cgu := talmhelper.GetCguDefinition(
				cguName,
				[]string{talmhelper.Spoke1Name},
				[]string{},
				[]string{policyName},
				talmparams.TalmTestNamespace, 1, 240)
			cgu.Definition.Spec.Backup = true

			// apply
			err := talmhelper.CreatePolicyAndCgu(
				ranfuncinittools.HubAPIClient,
				namespace.NewBuilder(ranfuncinittools.HubAPIClient, talmhelper.TemporaryNamespaceName).Definition,
				configurationPolicyv1.MustHave,
				configurationPolicyv1.Inform,
				policyName,
				fmt.Sprintf("%s-%s", talmparams.PolicySetNameCommonName, curName),
				fmt.Sprintf("%s-%s", talmparams.PlacementBindingCommonName, curName),
				fmt.Sprintf("%s-%s", talmparams.PlacementRuleCommonName, curName),
				talmparams.TalmTestNamespace,
				metav1.LabelSelector{},
				cgu,
			)
			Expect(err).To(BeNil())

			By("waiting for cgu to fail for spoke1")
			assertBackupStatus(cgu.Definition.Name, talmhelper.Spoke1Name, "UnrecoverableError")
		})
	})

	Context("with CGU disabled", func() {
		curName := "backupsequence"
		cguName := fmt.Sprintf("%s-%s", talmparams.CguCommonName, curName)
		policyName := fmt.Sprintf("%s-%s", talmhelper.PolicyName, curName)
		cguEnabled := false

		AfterEach(func() {
			// Delete generated CRs on Hub Cluster.
			hubErrList := talmhelper.CleanupTestResourcesOnClient(
				ranfuncinittools.HubAPIClient,
				cguName,
				policyName,
				talmparams.TalmTestNamespace,
				fmt.Sprintf("%s-%s", talmparams.PlacementBindingCommonName, curName),
				fmt.Sprintf("%s-%s", talmparams.PlacementRuleCommonName, curName),
				fmt.Sprintf("%s-%s", talmhelper.PolicySetName, curName),
				"",
				false,
			)
			Expect(hubErrList).To(BeEmpty())

			// Delete temporary namespace on spoke cluster.
			spokeClusterList := []*clients.Settings{ranfuncinittools.SpokeAPIClient}
			err := talmhelper.CleanupNamespace(spokeClusterList, talmhelper.TemporaryNamespaceName)
			Expect(err).ToNot(HaveOccurred())

		})

		// ocp-54294, ocp-54295
		It("verifies backup begins and succeeds after CGU is enabled", func() {
			if !ranfunchelper.IsVersionStringInRange(talmhelper.TalmHubVersion, "4.12", "") {
				Skip("backup begins after CGU enable requires talm 4.12 or higher")
			}

			By("creating a disabled cgu with backup enabled")
			cgu := talmhelper.GetCguDefinition(
				cguName,
				[]string{talmhelper.Spoke1Name},
				[]string{},
				[]string{policyName},
				talmparams.TalmTestNamespace, 1, 30)

			cgu.Definition.Spec.Backup = true
			// passing reference to cguEnabled because cgu.Spec.Enable is of type BoolAddr
			cgu.Definition.Spec.Enable = &cguEnabled

			// apply cgu
			err := talmhelper.CreatePolicyAndCgu(
				ranfuncinittools.HubAPIClient,
				namespace.NewBuilder(ranfuncinittools.HubAPIClient, talmhelper.TemporaryNamespaceName).Definition,
				configurationPolicyv1.MustHave,
				configurationPolicyv1.Inform,
				policyName,
				fmt.Sprintf("%s-%s", talmparams.PolicySetNameCommonName, curName),
				fmt.Sprintf("%s-%s", talmparams.PlacementBindingCommonName, curName),
				fmt.Sprintf("%s-%s", talmparams.PlacementRuleCommonName, curName),
				talmparams.TalmTestNamespace,
				metav1.LabelSelector{},
				cgu,
			)
			Expect(err).ToNot(HaveOccurred())

			By("checking backup does not begin when CGU is disabled")
			err = talmhelper.WaitForBackupStart(
				ranfuncinittools.HubAPIClient,
				cgu.Definition.Name,
				cgu.Definition.Namespace,
				2*time.Minute,
			)
			Expect(err).To(HaveOccurred())

			By("enalble CGU")
			err = talmhelper.EnableCgu(ranfuncinittools.HubAPIClient, &cgu)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for backup to begin")
			err = talmhelper.WaitForBackupStart(
				ranfuncinittools.HubAPIClient,
				cgu.Definition.Name,
				cgu.Definition.Namespace,
				1*time.Minute,
			)
			Expect(err).ToNot(HaveOccurred())

			// Wait for spoke cluster backup to finish and report Succeeded.
			By("waiting for cgu to indicate backup succeeded for spoke")
			assertBackupStatus(cgu.Definition.Name, talmhelper.Spoke1Name, "Succeeded")

		})

	})

})

var _ = Describe("Talm Backup Tests with two spokes", Ordered, func() {

	curName := "disk-full-multiple-spokes"
	cguName := fmt.Sprintf("%s-%s", talmparams.CguCommonName, curName)
	policyName := fmt.Sprintf("%s-%s", talmparams.PolicyNameCommonName, curName)

	BeforeAll(func() {
		if !ranfunchelper.IsVersionStringInRange(
			talmhelper.TalmHubVersion,
			"4.11",
			"",
		) {
			Skip("backup tests require talm 4.11 or higher")
		}
		// tests below requires all clusters to be present. hub + spoke1 + spoke2
		clusterList := talmhelper.GetAllTestClients()
		// Check that the required clusters are present
		err := cluster.CheckClustersPresent(clusterList)
		if err != nil {
			Skip(fmt.Sprintf("error occurred validating required clusters are present: %s", err.Error()))
		}
	})

	BeforeEach(func() {
		By("setting up filesystem to simulate low space")
		nodeList, err := nodes.List(ranfuncinittools.HubAPIClient)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(nodeList)).To(BeNumerically(">=", 1))

		nodeName := nodeList[0].Object.Name
		loopBackDevicePath = prepareEnvWithSmallMountPoint(nodeName)
	})

	AfterEach(func() {
		glog.V(100).Info("starting disk-full env clean up")
		diskFullEnvCleanup(nodeName, curName, loopBackDevicePath)
		// Delete temporary namespace on spoke cluster.
		spokeClusterList := []*clients.Settings{ranfuncinittools.SpokeAPIClient, talmhelper.Spoke2APIClient}
		for _, spokeCluster := range spokeClusterList {
			err := namespace.NewBuilder(spokeCluster, talmhelper.TemporaryNamespaceName).CleanObjects(5 * time.Minute)
			Expect(err).ToNot(HaveOccurred())

		}
	})

	It("should not affect backup on second spoke in same batch", func() {
		By("applying all the required CRs for backup")
		// prep cgu
		cgu := talmhelper.GetCguDefinition(
			cguName,
			[]string{talmhelper.Spoke1Name, talmhelper.Spoke2Name},
			[]string{},
			[]string{policyName},
			talmparams.TalmTestNamespace, 100, 240)
		cgu.Definition.Spec.Backup = true

		// apply
		err := talmhelper.CreatePolicyAndCgu(
			ranfuncinittools.HubAPIClient,
			namespace.NewBuilder(ranfuncinittools.HubAPIClient, talmhelper.TemporaryNamespaceName).Definition,
			configurationPolicyv1.MustHave,
			configurationPolicyv1.Inform,
			policyName,
			fmt.Sprintf("%s-%s", talmparams.PolicySetNameCommonName, curName),
			fmt.Sprintf("%s-%s", talmparams.PlacementBindingCommonName, curName),
			fmt.Sprintf("%s-%s", talmparams.PlacementRuleCommonName, curName),
			talmparams.TalmTestNamespace,
			metav1.LabelSelector{},
			cgu,
		)
		Expect(err).To(BeNil())

		By("waiting for cgu to indicate it failed for spoke1")
		assertBackupStatus(cgu.Definition.Name, talmhelper.Spoke1Name, "UnrecoverableError")

		By("waiting for cgu to indicate it succeeded for spoke2")
		assertBackupStatus(cgu.Definition.Name, talmhelper.Spoke2Name, "Succeeded")
	})

})

// diskFullEnvCleanup clean all the resources created for single cluster backup fail.
func diskFullEnvCleanup(nodeName, curName, currentlyUsingLoopDevicePath string) {
	// delete generated CRs
	talmhelper.CleanupTestResourcesOnClient(
		ranfuncinittools.HubAPIClient,
		fmt.Sprintf("%s-%s", talmparams.CguCommonName, curName),
		fmt.Sprintf("%s-%s", talmparams.PolicyNameCommonName, curName),
		talmparams.TalmTestNamespace,
		fmt.Sprintf("%s-%s", talmparams.PlacementBindingCommonName, curName),
		fmt.Sprintf("%s-%s", talmparams.PlacementRuleCommonName, curName),
		fmt.Sprintf("%s-%s", talmparams.PolicySetNameCommonName, curName),
		"",
		false,
	)

	// check where backup dir is mounted and start clean up
	safeToDeleteBackupDir := true
	// retrieve all mounts for backup dir
	output, err := cmd.ExecCmd([]string{fmt.Sprintf("findmnt -n -o SOURCE --target %s", backupPath)}, nodeName)
	Expect(err).To(BeNil())

	output = strings.TrimSuffix(output, "\n")
	if output != "" {
		outputArr := strings.Split(output, "\n")
		for _, devicePath := range outputArr {
			// retrieve all devices e.g part or loop
			deviceType, err := cmd.ExecCmd([]string{fmt.Sprintf("lsblk %s -o TYPE -n", devicePath)}, nodeName)
			Expect(err).To(BeNil())

			deviceType = strings.TrimSuffix(deviceType, "\n")

			if deviceType == "part" {
				safeToDeleteBackupDir = false

				glog.V(100).Info("partition detected for %s, "+
					"will not attempt to delete the folder (only the content if any)", backupPath)
			} else if deviceType == "loop" {

				if currentlyUsingLoopDevicePath == devicePath {
					// unmount and detach the loop device
					_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo umount --detach-loop %s", backupPath)}, nodeName)
					Expect(err).To(BeNil())

				} else {
					safeToDeleteBackupDir = false
					glog.V(100).Info("WARNING: most likely cleanup didnt complete during the previous run. ")
					/*
						Assuming loop0 is the unwanted one...
						look for clues with lsblk
						$ lsblk
						NAME   MAJ:MIN RM   SIZE RO TYPE MOUNTPOINT
						loop0    7:0    0   100M  0 loop /var/recovery -----> this line should not be there

						unmount it with: `sudo umount --detach-loop /var/recovery`
						check lsblk to verify there's nothing mounted to loop0 and line is gone completely

						if line is still there (but unmounted) make use `losetup` to see the status of loopdevice (loop0)
						$ losetup
						NAME       SIZELIMIT OFFSET AUTOCLEAR RO BACK-FILE                                      DIO LOG-SEC
						/dev/loop0         0      0         1  0 /var/ran-test-talm-recovery/100M.img (deleted)   0     512

						if you see (deleted) -- reboot the node. i.e sudo reboot.
						Once back loop0 should not appear anywhere (lsblk + losetup)

					*/
					glog.V(100).Info("See comments for manual cleanup of %s\n", devicePath)
				}
			}
		}
	}

	// if true there was a partition (most likely ZTP /w MC) so delete content instead of the whole thing
	if safeToDeleteBackupDir {
		_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo rm -rf %s", backupPath)}, nodeName)
		Expect(err).To(BeNil())
	} else {
		_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo rm -rf %s/*", backupPath)}, nodeName)
		Expect(err).To(BeNil())
	}

	// delete ran-test-talm-recovery folder
	_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo rm -rf %s", ranTestPath)}, nodeName)
	Expect(err).To(BeNil())
}

// prepareEnvWithSmallMountPoint use loopback device,
// a virtual file system backed by a file, to create a small partition
// helpful links https://stackoverflow.com/q/16044204 and https://youtu.be/r9CQhwci4tE
func prepareEnvWithSmallMountPoint(nodeName string) string {
	// create a dir for backup if not already there
	_, err := cmd.ExecCmd([]string{fmt.Sprintf("sudo mkdir -p %s", backupPath)}, nodeName)
	Expect(err).To(BeNil())

	// create a dir for ran test dir if not already there
	_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo mkdir -p %s", ranTestPath)}, nodeName)
	Expect(err).To(BeNil())

	// find the next available loopback device (OS takes care of creating a new one if needed)
	loopBackDevicePath, err := cmd.ExecCmd([]string{"sudo losetup -f"}, nodeName)
	Expect(err).To(BeNil())

	loopBackDevicePath = strings.TrimSpace(loopBackDevicePath)
	glog.V(100).Info("loopback device path: ", loopBackDevicePath)

	// create a file with desired size. It's where the file-system will live
	_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo fallocate -l %s %s/%s.img", fsSize, ranTestPath, fsSize)}, nodeName)
	Expect(err).To(BeNil())

	// create the loop device by assigning it with the file. tip: use losetup -a to check the status
	_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo losetup %s %s/%s.img", loopBackDevicePath,
		ranTestPath, fsSize)}, nodeName)
	Expect(err).To(BeNil())

	// format to your desired fs type. xfs is RH preferred but ext4 works too.
	_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo mkfs.xfs -f -q %s", loopBackDevicePath)}, nodeName)
	Expect(err).To(BeNil())

	// mount the fs to backup dir
	_, err = cmd.ExecCmd([]string{fmt.Sprintf("sudo mount %s %s", loopBackDevicePath, backupPath)}, nodeName)
	Expect(err).To(BeNil())

	return loopBackDevicePath
}

// assertBackupPodLog asserts status of backup struct.
func assertBackupStatus(cguName, spokeName, expectation string) {
	Eventually(func() string {

		cgu, err := cgu.Pull(ranfuncinittools.HubAPIClient, cguName, talmparams.TalmTestNamespace)
		Expect(err).To(BeNil())

		if cgu.Object.Status.Backup == nil {
			glog.V(100).Info("backup struct not ready yet")

			return ""
		}

		_, ok := cgu.Object.Status.Backup.Status[spokeName]
		if !ok {
			glog.V(100).Info("cluster name as key did not appear yet")

			return ""
		}

		glog.V(100).Infof("[%s] %s backup status: %s\n", cgu.Object.Name, spokeName,
			cgu.Object.Status.Backup.Status[spokeName])

		return cgu.Object.Status.Backup.Status[spokeName]
	}, 10*time.Minute, 10*time.Second).Should(Equal(expectation))
}
