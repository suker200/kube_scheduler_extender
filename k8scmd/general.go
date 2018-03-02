package k8scmd

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

func GetPod(d clientset.Interface, podName, podNameSpace string) (*k8sapi.Pod, error) {
	return d.Core().Pods(podNameSpace).Get(podName, v1.GetOptions{})
}

// func DrainNode() {
	
// }