/*
Copyright 2019 The KubeSphere Authors.

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

package informers

import (
	"reflect"
	"time"

	ext "github.com/gocrane/api/pkg/generated/clientset/versioned"
	cranev1informer "github.com/gocrane/api/pkg/generated/informers/externalversions"
	extinformers "github.com/gocrane/api/pkg/generated/informers/externalversions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	cranev1 "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"kubesphere.io/scheduling/pkg/client/clientset/versioned"
	ksinformers "kubesphere.io/scheduling/pkg/client/informers/externalversions"
)

// default re-sync period for all informer factories
const defaultResync = 600 * time.Second

// InformerFactory is a group all shared informer factories which kubesphere needed
// callers should check if the return value is nil
type InformerFactory interface {
	KubernetesSharedInformerFactory() k8sinformers.SharedInformerFactory
	ScheduleSharedInformerFactory() ksinformers.SharedInformerFactory
	ExtensionSharedInformerFactory() extinformers.SharedInformerFactory
	ApiExtensionSharedInformerFactory() apiextensionsinformers.SharedInformerFactory
	DynamicSharedInformerFactory() dynamicinformer.DynamicSharedInformerFactory
	CraneInformer() cranev1informer.SharedInformerFactory
	// Start shared informer factory one by one if they are not nil
	Start(stopCh <-chan struct{})
}

type GenericInformerFactory interface {
	Start(stopCh <-chan struct{})
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
}

type informerFactories struct {
	informerFactory              k8sinformers.SharedInformerFactory
	scheduleInformerFactory      ksinformers.SharedInformerFactory
	apiextensionsInformerFactory apiextensionsinformers.SharedInformerFactory
	extensionsInformerFactory    extinformers.SharedInformerFactory
	dynamicInformerFactory       dynamicinformer.DynamicSharedInformerFactory
	craneInformer                cranev1informer.SharedInformerFactory
}

func NewInformerFactories(client kubernetes.Interface, scheduleClient versioned.Interface,
	extClient ext.Interface, craneClient cranev1.Interface,
	apiextensionsClient apiextensionsclient.Interface, dynamicClient dynamic.Interface) InformerFactory {
	factory := &informerFactories{}

	if client != nil {
		factory.informerFactory = k8sinformers.NewSharedInformerFactory(client, defaultResync)
	}

	if scheduleClient != nil {
		factory.scheduleInformerFactory = ksinformers.NewSharedInformerFactory(scheduleClient, defaultResync)
	}

	if apiextensionsClient != nil {
		factory.apiextensionsInformerFactory = apiextensionsinformers.NewSharedInformerFactory(apiextensionsClient, defaultResync)
	}

	if dynamicClient != nil {
		factory.dynamicInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, defaultResync)
	}

	if craneClient != nil {
		factory.craneInformer = cranev1informer.NewSharedInformerFactory(craneClient, defaultResync)
	}

	if client != nil {
		factory.extensionsInformerFactory = extinformers.NewSharedInformerFactory(extClient, defaultResync)
	}

	return factory
}

func (f *informerFactories) DynamicSharedInformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return f.dynamicInformerFactory
}

func (f *informerFactories) KubernetesSharedInformerFactory() k8sinformers.SharedInformerFactory {
	return f.informerFactory
}

func (f *informerFactories) ScheduleSharedInformerFactory() ksinformers.SharedInformerFactory {
	return f.scheduleInformerFactory
}

func (f *informerFactories) ApiExtensionSharedInformerFactory() apiextensionsinformers.SharedInformerFactory {
	return f.apiextensionsInformerFactory
}

func (f *informerFactories) ExtensionSharedInformerFactory() extinformers.SharedInformerFactory {
	return f.extensionsInformerFactory
}

func (f *informerFactories) CraneInformer() extinformers.SharedInformerFactory {
	return f.craneInformer
}

func (f *informerFactories) Start(stopCh <-chan struct{}) {
	if f.informerFactory != nil {
		f.informerFactory.Start(stopCh)
	}

	if f.scheduleInformerFactory != nil {
		f.scheduleInformerFactory.Start(stopCh)
	}

	if f.apiextensionsInformerFactory != nil {
		f.apiextensionsInformerFactory.Start(stopCh)
	}

	if f.extensionsInformerFactory != nil {
		f.extensionsInformerFactory.Start(stopCh)
	}

	if f.dynamicInformerFactory != nil {
		f.dynamicInformerFactory.Start(stopCh)
	}

	if f.craneInformer != nil {
		f.craneInformer.Start(stopCh)
	}

}
