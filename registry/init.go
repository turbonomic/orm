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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Registry struct {
	cfg    *rest.Config
	ctx    context.Context
	scheme *runtime.Scheme

	OperandRegistry
	SourceRegistry
	Schema
	Client
	InformerFactory
}

var (
	resync = 10 * time.Minute
	stopCh chan struct{}

	r *Registry
)

func (r *Registry) Start(ctx context.Context) {
	r.ctx = ctx
	r.InformerFactory.Start(r.ctx.Done())
}

func GetORMRegistry(config *rest.Config, scheme *runtime.Scheme) (*Registry, error) {
	var err error

	if r == nil {
		r = &Registry{}
	}

	r.cfg = config
	if r.cfg == nil {
		return nil, errors.New("Null Config for discovery")
	}
	r.scheme = scheme

	r.ctx = context.TODO()
	r.Client.Interface, err = dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	r.OrmClient, err = client.New(config, client.Options{Scheme: r.scheme})
	if err != nil {
		return nil, err
	}

	r.DynamicSharedInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(r.Client, resync)
	return r, err
}
