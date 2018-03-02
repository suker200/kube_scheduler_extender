package main

import (
	k8sapiV1 "k8s.io/api/core/v1"
	"strconv"
	"time"
	"fmt"
)

func (config *ConfigInfo) CheckSpotNode(node k8sapiV1.Node) bool {
	if len(node.ObjectMeta.Labels) != 0 {
		if _, ok := node.ObjectMeta.Labels[config.SpotLabel]; ok {
			return true
		}
		return false
	} 
	return false
}

func (config *ConfigInfo) GetSpotPrice(zone, instanceType string) float64 {
	var spot_metrics Resp_PrometheusMetrics
	var err error
	spot_metrics = Get_PrometheusMetrics(config.PrometheusServer + "/api/v1/query?query=avg%20by%20(zone)%20(SpotPrice%7BinstanceType%3D%22" + instanceType + "%22%2C%20zone%3D%22" + zone + "%22%2Ctype%3D%22CurrentPrice%22%7D)")
	var spot_price float64
	for _, value := range spot_metrics.Data.Result {
		spot_price, err = strconv.ParseFloat(value.Value[1].(string), 64)
		if err != nil {
			spot_price = 0
		}
	}

	return spot_price
}

func (config *ConfigInfo) SpotPriceCheckScaleUp(spotPrice float64, instanceType string) bool {
	if spotPrice > config.SpotInfo[instanceType].MaxPriceScaleUP {
		return false
	}
	return true
}

func (config *ConfigInfo) SpotPriceCheckScaleDown(spotPrice float64, instanceType string) bool {
	if spotPrice < config.SpotInfo[instanceType].PriceScaleDOWN {
		return false
	}
	return true
}

// // We trigger action when there has something happens with our Happy Polla spot instance
// func (config ConfigInfo) SpotActionHub(c chan SpotStatus) {
// 	for {
// 		msg := <- c

// 	}
// }

// We will query prometheus for average price in 5m period. If > PriceScaleDOWN , Trigger prepare for interuption
func (config *ConfigInfo) SpotDetectHighPrice() {
	for {
		var spot_price_metric Resp_PrometheusMetrics
		spot_price_metric = Get_PrometheusMetrics(config.PrometheusServer + "/api/v1/query?query=avg_over_time(SpotPrice%7Btype%3D%22CurrentPrice%22%7D%5B5m%5D)")
		// spot_prices := make(map[string]float64)
		// core_dict := make(map[string]float64)

		for _, value := range spot_price_metric.Data.Result {
			spotPrice, err := strconv.ParseFloat(value.Value[1].(string), 64)
			if err != nil {
				continue
			}
			instanceType := value.Metric.InstanceType
			// zone := value.Metric.Zone
			if ok := config.SpotPriceCheckScaleDown(spotPrice, instanceType); ok {
				// Do something for preparing scaledown
				fmt.Println("Do something for preparing scaledown")
			}
		}
		time.Sleep(time.Duration(60) * time.Second)
	}
}
