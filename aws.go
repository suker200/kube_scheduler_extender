package main

import(
	"github.com/aws/aws-sdk-go/service/ec2"
    "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/client_golang/prometheus"
	"time"
	"fmt"
	"strconv"
)

func (config ConfigInfo) getSpotPricing(sess *session.Session, timeCheck time.Time, instanceType string) {
	svc := ec2.New(sess)
	for _, zone := range config.Zones {
		input := &ec2.DescribeSpotPriceHistoryInput{
			AvailabilityZone: aws.String(zone),
		    InstanceTypes: []*string{
		        aws.String(instanceType),
		    },
		    ProductDescriptions: []*string{
		        aws.String("Linux/UNIX"),
		    },
		    MaxResults: []*int64{
		    	aws.Int64(1),
		    }[0],
		    StartTime: &timeCheck,
		    EndTime: &timeCheck,
		}

		result, err := svc.DescribeSpotPriceHistory(input)
		if err == nil {
			pushMetric(result.SpotPriceHistory[0])
			// ch <- result.SpotPriceHistory[0]
		} else {
			fmt.Println(err.Error())
		}
	}
}

// func collector(ch chan *ec2.SpotPrice) {
// 	for {
// 		data := <- ch
// 		go pushMetric(data)
// 		// fmt.Println(data)
// 	}
// }

func (config ConfigInfo) SpotPricing() {
	// var InstanceTypes []string
    InstanceTypes := make([]string, 0, len(config.SpotInfo))
    for instance := range config.SpotInfo {
        InstanceTypes = append(InstanceTypes, instance)
    }


	for {

		sess, err := session.NewSession(&aws.Config{
	    	Region: aws.String(config.Region)},
		)

		if err == nil {
			timeCheck := time.Now().UTC()
			for _, instance := range InstanceTypes {
				go config.getSpotPricing(sess, timeCheck, instance)	
			}
		}

		time.Sleep(time.Duration(60) * time.Second)
	}
}

func pushMetric(spotPrice *ec2.SpotPrice) {
	gatewayUrl:="127.0.0.1:9091/"
	
	throughputGuage := prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "SpotPrice",
			Help: "SpotPrice in Second",
	})
	
	price, err := strconv.ParseFloat(*spotPrice.SpotPrice, 64)
	if err != nil {
		fmt.Println("Failed to convert price string to float ", err)
		return
	}

	throughputGuage.Set(price)
	 
	if err := push.Collectors(
			"SpotPrice", map[string]string{
				"zone": *spotPrice.AvailabilityZone,
				"instanceType": *spotPrice.InstanceType,
			},
			gatewayUrl, throughputGuage); err != nil {
		fmt.Println("Could not push completion time to Pushgateway:", err)
	}
}

// func (config ConfigInfo) SpotPricing() {
// 	// var ch = make(chan *ec2.SpotPrice)
// 	// go collector()
// 	worker()
// }

