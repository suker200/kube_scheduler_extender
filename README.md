# Kubernetes Custom Scheduler Extender

# Production READY

# Requirement:

 - k8s 1.7+

# Reason:
 - We face the issue that our application always peak the resource using, so the OOM Killed happens frequently, so we will configure the Limit resource x1,5 or x2 times with request config hence the pod can peak when necessary. When the limit can higher than the node resource which make the node overload when too much pods were assigned on.
 - We want control the resource usage per node ( our rule 60% resource usage is good)
 
# Target:
- kubernetes scheduler will not assign new pod to nodes which have request resource > $X % (default: CPU 60% , Mem 70%). We save 40% for application peak
- Do not schedule pod to server with low CPU Idle
- Do not schedule pod to server with high load

# Expand:
- We can add more filter in pod schedule with scheduler extender
- We intend to attach cluster autoscaler same stack with scheduler which improve stability and perforamnce.
- We choose the spot instance which has smallest price (support feature balance ASG when enable this option)

# Feature:
- we use default scheduler extender docker image, so keep all default scheduler features in our custom scheduler
- we add custom scheduler extender to our custom scheduler
- If prometheus server is not reachable, we just check node role and request/limit resource and return the result (timeout 10s). We have add prometheus to the same deployment with custom_scheduler (helm chart) with retention = 30mins, so we think that's okie for small prometheus. Improving and stabling scheduler feature.
- Schedule priority order: SpotInstance - onDemand

# Cluster Autoscaler support
Currently, we're using https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler, and this haven't supported custom scheduler/extender, so, I make a fork this with custom_scheduler support feature flag. (https://github.com/suker200/autoscaler/tree/custom_scheduler_extender)

 Flow predicate rquest api:
	- check current pod's node
		+ pod's node emtpy --> process
		+ pod's node not empty: if same (nodeinfo from request ) --> return false  else process

# Spot Instance
- when schedule pod to SpotInstance ?
	+ group node from scheduler: pick spot instance only if exist, remove ondemand else default 
- when scaling up SpotInstance node ?
	+ phase 1: spot instance highest priority if under maxpricescaleup
	+ phase 2: spot instance group rebalance

# Usage:
- Build docker image

```
glide up --strip-vendor

CGO_ENABLED=0 env GOOS=linux go build

```

- Update helm chart (We using helm chart for deploying application)

# Note: 
- If you are using cluster-autoscaler
	+ cluster autoscaler <= 0.6.x enable below option, to make sure cluster-autoscaler respect your custom-scheduler request, because the cluster-autoscaler have their node check before scaling node. (--verify-unschedulable-pods=false)
	+ cluster autoscaler > 0.6.x, please use my custom cluster-autoscaler which support "--custom-scheduler" for example
```
	AWS_REGION=us-east-1 ./cluster-autoscaler --cloud-provider=aws --node-group-auto-discovery=asg:tag=xxx --alsologtostderr --logtostderr --namespace=testing --v=4 --stderrthreshold=info --skip-nodes-with-local-storage=false --custom-scheduler=http://127.0.0.1:12345/v1/ca 
```

# Test
- Requirement:
	+ virtualbox
	+ minikube
	+ kubectl
	+ helm
	
- run: cd scheduler_test && sh -x test.sh

- we can test with minikube bootstrap kubeadm :) 