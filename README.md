# Operator Resource Mapping
[![GoDoc](https://godoc.org/github.com/turbonomic/orm?status.svg)](https://godoc.org/github.com/turbonomic/orm)
[![License](https://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Operator Resource Mapping](#operator-resource-mapping)
  - [Overview](#overview)
  - [QuickStart](#quickstart)
  - [Architecture](#architecture)
    - [Simple Implementation](#simple-implementation)
    - [Extensions](#extensions)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->


## Overview

Operator Resource Mapping (ORM) is a mechanism to allow assets like [Kubeturbo](https://github.com/turbonomic/kubeturbo/wiki) to manage resources in an Operator managed Kubernetes cluster, for example to [vertically scale containers](https://github.com/turbonomic/kubeturbo/wiki/Action-Details#resizing-vertical-scaling-of-containerized-workloads) or [horizontally scale pods](https://github.com/turbonomic/kubeturbo/wiki/Action-Details#slo-horizontal-scaling-private-preview).

ORM works at operand basis, user defines which operand holds the source of trueth of which resources and automatically coordinate resource changes.

## QuickStart

ORM leverages operator sdk to create/build project, follow the standard operator sdk approach to run it locally or generate images to deploy to a target cluster with right RBAC settings.

## Architecture

ORM introduces Custom Resource Definition (CRD) for users to define target operand, patterns and report status of the mapping (to be) done by orm. There are two types of the controllers working with ORM Custom Resources(CR)s: Enforcer and Mapper.

![image](docs/images/basic.png)

Mapper reads the Patters defined in ORM CR spec, translate it to actual mapping and monitor the target resource to update the value dynamically into ORM CR status.

Enforcer read the mapping in ORM CR status and enforce it to the operand defined in ORM CR spec. There various mode to enforce the change: none, once, always. 

- none: orm is for advisory only, no changes will be applied to operand
- once: orm apply the change only once when the mapping is created/updated, after that operand could be changed by others
- always: orm monitor the operand resource defined in ORM CR and ensure the target fields are updated as indicated in ORM

### Simple Implementation

A simple implementation of mapper and enforcer is provided for users to develop and test their ORMs. In this simple implementation, 

- simple-mapper watches the actual resource managed by the operand and use the changes outside operator controller (via managedFields) to 
fill the ORM CR
- simple-enforcer update the operand from ORM CR status

### Extensions

Mapper and Enforcer could be extended for complex or simply production use cases. Mapper could talk with 3rd party "brain" to decide what to be changed; wihle Enforcer could be repurposed to chase the source of trueth and take actions other than updating a resource in kubernetes.
