# Operator Resource Mapping

## Table of Contents

* [What is ORM ](#ORMDef)
* [Problem](#Problem)
* [Solution](#Solution)
* [Architecture](#Architecture)
* [Steps to Deploy ORM](#Steps_to_deploy)

## <a id="ORMDef"></a>What is ORM?

Operator Resource Mapping (ORM) is a mechanism to allow [Kubeturbo](https://github.com/turbonomic/kubeturbo/wiki) to manage resources in an Operator managed Kubernetes cluster, for example to [vertically scale containers](https://github.com/turbonomic/kubeturbo/wiki/Action-Details#resizing-vertical-scaling-of-containerized-workloads) or [horizontally scale pods](https://github.com/turbonomic/kubeturbo/wiki/Action-Details#slo-horizontal-scaling-private-preview).

## <a id="Problem"></a>Problem

The lifecycle of micro-service based application is managed by Operator, this uses declarative approach to manage the desired state of an application (like Pod replicas, memory limits, etc.). Any Turbo actions to manage the size of a workload controller like Deployment, either vertically or horizontally will be reverted by the Operator.

To Illustrate the problem, lets consider two scenarios when executing a container resizing action in kubetrubo with operator and without operator:

### Container Resize without Operator

<img src="https://github.com/SumanthKodali999/images/blob/main/resize_without_operator.png" width="700"/>

The update in the deployment will be successful.

### Container Resize with Operator

<img src="https://github.com/SumanthKodali999/images/blob/main/Resize_with_operator.png" width="700"/>

The update in the Deployment will be reverted by the Operator because the CR is the source of truth for resource values.

## <a id="Solution"></a>Solution

### Update the Source of Truth

The solution is to update the source of truth in the CR when executing resizing actions if controlled by the Operator so that Operator update the deployment to keep the resource values in sync.

<img src="https://github.com/SumanthKodali999/images/blob/main/ORM_Solution_with_operator.png" width="700"/>

To achieve this, we introduce a new CustomResource which helps Kubeturbo to identify the correct path in CR to update the values.

## <a id="Architecture"></a>Architecture

ORM introduces Custom Resource Definition(CRD) for users to define target owner, patterns and report status of the mapping.

<img src="https://github.com/SumanthKodali999/images/blob/main/KubeTurbo_ORM_Arch.png" width="700"/>

- KubeTurbo does have capability to read the pattern defined in ORM CR spec, translate it to actual mapping and monitor the target resource to update the value dynamically into ORM CR status.
- It also reads the mapping in ORM CR status and enforce it to the owner defined in the ORM CR spec.
- There are multiple ways to enforce the change: none, once, always
  - none: orm is for advisory only, no changes will be applied to owner
  - once: orm apply the change only once when the mapping is created/updated, after that owner could be changed by others
  - always: orm monitor the owner resource defined in ORM CR and ensure the target fields are updated as indicated in ORM

## <a id="Steps_to_deploy"></a>Steps to Deploy ORM

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

