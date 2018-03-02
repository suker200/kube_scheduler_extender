package main

import(
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
    "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/client_golang/prometheus"
	"time"
	"fmt"
	"strconv"
)

const (
	gatewayUrl = "127.0.0.1:9091/"
)

func (config *ConfigInfo) getSpotPricing(sess *session.Session, timeCheck time.Time, instanceType string) {
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
			func (spotPrice *ec2.SpotPrice) {
				throughputGuage := prometheus.NewGauge(prometheus.GaugeOpts{
						Name: "SpotPrice", //"SpotPrice",
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
							"type": "CurrentPrice",
						},
						gatewayUrl, throughputGuage); err != nil {
					fmt.Println("Could not push completion time to Pushgateway:", err)
				}
			}(result.SpotPriceHistory[0])

			func (spotPrice *ec2.SpotPrice) {
				throughputGuage := prometheus.NewGauge(prometheus.GaugeOpts{
						Name: "SpotPrice", //"SpotPrice",
						Help: "SpotPrice in Second",
				})
				
				if err != nil {
					fmt.Println("Failed to convert price string to float ", err)
					return
				}

				throughputGuage.Set(config.SpotInfo[instanceType].MaxPrice)
				 
				if err := push.Collectors(
						"SpotPrice", map[string]string{
							"zone": *spotPrice.AvailabilityZone,
							"instanceType": *spotPrice.InstanceType,
							"type": "MaxPrice",
						},
						gatewayUrl, throughputGuage); err != nil {
					fmt.Println("Could not push completion time to Pushgateway:", err)
				}
			}(result.SpotPriceHistory[0])
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

func (config *ConfigInfo) AWSSpotPricing() {
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

func describeInstance(sess *session.Session, instanceIds []string) (*ec2.DescribeInstancesOutput, error) {
	svc := ec2.New(sess)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	}

	return svc.DescribeInstances(input)
}

func validRequestSpotCheck(spotInstances []*ec2.SpotInstanceRequest) []string {
	var instancesRecycle []string
	for _, instance := range spotInstances {
		fmt.Println(*instance.InstanceId)
		i := *instance.InstanceId
		current := time.Now().UTC()
		if instance.ValidUntil != nil {
			twohour_beforeValidEnd := instance.ValidUntil.Add(-60 * time.Hour)
			if twohour_beforeValidEnd.Before(current) {
				instancesRecycle = append(instancesRecycle, i)
				fmt.Println("twohour_beforeValidEnd > current, we need draining " +  i)
			} 			
		}
	}
	return instancesRecycle
}

func describeSpotInstance(sess *session.Session, instanceIds []string) {
	svc := ec2.New(sess)
	input := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-id"),
				Values: aws.StringSlice(instanceIds),
			},
		},
		// SpotInstanceRequestIds: aws.StringSlice(instanceIds),
	}
	result, err := svc.DescribeSpotInstanceRequests(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}			
	} else {
		// fmt.Println(result)
		instancesRecycle := validRequestSpotCheck(result.SpotInstanceRequests)
		if len(instancesRecycle) != 0 {
			result, err := describeInstance(sess, instancesRecycle)
			if err != nil {
				fmt.Println("Do something for retry in short time: " + err.Error())
			} else {
				// fmt.Println(result.Reservations)
				var instancesRecycle []string
				for _, v := range result.Reservations {
					for _, i := range v.Instances {
						instancesRecycle = append(instancesRecycle, *i.PrivateDnsName)
					}
				}
				fmt.Println(instancesRecycle)
			}
		}
	}
}

func asgPushMetric(metricName, asgName string, asgInfo map[string]int64) {
	for k, v := range asgInfo {
		throughputGuage := prometheus.NewGauge(prometheus.GaugeOpts{
				Name: metricName, //"SpotPrice",
				Help: "AWS AutoscalingGroups info",
		})
	
		throughputGuage.Set(float64(v))
		 
		if err := push.Collectors(
				metricName, map[string]string{
					"asgName": asgName,
					"type": k,
				},
				gatewayUrl, throughputGuage); err != nil {
			fmt.Println("Could not push completion time to Pushgateway:", err)
		}
	}
}

func asgDescribe(sess *session.Session, asgs []string) {
	svc := autoscaling.New(sess)

	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: aws.StringSlice(asgs),
	}

	result, err := svc.DescribeAutoScalingGroups(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			fmt.Println(aerr.Error())
		} else {
			fmt.Println(err.Error())
		}			
	} else {
		// fmt.Println(result)
		for _, asg := range result.AutoScalingGroups {
			var a = make(map[string]int64)
			var instanceids []string
			a["MaxSize"] = *asg.MaxSize
			a["MinSize"] = *asg.MinSize
			a["CurrentSize"] = *asg.DesiredCapacity

			go asgPushMetric("InstanceGroupSize", *asg.AutoScalingGroupName, a)

			for _, i := range asg.Instances {
				instanceids = append(instanceids, *i.InstanceId)
			}
			describeSpotInstance(sess, instanceids)
		}
	}
}

func (config *ConfigInfo) AWSasgAutoDiscovery() {
	for {
		sess, err := session.NewSession(&aws.Config{
	    	Region: aws.String(config.Region)},
		)

		if err == nil {
			svc := autoscaling.New(sess)
			input := &autoscaling.DescribeTagsInput{
				Filters: []*autoscaling.Filter{
					{
						Name: aws.String("key"),
						Values: aws.StringSlice(config.AsgDiscoveryTag),
					},
				},
			}

			result, err := svc.DescribeTags(input)	
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					fmt.Println(aerr.Error())
				} else {
					fmt.Println(err.Error())
				}
			} else {	
				var asgsMap = make(map[string][]string)
				var asgs []string
				for _, v := range result.Tags {
					asgsMap[*v.ResourceId] = append(asgsMap[*v.ResourceId], *v.Key)
				}
				for k, v := range asgsMap {
					if len(v) == 2 {
						asgs = append(asgs, k)
					}
				}
				asgDescribe(sess, asgs)
			}		
		}

		time.Sleep(time.Duration(60) * time.Second)
	}
}
// func (config ConfigInfo) pushMetric(metricName string, spotPrice *ec2.SpotPrice) {
// 	gatewayUrl:="127.0.0.1:9091/"
	
// 	throughputGuage := prometheus.NewGauge(prometheus.GaugeOpts{
// 			Name: metricName, //"SpotPrice",
// 			// Help: "SpotPrice in Second",
// 	})
	
// 	price, err := strconv.ParseFloat(*spotPrice.SpotPrice, 64)
// 	if err != nil {
// 		fmt.Println("Failed to convert price string to float ", err)
// 		return
// 	}

// 	throughputGuage.Set(price)
	 
// 	if err := push.Collectors(
// 			metricName, map[string]string{
// 				"zone": *spotPrice.AvailabilityZone,
// 				"instanceType": *spotPrice.InstanceType,
// 			},
// 			gatewayUrl, throughputGuage); err != nil {
// 		fmt.Println("Could not push completion time to Pushgateway:", err)
// 	}
// }

// // func (config ConfigInfo) SpotPricing() {
// // 	// var ch = make(chan *ec2.SpotPrice)
// // 	// go collector()
// // 	worker()
// // }

