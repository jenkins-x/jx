package kube

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/log"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type NodeStatus struct {
	Name                      string
	AllocatedMemory           *resource.Quantity
	AllocatedCPU              *resource.Quantity
	CpuReqs                   resource.Quantity
	CpuLimits                 resource.Quantity
	percentCpuReq             int64
	percentCpuLimit           int64
	MemReqs                   resource.Quantity
	MemLimits                 resource.Quantity
	percentMemReq             int64
	percentMemLimit           int64
	numberOfNonTerminatedPods int
}

type ClusterStatus struct {
	Name                 string
	nodeCount            int
	totalCpuPercent      float64
	totalMemPercent      float64
	totalUsedMemory      int
	totalUsedCpu         int
	totalAllocatedMemory resource.Quantity
	totalAllocatedCpu    resource.Quantity
}

func GetClusterStatus(client kubernetes.Interface, namespace string) (ClusterStatus, error) {

	clusterStatus := ClusterStatus{
		totalAllocatedCpu:    resource.Quantity{},
		totalAllocatedMemory: resource.Quantity{},
	}

	config, _, err := LoadConfig()
	if err != nil {
		return clusterStatus, err
	}

	if config != nil {
		context := CurrentContext(config)
		if context != nil {
			clusterStatus.Name = context.Cluster
		}
	}

	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})

	if err != nil {
		return clusterStatus, err
	}
	clusterStatus.nodeCount = len(nodes.Items)
	if clusterStatus.nodeCount < 1 {
		msg := fmt.Sprintf("Number of nodes in cluster  = %d which is insufficent", clusterStatus.nodeCount)
		log.Fatal(msg)
		err = errors.NewServiceUnavailable(msg)
		return clusterStatus, err
	}
	for _, node := range nodes.Items {
		nodeStatus, err := Status(client, namespace, node)
		if err != nil {
			return clusterStatus, err
		}
		clusterStatus.totalCpuPercent += float64(nodeStatus.percentCpuReq)
		clusterStatus.totalMemPercent += float64(nodeStatus.percentMemReq)
		clusterStatus.totalAllocatedMemory.Add(*nodeStatus.AllocatedMemory)
		clusterStatus.totalAllocatedCpu.Add(*nodeStatus.AllocatedCPU)
		clusterStatus.totalUsedCpu += nodeStatus.CpuReqs.Size()
		clusterStatus.totalUsedMemory += nodeStatus.MemReqs.Size()

	}
	return clusterStatus, nil
}

func (clusterStatus *ClusterStatus) MinimumResourceLimit() int {
	return 80
}

func (clusterStatus *ClusterStatus) AverageCpuPercent() int {
	if clusterStatus.nodeCount > 0 {
		return int(clusterStatus.totalCpuPercent / float64(clusterStatus.nodeCount))
	}
	return int(clusterStatus.totalCpuPercent)
}

func (clusterStatus *ClusterStatus) AverageMemPercent() int {
	if clusterStatus.nodeCount > 0 {
		return int(clusterStatus.totalMemPercent / float64(clusterStatus.nodeCount))
	}
	return int(clusterStatus.totalMemPercent)
}

func (clusterStatus *ClusterStatus) NodeCount() int {
	return clusterStatus.nodeCount
}

func (clusterStatus *ClusterStatus) CheckResource() string {
	if clusterStatus.AverageMemPercent() >= clusterStatus.MinimumResourceLimit() {
		return "needs more free memory"
	}
	if clusterStatus.AverageCpuPercent() >= clusterStatus.MinimumResourceLimit() {
		return "needs more free compute"
	}
	return ""
}

func (clusterStatus *ClusterStatus) Info() string {
	str := fmt.Sprintf("Cluster(%s): %d nodes, memory %d%% of %s, cpu %d%% of %s",
		clusterStatus.Name,
		clusterStatus.NodeCount(),
		clusterStatus.AverageMemPercent(),
		clusterStatus.totalAllocatedMemory.String(),
		clusterStatus.AverageCpuPercent(),
		clusterStatus.totalAllocatedCpu.String())
	return str
}

