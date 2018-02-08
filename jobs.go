package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)


func Get_PrometheusMetrics(url string) Resp_PrometheusMetrics {
	var metrics Resp_PrometheusMetrics
	client := &http.Client{
		Timeout: time.Duration(120 * time.Second),
	}

	resp, err := client.Get(url)

	if err != nil {
		return metrics
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
			return metrics
		}
		if resp.StatusCode != 200 {
			fmt.Println("Alarm via telegram directly to devops, info: " + string(body))
			// return string(body), errors.New(string(body))
			return metrics
		}

		if err := json.Unmarshal(body, &metrics); err != nil {
			// AlarmMe(serviceName, nameSpace, "Failed", err.Error())
			fmt.Println(err.Error())
			return metrics
		}
		return metrics
	}
}

func Convert_Name(name string) string {
	name = strings.Replace(name, ".", "-", -1)
	name = strings.Replace(name, ":9100", ".ec2.internal", -1)
	name = "ip-" + name
	return name
}

