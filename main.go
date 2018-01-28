package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"errors"
	"strconv"
	// "errors"
	// "io/ioutil"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	rest "k8s.io/client-go/rest"
	k8sapi "k8s.io/kubernetes/pkg/api"
	k8sapiV1 "k8s.io/api/core/v1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	schedulerapi "k8s.io/kubernetes/plugin/pkg/scheduler/api"
	// "k8s.io/kubernetes/plugin/pkg/scheduler"
	"strings"
	"time"
	"regexp"
)

var Client *clientset.Clientset

// var Threshold_Config map[string]int64
var Threshold_Config ThreshodInfo
var Config map[string]string
var PrometheusHealthCheck bool
const (
	// LabelNodeRoleMaster specifies that a node is a master
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRoleMaster = "node-role.kubernetes.io/master"

	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"

	// NodeLabelKubeadmAlphaRole is a label that kubeadm applies to a Node as a hint that it has a particular purpose.
	// Use of NodeLabelRole is preferred.
	NodeLabelKubeadmAlphaRole = "kubeadm.alpha.kubernetes.io/role"
)

// func SchedulerWorker() {

// }

func findNodeRole(node k8sapiV1.Node) string {
	if _, ok := node.Labels[LabelNodeRoleMaster]; ok {
		return "Master"
	}
	if role := node.Labels[NodeLabelRole]; role != "" {
		return strings.Title(role)
	}
	if role := node.Labels[NodeLabelKubeadmAlphaRole]; role != "" {
		return strings.Title(role)
	}
	return ""
}

func CheckPrometheusServer() {
	for {
		client := &http.Client{
			Timeout: time.Duration(10 * time.Second),
		}

		_, err := client.Get(Config["prometheus_server"])	
		if err != nil {
			PrometheusHealthCheck = false
		} else {
			PrometheusHealthCheck = true
		}
		time.Sleep(time.Duration(1)*time.Second)
	}
}

func MetricChecker(result schedulerapi.ExtenderFilterResult, nResource NodeResource, node k8sapiV1.Node) (schedulerapi.ExtenderFilterResult, error) {
	if PrometheusHealthCheck {
		fmt.Println("Prometheus is alived")
		if Threshold_Config.Load != 0 {
			load_dict := make(map[string]float64)
			load_dict = Loads_metric()
			// fmt.Println(load_dict)
			fmt.Println("We commin threshold loads check")
			if _, ok := load_dict[nResource.Name]; ok {
				if load_dict[nResource.Name] < Threshold_Config.Load {
					fmt.Println("Allow node : " + nResource.Name + " scheduled pod with load " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64))
					// nodeList = append(nodeList, node)
				} else {
					fmt.Println("Node " + nResource.Name + " has load: " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64) + ", we do not schedule")
					result.FailedNodes[node.ObjectMeta.Name] = "Node has load > " + strconv.FormatFloat(Threshold_Config.Load, 'f', -1, 64)
					return result, errors.New("failed")
				}
			}
		}

		if Threshold_Config.CpuIdle != 0 {
			fmt.Println("We commin threshold cpu_idle check")
			cpu_dict := make(map[string]float64)
			cpu_dict = Cpu_Idle()
			// fmt.Println(cpu_dict)
			if _, ok := cpu_dict[nResource.Name]; ok {
					if cpu_dict[nResource.Name] < Threshold_Config.CpuIdle {
						fmt.Println("Node " + nResource.Name + " has cpu_idle: " + strconv.FormatFloat(cpu_dict[nResource.Name], 'f', -1, 64) + ", we do not scale")
						result.FailedNodes[node.ObjectMeta.Name] = "Node has cpu_idle < " + strconv.FormatFloat(Threshold_Config.CpuIdle, 'f', -1, 64)
						return result, errors.New("failed")					
					} else {
						fmt.Println("Allow node : " + nResource.Name + " scheduled pod with cpu_idle " + strconv.FormatFloat(cpu_dict[nResource.Name], 'f', -1, 64))							
					}
			}
		}
	}

	return result, nil
}

