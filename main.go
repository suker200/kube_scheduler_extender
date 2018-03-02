package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"errors"
	"strconv"
	// "errors"
	"io/ioutil"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	k8sapi "k8s.io/kubernetes/pkg/api"
	k8sapiV1 "k8s.io/api/core/v1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	schedulerapi "k8s.io/kubernetes/plugin/pkg/scheduler/api"
	// "k8s.io/kubernetes/plugin/pkg/scheduler"
	"strings"
	"time"
	"regexp"
	"gopkg.in/yaml.v2"
	"github.com/fsnotify/fsnotify"
	// "os"
)

var Client *clientset.Clientset

// var Config.Threshold map[string]int64
// var Config.Threshold ThreshodInfo
var Config ConfigInfo
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

		_, err := client.Get(Config.PrometheusServer)	
		if err != nil {
			PrometheusHealthCheck = false
		} else {
			PrometheusHealthCheck = true
		}
		time.Sleep(time.Duration(1)*time.Second)
	}
}

func DetectSpotInstance(nodeList []k8sapiV1.Node) []k8sapiV1.Node {
	var spotnodes = []k8sapiV1.Node{}
	var demandnodes = []k8sapiV1.Node{}
	for _, node := range nodeList {
		if ok := Config.CheckSpotNode(node); ok {
			spotnodes = append(spotnodes, node)
		} else {
			demandnodes = append(demandnodes, node)
		}
	}

	if len(spotnodes) != 0 {
		return spotnodes
	} 

	return demandnodes
}

func CheckNodeLabel(nodeList []k8sapiV1.Node, label string) []k8sapiV1.Node {
	var nodes = []k8sapiV1.Node{}
	for _, node := range nodeList {
		if len(node.ObjectMeta.Labels) != 0 {
			if _, ok := node.ObjectMeta.Labels[Config.SpotReserveLabel]; ok {
				nodes = append(nodes, node)
			}
		} 
	}

	return nodes
}

