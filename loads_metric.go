package main

import (
	// "net/url"
	"strconv"
)

func Loads_metric() map[string]float64 {
	// var metrics Resp_PrometheusMetrics
	// metrics = Get_PrometheusMetrics(Config["prometheus_server"] + "/api/v1/query?query=node_load1{}")
	// load_dict := make(map[string]float64)
	// for _, value := range metrics.Data.Result {
	// 	metric, err := strconv.ParseFloat(value.Value[1].(string), 64)
	// 	name := Convert_Name(value.Metric.Instance)
	// 	if err != nil {
	// 		load_dict[name] = 0
	// 	} else {
	// 		load_dict[name] = metric
	// 	}
	// }

	var load_metrics Resp_PrometheusMetrics
	var cpu_core_metrics Resp_PrometheusMetrics
	load_metrics = Get_PrometheusMetrics(Config["prometheus_server"] + "/api/v1/query?query=node_load5{}")
	cpu_core_metrics = Get_PrometheusMetrics(Config["prometheus_server"] + "/api/v1/query?query=kube_node_status_capacity_cpu_cores{}")
	load_dict := make(map[string]float64)
	// core_dict := make(map[string]float64)

	for num, value := range load_metrics.Data.Result {
		load, err := strconv.ParseFloat(value.Value[1].(string), 64)
		core, err := strconv.ParseFloat(cpu_core_metrics.Data.Result[num].Value[1].(string), 64)
		name := Convert_Name(value.Metric.Instance)
		if err != nil {
			load_dict[name] = 0
		} else {
			load_dict[name] = load - core
		}
	}

	return load_dict
	// fmt.Println(load_dict)
}
