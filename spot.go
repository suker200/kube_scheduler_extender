package main

import (
	k8sapiV1 "k8s.io/api/core/v1"
	"strconv"
	// "fmt"
)

func (config ConfigInfo) CheckSpotNode(node k8sapiV1.Node) bool {
	if len(node.ObjectMeta.Labels) != 0 {
		if _, ok := node.ObjectMeta.Labels[config.SpotLabel]; ok {
			return true
		}
		return false
	} 
	return false
}

func (config ConfigInfo) GetSpotPrice(zone, instanceType string) float64 {
	var spot_metrics Resp_PrometheusMetrics
	var err error
	spot_metrics = Get_PrometheusMetrics(config.PrometheusServer + "/api/v1/query?query=avg%20by%20(zone)%20(SpotPrice%7BinstanceType%3D%22" + instanceType + "%22%2C%20zone%3D%22" + zone + "%22%7D)")
	var spot_price float64
	for _, value := range spot_metrics.Data.Result {
		spot_price, err = strconv.ParseFloat(value.Value[1].(string), 64)
		if err != nil {
			spot_price = 0
		}
	}

	return spot_price
}

func (config ConfigInfo) SpotPriceCheckScaleUp(spotPrice float64, instanceType string) bool {
	if spotPrice > config.SpotInfo[instanceType].MaxPriceScaleUP {
		return false
	}
	return true
}

func (config ConfigInfo) SpotPriceCheckScaleDown(spotPrice float64, instanceType string) bool {
	if spotPrice < config.SpotInfo[instanceType].PriceScaleDOWN {
		return false
	}
	return true
}