func MetricChecker(result schedulerapi.ExtenderFilterResult, nResource NodeResource, node k8sapiV1.Node) (schedulerapi.ExtenderFilterResult, error) {
	if PrometheusHealthCheck {
		fmt.Println("Prometheus is alived")
		if Config.Threshold.Load != 0 {
			load_dict := make(map[string]float64)
			load_dict = Config.Loads_metric()
			// fmt.Println(load_dict)
			fmt.Println("We commin threshold loads check")
			fmt.Println(load_dict)
			fmt.Println(nResource.Name)
			fmt.Println("We commin threshold loads check")
			if _, ok := load_dict[nResource.Name]; ok {
				if load_dict[nResource.Name] < Config.Threshold.Load {
					fmt.Println("Allow node : " + nResource.Name + " scheduled pod with load " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64))
					// nodeList = append(nodeList, node)
				} else {
					fmt.Println("Node " + nResource.Name + " has load: " + strconv.FormatFloat(load_dict[nResource.Name], 'f', -1, 64) + ", we do not schedule")
					result.FailedNodes[node.ObjectMeta.Name] = "Node has load > " + strconv.FormatFloat(Config.Threshold.Load, 'f', -1, 64)
					return result, errors.New("failed")
				}
			}
		}

		if Config.Threshold.CpuIdle != 0 {
			fmt.Println("We commin threshold cpu_idle check")
			cpu_dict := make(map[string]float64)
			cpu_dict = Config.Cpu_Idle()
			fmt.Println(cpu_dict)
			fmt.Println(nResource.Name)
			fmt.Println("We commin threshold cpu_idle check")
			if _, ok := cpu_dict[nResource.Name]; ok {
				if cpu_dict[nResource.Name] < Config.Threshold.CpuIdle {
					fmt.Println("Node " + nResource.Name + " has cpu_idle: " + strconv.FormatFloat(cpu_dict[nResource.Name], 'f', -1, 64) + ", we do not scale")
					result.FailedNodes[node.ObjectMeta.Name] = "Node has cpu_idle < " + strconv.FormatFloat(Config.Threshold.CpuIdle, 'f', -1, 64)
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
	// nodes := args.Nodes
	result.Nodes = args.Nodes
	var err error
	var nodeList = []k8sapiV1.Node{} 
	// var nodes = []k8sapiV1.Node{} 

	for _, node := range args.Nodes.Items {
		var nResource = NodeResource{Schedule: true, Threshold: Config.Threshold}
		if role := findNodeRole(node); (role == "Master" || role == "") && ! Config.TestMode {
			fmt.Println("Node role master or role node empty")
			result.FailedNodes[node.ObjectMeta.Name] = "Node role master or role node empty"
			continue
		}

		nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", node.ObjectMeta.Name)

		if nResource.Schedule == false {
			result.FailedNodes[node.ObjectMeta.Name] = nResource.FailedMessage
			continue
		}

		result, err = MetricChecker(result, nResource, node)
		if err == nil {
			nodeList = append(nodeList, node)	
		} else {
			fmt.Println(err.Error())
		}	
	}

	if Config.SpotReserveLabel != "" {
		if _, ok := pod.ObjectMeta.Labels[Config.SpotReserveLabel]; ok {
			nodeList = CheckNodeLabel(nodeList, Config.SpotReserveLabel)
		}
	}
	
	// If spot enable and spot instance exist, remove all demand instance from list
	if Config.SpotEnable {
		result.Nodes.Items = DetectSpotInstance(nodeList)
	} else {
		result.Nodes.Items = nodeList
	}

	c.JSON(200, result)
}

// The latest scheduling decision from scheduler
func FinalScheduleResult(c *gin.Context) {
	var bind schedulerapi.ExtenderBindingArgs
	c.BindJSON(&bind)

	resp := map[string]string{
		"Error": "",
	}
	c.JSON(200, resp)
}


func ClusterAutoscaler(c *gin.Context) {
	var ca ScaleCA
	c.BindJSON(&ca)

	var nResource = NodeResource{Schedule: true, Threshold: Config.Threshold}
	var pod *k8sapi.Pod
	var err error

	result := schedulerapi.ExtenderFilterResult{}
	result.FailedNodes = make(map[string]string)
	
	resp := "true"

	r, _ := regexp.Compile("template-node-for")

	pod, err = GetPod(Client, ca.Pod.ObjectMeta.Name, ca.Pod.ObjectMeta.Namespace)
	// Case pod deleted before checked or apiserver connect issue --> just return 200
	if err != nil {
		fmt.Println(err.Error())
		r1, _ := regexp.Compile("not found")
		if r1.Match([]byte(err.Error())) {
			resp = "true"
		} else {
			resp = "false"	
		}
		return
	}

	fmt.Println("====================")
	fmt.Println(pod.ObjectMeta.Name)
	fmt.Println(pod.ObjectMeta.Namespace)
	fmt.Println(ca.Node)
	fmt.Println("====================")

	if ! r.Match([]byte(ca.Node)) {
		// if pod's NodeName as the same nodename from request --> return false  else process
		nResource.NodeRequest(Client, pod.ObjectMeta.Name, pod.ObjectMeta.Namespace, "", ca.Node)
		
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

	} else {
		// Case template. Checking spot instance for scaling up/down instead return true
		// use spot instance when cost okie , capacity okie, balance okie
		if _, ok := ca.Labels[Config.SpotLabel]; ok {
			fmt.Println("===== ca labels ====")
			fmt.Println(ca.Node)
			fmt.Println(ca.Labels)
			fmt.Println("===== ca labels ====")
			if Config.SpotEnable {
				var instanceZone string
				var instanceType string
				if Config.CloudProvider == "aws" {
					instanceZone = ca.Labels["failure-domain.beta.kubernetes.io/zone"]
					instanceType = ca.Labels["beta.kubernetes.io/instance-type"]
				} else {
					instanceZone = ca.Labels["failure-domain.beta.kubernetes.io/zone"]
					instanceType = ca.Labels["beta.kubernetes.io/instance-type"]
				}

				spotPrice := Config.GetSpotPrice(instanceZone, instanceType)
				if ok := Config.SpotPriceCheckScaleUp(spotPrice, instanceType); ! ok {
					resp = "false"
				}
			}
		}
	}

	if Config.SpotReserveLabel != "" {
		if _, ok := pod.ObjectMeta.Labels[Config.SpotReserveLabel]; ok {
			// pod scheduled to SpotReserve, if node is not SpotReserve, do not allow to schedule
			if _, ok := ca.Labels[Config.SpotReserveLabel]; ! ok {
				resp = "false"
			}
		} else {
			// pod not scheduled to SpotReserve, if node is SpotReserve, do not allow to schedule
			if _, ok := ca.Labels[Config.SpotReserveLabel]; ok {
				resp = "false"
			}				
		}
	}

	c.String(200, resp)
}

func Ping(c *gin.Context) {
	c.String(200, "pong")
}

func (config *ConfigInfo) ConfigParse(filepath string) error {
	yamlFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, config)
	return err
}

func (config *ConfigInfo) ConfigReload(configFile string) {
	watcher, _ := fsnotify.NewWatcher()

	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				fmt.Println("ConfigFile change:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					var configTmp *ConfigInfo
					if err := configTmp.ConfigParse(configFile); err == nil {
						config.ConfigParse(configFile)
					}
				}
			case err := <-watcher.Errors:
				fmt.Println(err.Error())
			}
		}
	}()

	err := watcher.Add(configFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	<-done
}

