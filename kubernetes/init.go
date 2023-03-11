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

package kubernetes

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

type ToolboxType struct {
	cfg    *rest.Config
	ctx    context.Context
	scheme *runtime.Scheme

	Schema
	Client
	InformerFactory
}

var (
	resync = 10 * time.Minute

	Toolbox *ToolboxType
)

func (r *ToolboxType) Start(ctx context.Context) {
	r.ctx = ctx
	r.InformerFactory.Start(r.ctx.Done())
}

func InitToolbox(config *rest.Config, scheme *runtime.Scheme) error {
	var err error

	if Toolbox == nil {
		Toolbox = &ToolboxType{}
	}

	Toolbox.cfg = config
	if Toolbox.cfg == nil {
		return errors.New("nil Config for discovery")
	}
	Toolbox.scheme = scheme

	Toolbox.ctx = context.TODO()
	Toolbox.Client.Interface, err = dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	Toolbox.OrmClient, err = client.New(config, client.Options{Scheme: Toolbox.scheme})
	if err != nil {
		return err
	}

	Toolbox.InformerFactory = InformerFactory{}
	Toolbox.InformerFactory.DynamicSharedInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(Toolbox.Client, resync)

	Toolbox.discoverSchemaMappings()

	return err
}
