package main

import (
	"fmt"
	"kube_scheduler_extender/k8scmd"
	"bytes"
	"os"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	// "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func GetPod(d clientset.Interface, podName, podNameSpace string) (*k8sapi.Pod, error) {
	return d.Core().Pods(podNameSpace).Get(podName, v1.GetOptions{})
}

func CustomBabyClientConfig(kubeconfigFile string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigFile
	overrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, overrides, os.Stdin)

	return clientConfig
}

func TaintNode(nodeName string) error {
	buf, _ := bytes.NewBuffer([]byte{}), bytes.NewBuffer([]byte{})

	var f cmdutil.Factory
	var clientConfig clientcmd.ClientConfig

	clientConfig = CustomBabyClientConfig("/data/kubeconfig")
	f = cmdutil.NewFactory(clientConfig)
	c := k8scmd.CustomNewCmdTaint(f, buf)

	c.Flags().Set("overwrite", "true")

	if err := c.RunE(c, []string{"nodes", nodeName, "custom-scheduler=draining:NoExecute"}); err != nil {
		fmt.Println(err.Error())
		return err
	} else {
		fmt.Println("We taint successfully")
		return nil
	}
}
