/*
Copyright 2020 KubeSphere Authors

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
	"time"

	extinformers "github.com/gocrane/api/pkg/generated/informers/externalversions"
	"k8s.io/client-go/dynamic/dynamicinformer"

	apiextensionsinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	ksfake "kubesphere.io/scheduling/pkg/client/clientset/versioned/fake"
	scheduleinformers "kubesphere.io/scheduling/pkg/client/informers/externalversions"
)

type nullInformerFactory struct {
	fakeK8sInformerFactory informers.SharedInformerFactory
	fakeKsInformerFactory  scheduleinformers.SharedInformerFactory
}

func NewNullInformerFactory() InformerFactory {
	fakeClient := fake.NewSimpleClientset()
	fakeInformerFactory := informers.NewSharedInformerFactory(fakeClient, time.Minute*10)

	fakeKsClient := ksfake.NewSimpleClientset()
	fakeKsInformerFactory := scheduleinformers.NewSharedInformerFactory(fakeKsClient, time.Minute*10)

	return &nullInformerFactory{
		fakeK8sInformerFactory: fakeInformerFactory,
		fakeKsInformerFactory:  fakeKsInformerFactory,
	}
}
func (n nullInformerFactory) KubernetesSharedInformerFactory() informers.SharedInformerFactory {
	return n.fakeK8sInformerFactory
}

func (n nullInformerFactory) ScheduleSharedInformerFactory() scheduleinformers.SharedInformerFactory {
	return n.fakeKsInformerFactory
}

func (n nullInformerFactory) ApiExtensionSharedInformerFactory() apiextensionsinformers.SharedInformerFactory {
	return nil
}

func (n nullInformerFactory) ExtensionSharedInformerFactory() extinformers.SharedInformerFactory {
	return nil
}

func (n nullInformerFactory) DynamicSharedInformerFactory() dynamicinformer.DynamicSharedInformerFactory {
	return nil
}

func (n nullInformerFactory) CraneInformer() extinformers.SharedInformerFactory {
	return nil
}

func (n nullInformerFactory) Start(stopCh <-chan struct{}) {
}