func main() {

	// test_mode := flag.Bool("test_mode", false, "enable test mode")
	// role_check := flag.Bool("role_check", false, "enable role check default: false") 
	// prometheus_server := flag.String("prometheus_server", "http://prometheus-prometheus-server.devops.svc.cluster.local", "prometheus server for query metrics")
	// cpu_threshold := flag.Int64("cpu_threshold", 60, "cpu threshold per node to make schedule decision")
	// memory_threshold := flag.Int64("memory_threshold", 70, "memory threshold per node to make schedule decision")
	// load_threshold := flag.Float64("load_threshold", 0, "load avg threshold per node to make schedule decision")
	// cpu_idle_threshold := flag.Float64("cpu_idle_threshold", 0, "cpu idle threshold per node to make schedule decision")
	// spot_enable := flag.Bool("spot_enable", false, "enable spot instance")
	configfile := flag.String("configfile", "", "specify config file path")
	// spot_label := flag.String("spot_label", "spot.instance", "specify k8s spot node labels for detecting spot node, example: spot.instance . If this key exist, this is spot node")
	flag.Parse()

	// Config.Threshold.Cpu = *cpu_threshold
	// Config.Threshold.Memory = *memory_threshold
	// Config.Threshold.Load = *load_threshold
	// Config.Threshold.CpuIdle = *cpu_idle_threshold

	// Config.PrometheusServer = *prometheus_server
	// Config.TestMode = *test_mode
	// Config.RoleCheck = *role_check
	// Config.SpotEnable = *spot_enable
	// Config.SpotLabel = *spot_label

	if err := Config.ConfigParse(*configfile); err != nil {
		panic(err)
	} else {
		fmt.Println(Config)
	}

	go Config.ConfigReload(*configfile)

	go Config.AWSSpotPricing()
	go Config.AWSasgAutoDiscovery()

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err.Error())
		if ! Config.TestMode {
			panic(err)
		}
	}

	if ! Config.TestMode {
		Client = clientset.NewForConfigOrDie(config)
	}
	
	r := gin.Default()
	r.POST("v1/scheduler", SchedulerFunc)
	// r.POST("v1/bind", FinalScheduleResult)
	// r.POST("v1/spotupdate", SpotUpdateFunc)
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
