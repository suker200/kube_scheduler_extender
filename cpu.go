package main

import (
	"strconv"
)

func (config *ConfigInfo) Cpu_Idle() map[string]float64 {
	var cpu_metrics Resp_PrometheusMetrics
	cpu_metrics = Get_PrometheusMetrics(config.PrometheusServer + "/api/v1/query?query=avg+by+(instance)+(irate(node_cpu%7Bmode%3D%22idle%22%7D%5B5m%5D))+*+100")
	cpu_dict := make(map[string]float64)
	// core_dict := make(map[string]float64)

	for _, value := range cpu_metrics.Data.Result {
		cpu_idle, err := strconv.ParseFloat(value.Value[1].(string), 64)
		name := Convert_Name(value.Metric.Instance)
		if err != nil {
			cpu_dict[name] = 100
		} else {
			cpu_dict[name] = cpu_idle
		}
	}

	return cpu_dict
	// fmt.Println(load_dict)
}
