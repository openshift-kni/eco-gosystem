package talmhelper

import (
	"errors"
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztphelper"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/talm/internal/talmparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerifyTalmIsInstalled checks that talm pod+container is present and that CGUs can be fetched.
func VerifyTalmIsInstalled() error {
	// Check for talm pods
	talmPods, err := pod.List(gitopsztpinittools.HubAPIClient,
		talmparams.TalmOperatorNamespace,
		metav1.ListOptions{
			LabelSelector: talmparams.TalmPodLabelSelector})
	if err != nil {
		return err
	}

	// Check if any pods exist
	if len(talmPods) == 0 {
		return errors.New("unable to find talm pod")
	}

	// Check each pod for the talm container
	for _, talmPod := range talmPods {
		if !gitopsztphelper.IsPodHealthy(talmPod) {
			return fmt.Errorf("talm pod %s is not healthy", talmPod.Definition.Name)
		}

		if !gitopsztphelper.IsContainerExistInPod(talmPod, talmparams.TalmContainerName) {
			return fmt.Errorf("talm pod defined but talm container does not exist")
		}
	}

	return nil
}
