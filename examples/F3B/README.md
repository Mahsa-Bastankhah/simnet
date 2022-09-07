# F3B test

## Minikube

### Install Minikube
First you need to install openVPN,docker and VirtualBox, then you would be able to run the minikube commands
```bash 
# Start the cluster
minikube start --docker-opt bip=172.18.0.1/16 --driver=virtualbox
```

### Build the docker image 
```bash
# Run from Simnet directory
eval $(minikube docker-env)

cd $SIMNET_DIR
make build_monitor
make build_router
```

### Run the simulation
You can always change the docker image that is used by the test in the `main` function of `main.go`.
You also need to comment out the Kubernetes configuration and instead use the Minikube configuration in the `main` function of `main.go`
```bash
cd $SIMNET_DIR
make EXAMPLE="F3B" run # be patient, time for a coffee !
# or make run to run the default simulation.

# In case the simulation fails and cannot clean the cluster correctly, you
# can force the deletion.
make clean
```

## Kubernetes

### Run the simulation
You can always change the docker image that is used by the test in the `main` function of `main.go`.
You also need to comment out the Minikube configuration and instead use the Kubernetes configuration in the `main` function of `main.go`
```bash
cd $SIMNET_DIR
cd F3B
```
You can run the entire test all at once

```bash
go run main.go
```
Or you can do this step by step. First deploy the nodes which runs the `main` function and then run the `Execute` function.
 ```bash
# ... or do it step by step
go run main.go -do-deploy

# This can be done multiple times but be aware that statistics will be
# overwritten.
go run main.go -do-execute
```

In after finishing the test you should clean the deployments.
```bash
# Important to reduce the cost
go run main.go -do-clean
```

In case the above command raised an error you can manually delete the deployment.
```bash
microk8s kubectl delete deployment --all
microk8s kubectl delete service simnet-router
```
### Logs
List all the deployment and see their status.
```bash
microk8s kubectl get all --all-namespaces
```

You can get the logs of a specific deployment
```bash
microk8s kubectl logs <pod_name>
```
