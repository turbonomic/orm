# Compatibility

## Overview

Operator Resource Mapping(ORM) moved from turbonomic.com to devops.turbonomic.io and the schema has changed. In order to automatically upgrade old ORM resource, a compatibility controller is introduced to create new ORM resource for each old ORM resource with the same name in the same namespace in cluster and convert the patterns accordingly.

## Implementation Details

The compatibility controller is a typical kubernetes controller reconciles old orm resource only. That means

1. All changes to old orm resource are converted to new orm immediately, changes in new orm won't be pushed back to old orm
2. The compatibility controller simply find all mappings in legacy CR and convert them one by one to new ORM CR. It does not leverage parameter list and other pattern capabilities in new implementation
3. The best way to have new ORM CR with all new pattern capabilities if to compose a new one.

## Old ORM Archive

CRD and Samples of old ORM are moved to [archive folder](../archive/)