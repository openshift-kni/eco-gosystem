package powermanagementhelper

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/internal/gitopsztpinittools"
	"github.com/openshift-kni/eco-gosystem/tests/gitopsztp/powermanagement/internal/powermanagementparams"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/onsi/gomega"
)

// DeployProcessExporter deploys process exporter and returns the daemonset and error if any.
func DeployProcessExporter() *daemonset.Builder {
	daemonSet, err := daemonset.Pull(gitopsztpinittools.HubAPIClient, powermanagementparams.PromNamespace,
		powermanagementparams.ProcessExporterPodName)
	Expect(err).ShouldNot(HaveOccurred())

	Eventually(func() error {
		daemonSet, err = daemonset.Pull(gitopsztpinittools.HubAPIClient, powermanagementparams.PromNamespace,
			powermanagementparams.ProcessExporterPodName)

		if err != nil {
			glog.V(100).Infof("Failed to retrieve status for daemonset: ", daemonSet.Object.Name)

			return err
		}
		if daemonSet.Object.Spec.Template.Spec.Containers[0].Image != powermanagementparams.ProcessExporterImage {
			// Update dummy helper.Config.Ran.ProcessExporterImage to configured value
			daemonSet.Object.Spec.Template.Spec.Containers[0].Image = powermanagementparams.ProcessExporterImage
			daemonSet, err = daemonSet.Update()

			return fmt.Errorf("image updated, retrieve updated daemonset in next round")
		}
		if daemonSet.Object.Status.NumberReady == daemonSet.Object.Status.DesiredNumberScheduled {
			glog.V(100).Infof("All pods are ready in daemonset: %s\n", daemonSet.Object.Name)

			return nil
		}

		return fmt.Errorf("not all daemon pods are ready in daemonset: %s", daemonSet.Object.Name)
	}, 5*time.Minute, 5*time.Second).ShouldNot(HaveOccurred())

	return daemonSet
}

// GetPerformanceProfileWithCPUSet returns the first performance profile found with reserved and isolated cpuset.
func GetPerformanceProfileWithCPUSet() (*nto.Builder, error) {
	profilesBuilder, err := nto.ListProfiles(gitopsztpinittools.HubAPIClient)
	if err != nil {
		return nil, err
	}

	for _, profileBuilder := range profilesBuilder {
		if profileBuilder.Object.Spec.CPU != nil &&
			profileBuilder.Object.Spec.CPU.Reserved != nil &&
			profileBuilder.Object.Spec.CPU.Isolated != nil {
			return profileBuilder, nil
		}
	}

	return nil, fmt.Errorf("performance profile with Reserved and Isolated CPU set is not found")
}

// IsSingleNodeCluster returns true if the cluster is single-node, otherwise false.
func IsSingleNodeCluster(client *clients.Settings) bool {
	nodeList, err := nodes.List(client)
	Expect(err).ToNot(HaveOccurred())

	return len(nodeList) == 1
}

// DefineQoSTestPod defines test pod with given cpu and memory resources.
func DefineQoSTestPod(namespace namespace.Builder, nodeName, cpuReq, cpuLimit, memReq, memLimit string) *pod.Builder {
	image := powermanagementparams.CnfTestImage
	// Create namespace if not already created
	if !namespace.Exists() {
		glog.V(100).Infof("Creating namespace:", namespace)
		_, err := namespace.Create()
		Expect(err).ShouldNot(HaveOccurred())
	}

	pod := pod.NewBuilder(gitopsztpinittools.HubAPIClient, "", namespace.Definition.Name, image)
	pod = RedefineContainerResources(pod, cpuReq, cpuLimit, memReq, memLimit).DefineOnNode(nodeName)

	return pod
}

// RedefineContainerResources redefines a pod builder with CPU and Memory resources in first container
// Use empty string to skip a resource. e.g., cpuLimit="".
func RedefineContainerResources(
	pod *pod.Builder, cpuRequest string, cpuLimit string, memoryRequest string, memoryLimit string) *pod.Builder {
	pod.Definition.Spec.Containers[0].Resources.Requests = corev1.ResourceList{}
	pod.Definition.Spec.Containers[0].Resources.Limits = corev1.ResourceList{}

	if cpuLimit != "" {
		pod.Definition.Spec.Containers[0].Resources.Limits["cpu"] = resource.MustParse(cpuLimit)
	}

	if cpuRequest != "" {
		pod.Definition.Spec.Containers[0].Resources.Requests["cpu"] = resource.MustParse(cpuRequest)
	}

	if memoryLimit != "" {
		pod.Definition.Spec.Containers[0].Resources.Limits["memory"] = resource.MustParse(memoryLimit)
	}

	if memoryRequest != "" {
		pod.Definition.Spec.Containers[0].Resources.Requests["memory"] = resource.MustParse(memoryRequest)
	}

	return pod
}