func SchedulerFunc(c *gin.Context) {
	fmt.Println("We start Scheduler")
	var args schedulerapi.ExtenderArgs
	c.BindJSON(&args)

	result := schedulerapi.ExtenderFilterResult{}
	result.FailedNodes = make(map[string]string)
	pod := args.Pod
	nodes := args.Nodes
	result.Nodes = nodes
	var err error
	var nodeList = []k8sapiV1.Node{}

	for _, node := range nodes.Items {
		// fmt.Println(Config)
		// fmt.Println(Threshold_Config)
		var nResource = NodeResource{Schedule: true, Threshold: Threshold_Config}
		if role := findNodeRole(node); (role == "Master" || role == "") && Config["role_check"] == "true" {
			fmt.Println("Node role master or role node empty")
			result.FailedNodes[node.ObjectMeta.Name] = "Node role master or role node empty"
			continue
		}

		nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", node.ObjectMeta.Name)
		// if nResource.Schedule {
		// 	nodeList = append(nodeList, node)
		// } else {
		// 	result.FailedNodes[node.ObjectMeta.Name] = nResource.FailedMessage
		// }

		if nResource.Schedule == false {
			result.FailedNodes[node.ObjectMeta.Name] = nResource.FailedMessage
			continue
		}

		// if PrometheusHealthCheck {
		// 	fmt.Println("Prometheus is alived")
		// 	if Threshold_Config.Load != 0 {
		// 		load_dict := make(map[string]float64)
		// 		load_dict = Loads_metric()
		// 		fmt.Println(load_dict)
		// 		fmt.Println("We commin threshold loads check")
		// 		if _, ok := load_dict[nResource.Name]; ok {
		// 			if load_dict[nResource.Name] < Threshold_Config.Load {
		// 				fmt.Println("Allow node : " + nResource.Name + " scheduled pod with load " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64))
		// 				// nodeList = append(nodeList, node)
		// 			} else {
		// 				fmt.Println("Node " + nResource.Name + " has load: " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64) + ", we do not schedule")
		// 				result.FailedNodes[node.ObjectMeta.Name] = "Node has load > " + strconv.FormatFloat(Threshold_Config.Load, 'f', -1, 64)
		// 				continue
		// 			}
		// 		}

		// 		if Threshold_Config.CpuIdle != 0 {
		// 			fmt.Println("We commin threshold cpu_idle check")
		// 			cpu_dict := make(map[string]float64)
		// 			cpu_dict = Cpu_Idle()
		// 			fmt.Println(cpu_dict)
		// 			if _, ok := cpu_dict[nResource.Name]; ok {
		// 					if cpu_dict[nResource.Name] < Threshold_Config.CpuIdle {
		// 						fmt.Println("Node " + nResource.Name + " has cpu_idle: " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64) + ", we do not scale")
		// 						result.FailedNodes[node.ObjectMeta.Name] = "Node has cpu_idle < " + strconv.FormatFloat(Threshold_Config.CpuIdle, 'f', -1, 64)
		// 						continue							
		// 					} else {
		// 						fmt.Println("Allow node : " + nResource.Name + " scheduled pod with cpu_idle " + strconv.FormatFloat(cpu_dict[nResource.Name], 'f', -1, 64))							
		// 					}
		// 			}
		// 		}

		// 		nodeList = append(nodeList, node)
		// 	} else {
		// 		nodeList = append(nodeList, node)
		// 	}
		// } else {
		// 	nodeList = append(nodeList, node)
		// 	fmt.Println("Prometheus Server Failed, Bypass apply Advanced checking")
		// }

		result, err = MetricChecker(result, nResource, node)
		if err == nil {
			nodeList = append(nodeList, node)	
		} else {
			fmt.Println(err.Error())
		}
		
	}
	result.Nodes.Items = nodeList
	c.JSON(200, result)
}

// The latest scheduling decision from scheduler
func FinalScheduleResult(c *gin.Context) {
	var bind schedulerapi.ExtenderBindingArgs
	c.BindJSON(&bind)

	fmt.Println("=====")
	fmt.Println("latest scheduling decision from scheduler")
	fmt.Println(bind)
	fmt.Println("=====")
	resp := map[string]string{
		"Error": "",
	}
	c.JSON(200, resp)
}

