# Operator Resource Mapping
Operator Resource Mapping (ORM) is a mechanism to allow [Kubeturbo](https://github.com/turbonomic/kubeturbo/wiki) to manage resources in an Operator managed Kubernetes cluster, for example to [vertically scale containers](https://github.com/turbonomic/kubeturbo/wiki/Action-Details#resizing-vertical-scaling-of-containerized-workloads) or [horizontally scale pods](https://github.com/turbonomic/kubeturbo/wiki/Action-Details#slo-horizontal-scaling-private-preview).

Here’re the steps to deploy it:
1. Create the ORM Customer Resource Definition (CRD) in the kubernetes cluster (where kubeturbo is also running):
```bash
kubectl apply -f https://raw.githubusercontent.com/turbonomic/orm/master/config/crd/bases/turbo_operator_resource_mapping_crd_v1.yaml
```
>This CRD supports kubnernetes 1.16 and higher.
2. Next deploy the ORM Custom Resource (CR) for your application in the namespace of that app. Sample CRs are located [here](https://github.com/turbonomic/orm/tree/master/config/samples). In our example, to allow for resizing of Turbonomic Server app services, we will deploy the Turbonomic XL ORM CR into the namespace where the Turbonomic Server is running:
```
kubectl -n turbonomic apply -f https://raw.githubusercontent.com/turbonomic/orm/master/config/samples/turbo_operator_resource_mapping_sample_cr.yaml
```
3. Rediscover Kubeturbo target from Turbonomic UI and NO need to restart the corresponding Kubeturbo pod in cluster. ORM CR will be successfully discovered when you see a log message from Kubeturbo like this:
```
I0806 14:08:56.506469       1 k8s_discovery_client.go:271] Discovered 1 Operator managed Custom Resources in cluster Kubernetes-Turbonomic.
```
