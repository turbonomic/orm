# Compatibility

## Overview

Operator Resource Mapping(ORM) moved from turbonomic.com to devops.turbonomic.io and the schema has changed. In order to automatically upgrade old ORM resource, a compatibility controller is introduced to create new ORM resource for each old ORM resource with the same name in the same namespace in cluster and convert the patterns accordingly.

## Implementation Details

The compatibility controller is a typical kubernetes controller reconciles old orm resource only. That means

1. All changes to old orm resource are converted to new orm immediately, changes in new orm won't be pushed back to old orm
2. `spec.operand` in new orm is generated from `metadata.name` of old orm. namespace and name are not provided in old orm, simple enforcer will search the same namespace of the old orm and pick 1 resource if available.
3. If the operator CRD is not installed by the time the old orm is created, compatibility controller won't be able to find right Kind and Version for operator, it uses the resource as Kind and group without version as APIVersion 
4. `spec.enforcement` is a feature in new orm, it is set to once when the new orm is created and won't change by the old orm.
5. `spec.resourceMappings[].srcResourceSpec.componentNames` in old orm are aggregated to `spec.mappings.parameters` with an auto generated index attached to the name. e.g. `componentNames-0`
6. source and destnation path of mapping are copied as is with the replacement of `componentName` to `componentName-`+index

## Old ORM Archive

CRD and Samples of old ORM are moved to [archive folder](../archive/)