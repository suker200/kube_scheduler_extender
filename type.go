package main

import (
	"k8s.io/apimachinery/pkg/api/resource"
	k8sapi "k8s.io/kubernetes/pkg/api"
)

type NodeResource struct {
	Name          string
	Schedule      bool
	CpuReqs       resource.Quantity
	CpuLimits     resource.Quantity
	MemoryReqs    resource.Quantity
	MemoryLimits  resource.Quantity
	Allocate      k8sapi.ResourceList
	FailedMessage string
	Threshold     map[string]int64
}
