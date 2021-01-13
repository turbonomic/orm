# Operator Resource Mapping
Operator Resource Mapping (ORM) is a mechanism to allow [Kubeturbo](https://github.com/turbonomic/kubeturbo) to manage resources in an Operator managed Kubernetes cluster, for example to vertically scale containers or horizontally scale pods.

Here’re the steps to deploy it:
1. Create ORM CRD to the target XL cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/turbonomic/orm/master/config/crd/bases/turbo_operator_resource_mapping_crd.yaml
```
2. Create XL ORM CR to each tenant namespace in XL cluster:
```
kubectl -n turbonomic apply -f https://raw.githubusercontent.com/turbonomic/orm/master/config/samples/turbo_operator_resource_mapping_sample_cr.yaml
```
3. Rediscover Kubeturbo target from Turbonomic UI and NO need to restart the corresponding Kubeturbo pod in XL cluster. ORM CR will be successfully discovered when you see a log message from Kubeturbo like this:
```
I0806 14:08:56.506469       1 k8s_discovery_client.go:271] Discovered 1 Operator managed Custom Resources in cluster Kubernetes-Turbonomic.
```
