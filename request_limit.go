package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sapi "k8s.io/kubernetes/pkg/api"

	resourcehelper "k8s.io/kubernetes/pkg/api/resource"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

func (nodeResource *NodeResource) NodeRequest(d clientset.Interface, podName, podNamespace, namespace, name string) {
	nodeResource.getNodeResource(d, namespace, name)

	// pod, err := d.Core().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	pod, err := d.Core().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		nodeResource.Schedule = false
		nodeResource.FailedMessage = err.Error()
		return
	}

	var podList = k8sapi.PodList{Items: []k8sapi.Pod{*pod}}
	podReqs, _, err := getPodsTotalRequestsAndLimits(&podList)

	if err != nil {
		nodeResource.Schedule = false
		nodeResource.FailedMessage = err.Error()
		return
	}

	N_pCpuReqs := podReqs[k8sapi.ResourceCPU]
	N_pMemReqs := podReqs[k8sapi.ResourceMemory]
	pCpuReqs := float64(nodeResource.CpuReqs.MilliValue()+N_pCpuReqs.MilliValue()) / float64(nodeResource.Allocate.Cpu().MilliValue()) * 100
	pMemoryReqs := float64(nodeResource.MemoryReqs.Value()+N_pMemReqs.Value()) / float64(nodeResource.Allocate.Memory().Value()) * 100
	if nodeResource.Schedule {
		if int64(pCpuReqs) > nodeResource.Threshold["cpu"] || int64(pMemoryReqs) > nodeResource.Threshold["memory"] {
			fmt.Println("pod " + podName + " with namespace: " + podNamespace + " wasn't scheduled to node: " + nodeResource.Name)
			fmt.Println("Node Information - cpu: " + fmt.Sprint(pCpuReqs) + " memory: " + fmt.Sprint(pMemoryReqs))
			nodeResource.Schedule = false
			nodeResource.FailedMessage = "pCpuReqs > " + fmt.Sprint(nodeResource.Threshold["cpu"]) + " or pMemoryReqs > " + fmt.Sprint(nodeResource.Threshold["memory"]) + " greater than thresh hold"
		} else {
			fmt.Println("pod " + podName + " with namespace: " + podNamespace + " could be scheduled to node: " + nodeResource.Name)
			fmt.Println(nodeResource.Name + " scheduled")
			nodeResource.Schedule = true
		}
	}

	return
}

func (nodeResource *NodeResource) getNodeResource(d clientset.Interface, namespace, name string) {
	nodeResource.Name = name

	mc := d.Core().Nodes()
	node, err := mc.Get(name, metav1.GetOptions{})
	var podList *k8sapi.PodList

	if err != nil {
		nodeResource.Schedule = false
		fmt.Print(err.Error())
		// return nodeResource, err
		return
	}

	fieldSelector, err := fields.ParseSelector("spec.nodeName=" + name + ",status.phase!=" + string(k8sapi.PodSucceeded) + ",status.phase!=" + string(k8sapi.PodFailed))
	if err != nil {
		fmt.Print(err.Error())
		nodeResource.Schedule = false
		return
	}
	fmt.Println("Field selector is " + fieldSelector.String())
	podList, err = d.Core().Pods(namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		fmt.Print(err.Error())
		nodeResource.Schedule = false
		// return nodeResource, err
		return
	}

	nodeResource.describeNodeResource(podList, node)

	return
}

func (nodeResource *NodeResource) describeNodeResource(podList *k8sapi.PodList, node *k8sapi.Node) {
	reqs, limits, err := getPodsTotalRequestsAndLimits(podList)
	if err != nil {
		fmt.Println(err.Error())
		nodeResource.Schedule = false
		// return nodeResource, err
		return
	}

	allocatable := node.Status.Capacity
	if len(node.Status.Allocatable) > 0 {
		allocatable = node.Status.Allocatable
	}

	cpuReqs, cpuLimits, memoryReqs, memoryLimits := reqs[k8sapi.ResourceCPU], limits[k8sapi.ResourceCPU], reqs[k8sapi.ResourceMemory], limits[k8sapi.ResourceMemory]
	nodeResource.Allocate = allocatable
	nodeResource.CpuReqs = cpuReqs
	nodeResource.CpuLimits = cpuLimits
	nodeResource.MemoryReqs = memoryReqs
	nodeResource.MemoryLimits = memoryLimits

	// return nodeResource, err
	return
}

func getPodsTotalRequestsAndLimits(podList *k8sapi.PodList) (reqs map[k8sapi.ResourceName]resource.Quantity, limits map[k8sapi.ResourceName]resource.Quantity, err error) {
	reqs, limits = map[k8sapi.ResourceName]resource.Quantity{}, map[k8sapi.ResourceName]resource.Quantity{}
	for _, pod := range podList.Items {
		podReqs, podLimits, err := resourcehelper.PodRequestsAndLimits(&pod)
		if err != nil {
			return nil, nil, err
		}
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
	return reqs, limits, err
}
