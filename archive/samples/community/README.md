# Jaeger Operator Resource Mapping

"jaeger_operator_resource_mapping_sample_cr.yaml" provides a sample ORM template so that [Kubeturbo](https://github.com/turbonomic/kubeturbo) can control the resources of [Jaeger](https://www.jaegertracing.io/) managed by [Jaeger Operator](https://github.com/jaegertracing/jaeger-operator)

## Getting started

### Install Jaeger from Jaeger Operator
Follow the steps mentioned in https://github.com/jaegertracing/jaeger-operator#getting-started.

### Deploy ORM for Jaeger Operator
1. Create ORM CRD in Turbonomic cluster if not exists:
```bash
kubectl apply -f https://raw.githubusercontent.com/turbonomic/orm/master/config/crd/bases/turbo_operator_resource_mapping_crd.yaml
```
2. Deploy "jaeger_operator_resource_mapping_sample_cr.yaml" in the same namespace Jaeger is deployed:
```bash
kubectl -n <jaeger_namespace> apply -f https://raw.githubusercontent.com/turbonomic/orm/master/config/samples/jaeger_orm/jaeger_operator_resource_mapping_sample_cr.yaml
```