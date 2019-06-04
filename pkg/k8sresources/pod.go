package k8sresources

import (
	"fmt"
	"strings"

	"github.com/bonnefoa/kubectl-fzf/pkg/util"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
)

// PodHeader is the header for pod files
const PodHeader = "Namespace Name PodIp HostIp NodeName Phase Containers Tolerations Claims Age Labels\n"

// Pod is the summary of a kubernetes pod
type Pod struct {
	ResourceMeta
	hostIP      string
	podIP       string
	nodeName    string
	tolerations []string
	containers  []string
	claims      []string
	phase       string
}

func getPhase(p *corev1.Pod) string {
	for _, v := range p.Status.ContainerStatuses {
		if v.State.Waiting != nil && v.State.Waiting.Reason != "Completed" {
			return v.State.Waiting.Reason
		}
	}
	return string(p.Status.Phase)
}

// NewPodFromRuntime builds a pod from informer result
func NewPodFromRuntime(obj interface{}, config CtorConfig) K8sResource {
	p := &Pod{}
	p.FromRuntime(obj, config)
	return p
}

// FromRuntime builds object from the informer's result
func (p *Pod) FromRuntime(obj interface{}, config CtorConfig) {
	pod := obj.(*corev1.Pod)
	glog.V(19).Infof("Reading meta %#v", pod)
	p.FromObjectMeta(pod.ObjectMeta)
	p.hostIP = pod.Status.HostIP
	p.podIP = pod.Status.PodIP
	spec := pod.Spec
	p.nodeName = spec.NodeName
	p.phase = getPhase(pod)

	containers := spec.Containers
	containers = append(containers, spec.InitContainers...)
	p.containers = make([]string, len(containers))
	for k, v := range containers {
		p.containers[k] = v.Name
	}

	volumes := spec.Volumes
	for _, v := range volumes {
		if v.PersistentVolumeClaim != nil {
			p.claims = append(p.claims, v.PersistentVolumeClaim.ClaimName)
		}
	}
	tolerations := spec.Tolerations
	p.tolerations = make([]string, 0)
	for _, v := range tolerations {
		if strings.HasPrefix(v.Key, "node.kubernetes.io") {
			continue
		}
		var toleration string
		if v.Operator == "Equal" {
			toleration = fmt.Sprintf("%s=%s:%s", v.Key, v.Value, v.Effect)
		} else if v.Key == "" {
			toleration = "Exists"
		} else {
			toleration = fmt.Sprintf("%s:%s", v.Key, v.Effect)
		}
		p.tolerations = append(p.tolerations, toleration)
	}
}

// HasChanged returns true if the resource's dump needs to be updated
func (p *Pod) HasChanged(k K8sResource) bool {
	oldPod := k.(*Pod)
	return (p.podIP != oldPod.podIP ||
		p.phase != oldPod.phase ||
		util.StringMapsEqual(p.labels, oldPod.labels) ||
		p.nodeName != oldPod.nodeName)
}

// ToString serializes the object to strings
func (p *Pod) ToString() string {
	lst := []string{
		p.namespace,
		p.name,
		p.podIP,
		p.hostIP,
		p.nodeName,
		p.phase,
		util.JoinSlicesOrNone(p.containers, ","),
		util.JoinSlicesOrNone(p.tolerations, ","),
		util.JoinSlicesOrNone(p.claims, ","),
		p.resourceAge(),
		p.labelsString(),
	}
	return util.DumpLine(lst)
}
