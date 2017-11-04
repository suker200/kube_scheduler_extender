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

# Feature:
- we use default scheduler extender docker image, so keep all default scheduler features in our custom scheduler
- we add custom scheduler extender to our custom scheduler

# Usage:
- Build docker image

```
glide up --strip-vendor

CGO_ENABLED=0 env GOOS=linux go build

```

- Update helm chart (We using helm chart for deploying application)

# Note: 
- If you are using cluster-autoscaler, enable below option, to make sure cluster-autoscaler respect your custom-scheduler request, because the cluster-autoscaler have their node check before scaling node.

```
--verify-unschedulable-pods=false
```

- we can test with minikube :) 