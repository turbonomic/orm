# Operator Resource Mapping with Redis 

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Prepare Redis Cluster](#prepare-redis-cluster)
- [Horizontal Scaling](#horizontal-scaling)
  - [Declare relationship between operator and owned resource](#declare-relationship-between-operator-and-owned-resource)
  - [Horizontal Pod AutoScaler](#horizontal-pod-autoscaler)
  - [Coordinate the changes](#coordinate-the-changes)
  - [Final Result](#final-result)
- [Vertical Scaling](#vertical-scaling)
  - [Prerequisite](#prerequisite)
  - [Vertical Pod AutoScaler](#vertical-pod-autoscaler)
  - [Coordinate the changes](#coordinate-the-changes-1)
  - [Final Result](#final-result-1)
- [Next Step](#next-step)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


This document describes how to use this project in Redis Operator with Auto Scalers. We need vpa and hpa generator controllers in utils folder.

## Prepare Redis Cluster

In order to show relationship between operator and the resource it manages we use Redis operator from [OT_CONTAINER-KIT](https://github.com/OT-CONTAINER-KIT/redis-operator#quickstart). We created redis cluster.

```shell
%helm list -A
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART                      APP VERSION
redis-cluster   ot-operators    1               2023-03-13 12:52:52.716665 -0400 EDT    deployed        redis-cluster-0.14.3       0.14.0     
redis-operator  ot-operators    1               2023-03-13 12:31:40.264923 -0400 EDT    deployed        redis-operator-0.14.3      0.14.0     

% kubectl get rediscluster -A
NAMESPACE      NAME            CLUSTERSIZE   LEADERREPLICAS   FOLLOWERREPLICAS   AGE
ot-operators   redis-cluster   3             3                3                  4h7m

% kubectl get sts -n ot-operators 
NAME                     READY   AGE
redis-cluster-follower   3/3     4h7m
redis-cluster-leader     3/3     4h8m
```

## Horizontal Scaling

The replicas of redis-cluster-leader is controlled by RedisCluster resource from Redis Operator

```yaml
% kubectl get rediscluster -n ot-operators   redis-cluster -o yaml
apiVersion: redis.redis.opstreelabs.in/v1beta1
kind: RedisCluster
metadata:
  annotations:
    meta.helm.sh/release-name: redis-cluster
    meta.helm.sh/release-namespace: ot-operators
    test: value
  name: redis-cluster
  namespace: ot-operators
...
spec:
  redisLeader:
    replicas: 3
...
```

### Declare relationship between operator and owned resource

We need the operatorresourcemapping resource defined in orm.yaml to declare the relationship

```yaml
% kubectl apply -f ./config/samples/redis-autoscaler/orm.yaml -o yaml
apiVersion: devops.turbonomic.io/v1alpha1
kind: OperatorResourceMapping
metadata:
  name: rediscluster
  namespace: ot-operators
...
spec:
  mappings:
    patterns:
    - owned:
        apiVersion: apps/v1
        kind: StatefulSet
        name: redis-cluster-leader
        path: .spec.replicas
      ownerPath: .spec.redisLeader.replicas
  owner:
    apiVersion: redis.redis.opstreelabs.in/v1beta1
    kind: RedisCluster
    name: redis-cluster
```

This ORM let our controllers coordinate changes to the operand. if our controller is running, you'll find the value in status.

### Horizontal Pod AutoScaler

Horizontal Pod AutoScaler is part of kubernetes, we just need to create the HPA resource for it

```yaml
% kubectl apply -f ./config/samples/redis-autoscaler/redis-hpa.yaml -o yaml
apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  name: redis-cluster-leader
  namespace: ot-operators
...
spec:
  maxReplicas: 5
  minReplicas: 1
  scaleTargetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: redis-cluster-leader
  targetCPUUtilizationPercentage: 80
status:
  currentReplicas: 0
  desiredReplicas: 0
```

Without our project, kubernetes attempts to modify the `spec.replicas` in StatefulSet with value from its `status.desiredReplicas`. This change will eventually be reverted by operator controller back to the value defined in RedisCluster resource.

### Coordinate the changes

Mke sure our controller is started, you'll find an AdviceMapping resource is generated automatically with the right operand as the owner in the mappings. 

<em>In this quick example, we create AdviceMapping for all values in desiredReplicas. The controller could be upgraded to generate AdviceMapping for non-zero desiredReplicas. </em>

```yaml
kubectl get am -n ot-operators -o yaml
apiVersion: v1
items:
- apiVersion: devops.turbonomic.io/v1alpha1
  kind: AdviceMapping
  metadata:
    name: redis-cluster-leader
    namespace: ot-operators
...
  spec:
    mappings:
    - advisor:
        apiVersion: autoscaling/v2
        kind: HorizontalPodAutoscaler
        name: redis-cluster-leader
        namespace: ot-operators
        path: .status.desiredReplicas
      target:
        apiVersion: apps/v1
        kind: StatefulSet
        name: redis-cluster-leader
        namespace: ot-operators
        path: .spec.replicas
  status:
    advices:
    - adviceValue:
        desiredReplicas: 0
      owner:
        apiVersion: redis.redis.opstreelabs.in/v1beta1
        kind: RedisCluster
        name: redis-cluster
        namespace: ot-operators
        path: .spec.redisLeader.replicas
      target:
        apiVersion: apps/v1
        kind: StatefulSet
        name: redis-cluster-leader
        namespace: ot-operators
        path: .spec.replicas
```

### Final Result

Now you'll find the RedisCluster is updated by our controller. Therefore `spec.replicas` in StatefulSet is adjusted by operator accordingly.

```yaml
% kubectl get rediscluster -n ot-operators   redis-cluster -o yaml
apiVersion: redis.redis.opstreelabs.in/v1beta1
kind: RedisCluster
metadata:
  annotations:
    meta.helm.sh/release-name: redis-cluster
    meta.helm.sh/release-namespace: ot-operators
    test: value
  name: redis-cluster
  namespace: ot-operators
...
spec:
  redisLeader:
    replicas: 3
...
```

```shell
kubectl get sts -A     
NAMESPACE      NAME                     READY   AGE
ot-operators   redis-cluster-follower   3/3     4h27m
ot-operators   redis-cluster-leader     0/0     4h27m
```

## Vertical Scaling

### Prerequisite

Unfortunately, redis operand does not have control points for container resources, so we have to disable the operator to avoid the resource to be reverted back to empty. 

```shell
% kubectl scale deploy -n ot-operators   redis-operator --replicas=0 
deployment.apps/redis-operator scaled
% kubectl get deploy -n ot-operators                                 
NAME             READY   UP-TO-DATE   AVAILABLE   AGE
redis-operator   0/0     0            0           5h41m
```

please also revert the `spec.redisLeader.replicas` back to 3, so that VPA has something to work with.


### Vertical Pod AutoScaler

We install Vertical Pod AutoScaler from [vpa git repo](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler#install-command)

After VPA is started

```shell
% kubectl get pods -A | grep vpa
kube-system    vpa-admission-controller-7c7666f6cd-4qt2z        1/1     Running   0               7h36m
kube-system    vpa-recommender-786476d7cc-l7vrs                 1/1     Running   0               7h36m
kube-system    vpa-updater-79d74db98b-9hmft                     1/1     Running   0               7h36m
```

Apply the vpa resource from this sample

```yaml
% kubectl apply -f config/samples/redis-autoscaler/redis-vpa.yaml -o yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: redis-vpa
  namespace: ot-operators
...
spec:
  targetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: redis-cluster-leader
  updatePolicy:
    updateMode: Auto
```

The VPA controller will provide recommendations shortly after 
```yaml
kubectl get vpa -A -o yaml
apiVersion: v1
items:
- apiVersion: autoscaling.k8s.io/v1
  kind: VerticalPodAutoscaler
  metadata:
    name: redis-vpa
    namespace: ot-operators
...
  spec:
    targetRef:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
    updatePolicy:
      updateMode: Auto
  status:
    recommendation:
      containerRecommendations:
      - containerName: redis-cluster-leader
        lowerBound:
          cpu: 12m
          memory: 131072k
        target:
          cpu: 12m
          memory: 131072k
        uncappedTarget:
          cpu: 12m
          memory: 131072k
        upperBound:
          cpu: 71m
          memory: 131072k
      - containerName: redis-exporter
        lowerBound:
          cpu: 12m
          memory: 131072k
        target:
          cpu: 12m
          memory: 131072k
        uncappedTarget:
          cpu: 12m
          memory: 131072k
        upperBound:
          cpu: 71m
          memory: 131072k
...
```

Without our project, in `Auto` mode, VPA controllers evict and update the Pod, but leave the spec in StatefulSet unchanged. In that case, things could be reverted by kubernetes controllers.

### Coordinate the changes

After our controller is started, an ActiveMapping resource is generated for those 2 containers' recommendations. And owners are route to the StatefulSet because there is no ORM describe the relationship to an operand. So our controllers will coordinate the `resource` back to StatefulSet itself.

```yaml
%kubectl get am -n ot-operators redis-vpa  -o yaml
apiVersion: devops.turbonomic.io/v1alpha1
kind: AdviceMapping
metadata:
  name: redis-vpa
  namespace: ot-operators
...
spec:
  mappings:
  - advisor:
      apiVersion: autoscaling.k8s.io/v1
      kind: VerticalPodAutoscaler
      name: redis-vpa
      namespace: ot-operators
      path: .status.recommendation.containerRecommendations[?(@.containerName=="redis-cluster-leader")].lowerBound
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-cluster-leader")].resources.requests
  - advisor:
      apiVersion: autoscaling.k8s.io/v1
      kind: VerticalPodAutoscaler
      name: redis-vpa
      namespace: ot-operators
      path: .status.recommendation.containerRecommendations[?(@.containerName=="redis-cluster-leader")].upperBound
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-cluster-leader")].resources.limits
  - advisor:
      apiVersion: autoscaling.k8s.io/v1
      kind: VerticalPodAutoscaler
      name: redis-vpa
      namespace: ot-operators
      path: .status.recommendation.containerRecommendations[?(@.containerName=="redis-exporter")].lowerBound
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-exporter")].resources.requests
  - advisor:
      apiVersion: autoscaling.k8s.io/v1
      kind: VerticalPodAutoscaler
      name: redis-vpa
      namespace: ot-operators
      path: .status.recommendation.containerRecommendations[?(@.containerName=="redis-exporter")].upperBound
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-exporter")].resources.limits
status:
  advices:
  - adviceValue:
      lowerBound:
        cpu: 12m
        memory: 131072k
    owner:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-cluster-leader")].resources.requests
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-cluster-leader")].resources.requests
  - adviceValue:
      upperBound:
        cpu: 71m
        memory: 131072k
    owner:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-cluster-leader")].resources.limits
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-cluster-leader")].resources.limits
  - adviceValue:
      lowerBound:
        cpu: 12m
        memory: 131072k
    owner:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-exporter")].resources.requests
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-exporter")].resources.requests
  - adviceValue:
      upperBound:
        cpu: 71m
        memory: 131072k
    owner:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-exporter")].resources.limits
    target:
      apiVersion: apps/v1
      kind: StatefulSet
      name: redis-cluster-leader
      namespace: ot-operators
      path: .spec.template.spec.containers[?(@.name=="redis-exporter")].resources.limits
```

### Final Result

Now you can find the changes are applied to StatefulSet accordingly:

```yaml
% kubectl get sts -n ot-operators   redis-cluster-leader -o yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis-cluster-leader
  namespace: ot-operators
  ownerReferences:
  - apiVersion: redis.redis.opstreelabs.in/v1beta1
    controller: true
    kind: RedisCluster
    name: redis-cluster
    uid: 8d7ebb93-fef3-4306-b1a1-d98ab4ba8180
  resourceVersion: "886798"
  uid: 72bd7910-0546-4ccf-9aa4-fa99c7beb304
spec:
  template:
    spec:
      containers:
    - name: redis-cluster-leader
        resources:
          limits:
            cpu: 62m
            memory: 131072k
          requests:
            cpu: 12m
            memory: 131072k   
    ...
    - name: redis-exporter
      resources:
        limits:
          cpu: 62m
          memory: 131072k
        requests:
          cpu: 12m
          memory: 131072k
...
```

## Next Step

Now you know how to use OperatorResourceMapping and AdviceMapping to coordinate changes to right resources, go ahead create your own mappings. 