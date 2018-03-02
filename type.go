package main

import (
	"k8s.io/apimachinery/pkg/api/resource"
	k8sapi "k8s.io/kubernetes/pkg/api"
	k8sapiV1 "k8s.io/api/core/v1"
)

type ConfigInfo struct {
	Threshold	ThreshodInfo `yaml:"threshold_config"`
	PrometheusServer string `yaml:"prometheus_server"`
	TestMode bool	`yaml:"test_mode"`
	CloudProvider string `yaml:"cloud_provider"`
	Region	string `yaml:"region"`
	Zones	[]string `yaml:"zones"`
	RoleCheck bool `yaml:"role_check"`
	SpotEnable bool `yaml:"spot_enable"`
	SpotLabel string `yaml:"spot_label"`
	SpotReserveLabel string `yaml:"spot_reserve_label"`
	SpotInfo map[string]SpotConfig `yaml:"spot_config"`
	SpotDemandBalance	int `yaml:"spotdemandbalance"`
	AsgDiscoveryTag []string `yaml:"asg_auto_discovery_tag"`
}

type NodeResource struct {
	Name          string
	Schedule      bool
	CpuReqs       resource.Quantity
	CpuLimits     resource.Quantity
	MemoryReqs    resource.Quantity
	MemoryLimits  resource.Quantity
	Allocate      k8sapi.ResourceList
	Load          float64
	FailedMessage string
	Threshold     ThreshodInfo
}

type ThreshodInfo struct {
	Cpu    int64 `yaml:"cpu"`
	Memory int64 `yaml:"memory"`
	Load   float64 `yaml:"load"`
	CpuIdle float64 `yaml:"cpuidle"`
}

type Resp_PrometheusMetrics struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Metric struct {
				Instance string `json:"instance"`
				InstanceType string `json:"instanceType"`
				Zone 	string `json:"zone"`
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type ScaleCA struct {
	Pod k8sapiV1.Pod `json:"pod"`
	// Node k8sapiV1.Node `json:"node"`
	Node string `json:"node"`
	Labels map[string]string `json:"labels"`
}

type SpotConfig struct {
	MaxPrice float64 `yaml:"maxprice"`
	MaxPriceScaleUP float64 `yaml:"maxpricescaleup"`
	PriceScaleDOWN float64 `yaml:"pricescaledown"`
}

type SpotStatus struct {
	Zone string `json:"zone"`
	InstanceType string `json:"instanceType"`
	Message string `json:"message"`
	ID string `json:"id"`
}

type ASGInfo struct {
	Name string `json:"name"`
	MaxSize int64 `json:"maxsize"`
	MinSize int64 `json:"minsize"`
	CurrentSize int64 `json:"currentsize"`
	InstanceIDs []string `json:"instancesid"`
}