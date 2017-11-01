package main

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

func GetPod(d clientset.Interface, podName string) (*k8sapi.Pod, error) {
	return d.Core().Pods("").Get(podName, v1.GetOptions{})
}
