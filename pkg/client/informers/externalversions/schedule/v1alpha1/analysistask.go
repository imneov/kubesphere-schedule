/*
Copyright 2022 The KubeSphere Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	schedulev1alpha1 "kubesphere.io/scheduling/api/schedule/v1alpha1"
	versioned "kubesphere.io/scheduling/pkg/client/clientset/versioned"
	internalinterfaces "kubesphere.io/scheduling/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "kubesphere.io/scheduling/pkg/client/listers/schedule/v1alpha1"
)

// AnalysisTaskInformer provides access to a shared informer and lister for
// AnalysisTasks.
type AnalysisTaskInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.AnalysisTaskLister
}

type analysisTaskInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAnalysisTaskInformer constructs a new informer for AnalysisTask type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAnalysisTaskInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAnalysisTaskInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAnalysisTaskInformer constructs a new informer for AnalysisTask type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAnalysisTaskInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ScheduleV1alpha1().AnalysisTasks(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ScheduleV1alpha1().AnalysisTasks(namespace).Watch(context.TODO(), options)
			},
		},
		&schedulev1alpha1.AnalysisTask{},
		resyncPeriod,
		indexers,
	)
}

func (f *analysisTaskInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAnalysisTaskInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *analysisTaskInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&schedulev1alpha1.AnalysisTask{}, f.defaultInformer)
}

func (f *analysisTaskInformer) Lister() v1alpha1.AnalysisTaskLister {
	return v1alpha1.NewAnalysisTaskLister(f.Informer().GetIndexer())
}