func Status(client kubernetes.Interface, namespace string, node v1.Node) (NodeStatus, error) {
	nodeStatus := NodeStatus{}
	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + node.Name + ",status.phase!=" + string(v1.PodSucceeded) + ",status.phase!=" + string(v1.PodFailed))
	if err != nil {
		return nodeStatus, err
	}

	allocatable := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		allocatable = node.Status.Allocatable
	}

	nodeStatus.Name = node.Name
	nodeStatus.AllocatedCPU = allocatable.Cpu()
	nodeStatus.AllocatedMemory = allocatable.Memory()

	// in a policy aware setting, users may have access to a node, but not all pods
	// in that case, we note that the user does not have access to the pods

	nodeNonTerminatedPodsList, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		if !errors.IsForbidden(err) {
			return nodeStatus, err
		}
	}

	nodeStatus.numberOfNonTerminatedPods = len(nodeNonTerminatedPodsList.Items)

	reqs, limits := getPodsTotalRequestsAndLimits(nodeNonTerminatedPodsList)
	cpuReqs, cpuLimits, memoryReqs, memoryLimits := reqs[v1.ResourceCPU], limits[v1.ResourceCPU], reqs[v1.ResourceMemory], limits[v1.ResourceMemory]
	fractionCpuReqs := float64(0)
	fractionCpuLimits := float64(0)
	if allocatable.Cpu().MilliValue() != 0 {
		fractionCpuReqs = float64(cpuReqs.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
		fractionCpuLimits = float64(cpuLimits.MilliValue()) / float64(allocatable.Cpu().MilliValue()) * 100
	}
	fractionMemoryReqs := float64(0)
	fractionMemoryLimits := float64(0)
	if allocatable.Memory().Value() != 0 {
		fractionMemoryReqs = float64(memoryReqs.Value()) / float64(allocatable.Memory().Value()) * 100
		fractionMemoryLimits = float64(memoryLimits.Value()) / float64(allocatable.Memory().Value()) * 100
	}

	nodeStatus.CpuReqs = cpuReqs
	nodeStatus.percentCpuReq = int64(fractionCpuReqs)
	nodeStatus.CpuLimits = cpuLimits
	nodeStatus.percentCpuLimit = int64(fractionCpuLimits)
	nodeStatus.MemReqs = memoryReqs
	nodeStatus.percentMemReq = int64(fractionMemoryReqs)
	nodeStatus.MemLimits = memoryLimits
	nodeStatus.percentMemLimit = int64(fractionMemoryLimits)

	return nodeStatus, nil
}

func getPodsTotalRequestsAndLimits(podList *v1.PodList) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, pod := range podList.Items {

		podReqs, podLimits := PodRequestsAndLimits(&pod)
		for podReqName, podReqValue := range podReqs {
			if value, ok := reqs[podReqName]; !ok {
				reqs[podReqName] = *podReqValue.Copy()
			} else {
				value.Add(podReqValue)
				reqs[podReqName] = value
			}
		}
		for podLimitName, podLimitValue := range podLimits {
			if value, ok := limits[podLimitName]; !ok {
				limits[podLimitName] = *podLimitValue.Copy()
			} else {
				value.Add(podLimitValue)
				limits[podLimitName] = value
			}
		}
	}
	return
}

func PodRequestsAndLimits(pod *v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = *quantity.Copy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = *quantity.Copy()
			}
		}
	}
	return
}

func RoleBindings(client kubernetes.Interface, namespace string) (string, error) {
	binding, err := client.Rbac().RoleBindings(namespace).Get("", metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	result := ""

	for _, s := range binding.Subjects {
		result += fmt.Sprintf("%s\t%s\t%s\n", s.Kind, s.Name, s.Namespace)
	}

	return result, nil
}