func ClusterAutoscaler(c *gin.Context) {
	var ca ScaleCA
	c.BindJSON(&ca)

	var nResource = NodeResource{Schedule: true, Threshold: Threshold_Config}
	var pod *k8sapi.Pod
	var err error
	// var nodeList = []k8sapiV1.Node{}
	result := schedulerapi.ExtenderFilterResult{}
	result.FailedNodes = make(map[string]string)
	
	resp := "true"
	// fmt.Println(ca.Pod)


	r, _ := regexp.Compile("template-node-for-nodes")

	if ! r.Match([]byte(ca.Node)) {
		pod, err = GetPod(Client, ca.Pod.ObjectMeta.Name, ca.Pod.ObjectMeta.Namespace)
		// fmt.Println(pod)
		// Case pod deleted before checked or apiserver connect issue --> just return 200
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("oh yeah")
			resp = "false"
		}
		fmt.Println("====================")
		fmt.Println(pod.ObjectMeta.Name)
		fmt.Println(pod.ObjectMeta.Namespace)
		fmt.Println(ca.Node)
		fmt.Println("====================")
		// if pod's NodeName as the same nodename from request --> return false  else process
		nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", ca.Node)
		

		// if pod.Spec.NodeName != ca.NodeName {

		// 	nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", ca.NodeName)

		// 	if nResource.Schedule == false {
		// 		fmt.Println(nResource.FailedMessage)
		// 		resp = "false"
		// 	} else {
		// 		nodeList = append(nodeList, node)
		// 		nodeList = MetricChecker(nodeList, nResource)
		// 	}
		// } else {
		// 	resp = "false"
		// }
		if pod.Status.Phase == "Pending" {
			nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", ca.Node)

			if nResource.Schedule == false {
				fmt.Println(nResource.FailedMessage)
				resp = "false"
			}
		} else {
			if pod.Spec.NodeName != ca.Node {

				nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", ca.Node)

				if nResource.Schedule == false {
					fmt.Println(nResource.FailedMessage)
					resp = "false"
				}
			}
		}

		var nodeInfo k8sapiV1.Node
		nodeInfo.ObjectMeta.Name = ca.Node
		if resp == "true" {
			_, err = MetricChecker(result, nResource, nodeInfo)
			if err != nil {
				resp = "false"
			}
		}
	}

	c.String(200, resp)
}

func Ping(c *gin.Context) {
	c.String(200, "pong")
}

func main() {
	// Threshold_Config = make(map[string]int64)
	Config = make(map[string]string)

	// certData, _ := ioutil.ReadFile("/data/suker/git/minikube/.minikube/apiserver.crt")

	// keyData, _ := ioutil.ReadFile("/data/suker/git/minikube/.minikube/apiserver.key")

	// // var err error
	// config := &rest.Config{
	// 	Host: "https://127.0.0.1:8443",
	// 	TLSClientConfig: rest.TLSClientConfig{
	// 		Insecure: true,
	// 		CertFile: "/data/suker/git/minikube/.minikube/apiserver.crt",
	// 		KeyFile:  "/data/suker/git/minikube/.minikube/apiserver.key",
	// 		CertData: certData,
	// 		KeyData:  keyData,
	// 	},
	// }

	test_mode := flag.String("test_mode", "false", "enable test mode")
	role_check := flag.String("role_check", "false", "enable role check default: false") 
	prometheus_server := flag.String("prometheus_server", "http://prometheus-prometheus-server.devops.svc.cluster.local", "prometheus server for query metrics")
	cpu_threshold := flag.Int64("cpu_threshold", 60, "cpu threshold per node to make schedule decision")
	memory_threshold := flag.Int64("memory_threshold", 70, "memory threshold per node to make schedule decision")
	load_threshold := flag.Float64("load_threshold", 0, "load avg threshold per node to make schedule decision")
	cpu_idle_threshold := flag.Float64("cpu_idle_threshold", 0, "cpu idle threshold per node to make schedule decision")
	flag.Parse()

	Threshold_Config.Cpu = *cpu_threshold
	Threshold_Config.Memory = *memory_threshold
	Threshold_Config.Load = *load_threshold
	Threshold_Config.CpuIdle = *cpu_idle_threshold
	Config["prometheus_server"] = *prometheus_server
	Config["test_mode"] = *test_mode
	Config["role_check"] = *role_check

	fmt.Println(Config)
	fmt.Println(Threshold_Config)

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err.Error())
		if Config["test_mode"] == "false" {
			panic(err)
		}
	}

	if Config["test_mode"] == "false" {
		Client = clientset.NewForConfigOrDie(config)
	}
	
	r := gin.Default()
	r.POST("v1/scheduler", SchedulerFunc)
	// r.POST("v1/bind", FinalScheduleResult)
	r.POST("v1/ca", ClusterAutoscaler)
	r.GET("/ping", Ping)

	s := &http.Server{
		Addr:           ":12345",
		Handler:        r,
		ReadTimeout:    120 * time.Second,
		WriteTimeout:   120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go CheckPrometheusServer()

	s.ListenAndServe()
}
