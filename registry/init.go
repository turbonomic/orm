/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registry

import (
	"context"
	"errors"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Registry struct {
	cfg *rest.Config
	ctx context.Context

	OperandRegistry
	SourceRegistry
	Schema
	Client
	Informer
}

var (
	r *Registry
)

func GetORMRegistry(config *rest.Config) (*Registry, error) {
	var err error

	if r == nil {
		r = &Registry{}
	}

	r.cfg = config

	if r.cfg == nil {
		return nil, errors.New("Null Config for discovery")
	}

	r.ctx = context.TODO()
	r.Client.Interface, err = dynamic.NewForConfig(config)

	return r, err
}
