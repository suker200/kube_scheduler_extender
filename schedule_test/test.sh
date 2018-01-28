#!/bin/bash

minikube start --memory=4096 --kubernetes-version v1.8.3 --bootstrapper kubeadm

# this for loop waits until kubectl can access the api server that Minikube has created
for i in {1..150}; do # timeout for 5 minutes
	kubectl get po &> /dev/null
	if [ $? -ne 1 ]; then
		break
	fi
	
	sleep 2

	if [ $i -eq 150 ]; then
		echo "Minikube start timeout"
		exit 1
	fi
done

kubectl -n kube-system create sa tiller
kubectl create clusterrolebinding tiller --clusterrole cluster-admin --serviceaccount=kube-system:tiller
helm init --service-account tiller

for i in {1..150}; do
	helm list &> /dev/null
	if [ $? -ne 1 ]; then
		break
	fi
	
	sleep 2

	if [ $i -eq 150 ]; then
		echo "helm init failed"
		exit 1
	fi
done

helm upgrade -i --namespace=kube-system scheduler-extender  scheduler-extender

for i in {1..300}; do
	kubectl -n kube-system get po -l app=scheduler-extender-test | grep Running &> /dev/null
	if [ $? -ne 1 ]; then
		break
	fi
	
	sleep 2

	if [ $i -eq 300 ]; then
		echo "Deploy scheduler-extender-test failed"
		exit 1
	fi
done

kubectl apply -f nginx.yaml

for i in {1..300}; do
	kubectl -n devops get po -l app=nginx | grep Running &> /dev/null
	if [ $? -ne 1 ]; then
		break
	fi
	
	sleep 2

	if [ $i -eq 300 ]; then
		echo "Deploy nginx application failed"
		exit 1
	fi
done

for i in {1..10}; do
	curl -s -I $(minikube service -n devops nginx --url) | grep "HTTP/1.1 200 OK"
	if [ $? -eq 0 ]; then
		echo "[Without Prometheus] Test Successfully"
		break
	fi

	sleep 2

	if [ $i -eq 10 ]; then
		echo "Deploy nginx application failed"
		exit 1
	fi
done

# ### Test with prometheus
# kubectl delete -f nginx.yaml
# helm upgrade -i --namespace=devops prometheus prometheus

# for i in {1..300}; do
# 	kubectl -n devops get po -l app=prometheus -l component=server | grep Running &> /dev/null
# 	if [ $? -ne 1 ]; then
# 		kubectl -n devops get po -l app=prometheus -l component=node-exporter | grep Running &> /dev/null
# 		if [ $? -ne 1 ]; then
# 			break
# 		fi
# 	fi
	
# 	sleep 2

# 	if [ $i -eq 300 ]; then
# 		echo "Deploy prometheus failed"
# 		exit 1
# 	fi
# done

# for i in {1..120}; do
# 	echo "Sleep " $i
# done

# kubectl create -f nginx.yaml


# for i in {1..300}; do
# 	kubectl -n devops get po -l app=nginx | grep Running &> /dev/null
# 	if [ $? -ne 1 ]; then
# 		break
# 	fi
	
# 	sleep 2

# 	if [ $i -eq 300 ]; then
# 		echo "Deploy nginx application failed"
# 		exit 1
# 	fi
# done

# for i in {1..10}; do
# 	curl -s -I $(minikube service -n devops nginx --url) | grep "HTTP/1.1 200 OK"
# 	if [ $? -eq 0 ]; then
# 		echo "[With Prometheus] Test Successfully"
# 		break
# 	fi

# 	sleep 2

# 	if [ $i -eq 10 ]; then
# 		echo "Deploy nginx application failed"
# 		exit 1
# 	fi
# done


# #$(minikube service -n kube-system nginx-ingress-controller-proxy-protocol-test --url | cut -d '/' -f3)

