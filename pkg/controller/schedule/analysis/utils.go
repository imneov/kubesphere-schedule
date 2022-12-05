/*
Copyright 2021 The tKeel Authors.

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

package analysis

import (
	"fmt"
	"reflect"
	"strings"

	cranev1alpha1 "github.com/gocrane/api/analysis/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubesphere.io/scheduling/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func NotNil(obj interface{}, msgs ...string) error {
	if !IsNil(obj) {
		return nil
	}
	if len(msgs) > 0 {
		info := ""
		info = strings.Join(msgs, " ")
		info = fmt.Sprintf("(%s)", info)
		return fmt.Errorf("[%v] is nil %v", obj, msgs)
	}
	return fmt.Errorf("[%v] is nil", obj)
}
func IsNil(v interface{}) bool {
	valueOf := reflect.ValueOf(v)

	k := valueOf.Kind()

	switch k {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return valueOf.IsNil()
	default:
		return v == nil
	}
}

func Label(labels map[string]string, name string) string {
	if labels == nil {
		return ""
	}
	if label, ok := labels[name]; ok {
		return label
	}
	return ""
}

func getObjectReference(workload interface{}) (obj corev1.ObjectReference, taskKey string) {
	gv := appsv1.SchemeGroupVersion
	switch workload := (workload).(type) {
	case *appsv1.Deployment:
		return corev1.ObjectReference{
			Kind:       gv.WithKind("Deployment").Kind,
			APIVersion: gv.String(),
			Name:       workload.Name,
			Namespace:  workload.Namespace,
		}, Label(workload.Labels, constants.AnalysisTaskLabelKey)
	case appsv1.Deployment:
		return corev1.ObjectReference{
			Kind:       gv.WithKind("Deployment").Kind,
			APIVersion: gv.String(),
			Name:       workload.Name,
			Namespace:  workload.Namespace,
		}, Label(workload.Labels, constants.AnalysisTaskLabelKey)
	case *appsv1.StatefulSet:
		return corev1.ObjectReference{
			Kind:       gv.WithKind("StatefulSet").Kind,
			APIVersion: gv.String(),
			Name:       workload.Name,
			Namespace:  workload.Namespace,
		}, Label(workload.Labels, constants.AnalysisTaskLabelKey)
	case appsv1.StatefulSet:
		return corev1.ObjectReference{
			Kind:       gv.WithKind("StatefulSet").Kind,
			APIVersion: gv.String(),
			Name:       workload.Name,
			Namespace:  workload.Namespace,
		}, Label(workload.Labels, constants.AnalysisTaskLabelKey)
	case *appsv1.DaemonSet:
		return corev1.ObjectReference{
			Kind:       gv.WithKind("DaemonSet").Kind,
			APIVersion: gv.String(),
			Name:       workload.Name,
			Namespace:  workload.Namespace,
		}, Label(workload.Labels, constants.AnalysisTaskLabelKey)
	case appsv1.DaemonSet:
		return corev1.ObjectReference{
			Kind:       gv.WithKind("DaemonSet").Kind,
			APIVersion: gv.String(),
			Name:       workload.Name,
			Namespace:  workload.Namespace,
		}, Label(workload.Labels, constants.AnalysisTaskLabelKey)
	default:
		panic("unsupported workload type")
	}
}

var getQKV = apiutil.GVKForObject

func convertAnalytics(name string, target corev1.ObjectReference, strategy cranev1alpha1.CompletionStrategy) (analytics *cranev1alpha1.Analytics) {
	analytics = &cranev1alpha1.Analytics{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: cranev1alpha1.AnalyticsSpec{
			Type: cranev1alpha1.AnalysisTypeResource,
			ResourceSelectors: []cranev1alpha1.ResourceSelector{cranev1alpha1.ResourceSelector{
				Kind: target.Kind, APIVersion: target.APIVersion, Name: target.Name,
			}},
			CompletionStrategy: strategy,
		},
	}
	return analytics
}

// labelAnalyticsWithAnalysisName adds a kubesphere.io/analysis=[analysisName] label to crane.analysis
func labelAnalyticsWithAnalysisName(analytics *cranev1alpha1.Analytics, analysisTaskName string) *cranev1alpha1.Analytics {
	if analytics.Labels == nil {
		analytics.Labels = make(map[string]string, 0)
	}
	analytics.Labels[constants.AnalysisTaskLabelKey] = analysisTaskName
	return analytics
}

func genAnalysisTaskIndexKey(namespace, name string) string {
	return strings.ToLower(fmt.Sprintf("%s/%s", namespace, name))
}

func genAnalysisName(taskKind, kind, name string) string {
	return strings.ToLower(fmt.Sprintf("kubesphere-%s-%s-%s", taskKind, kind, name))
}

func genWorkloadName(kind, name string) string {
	return strings.ToLower(fmt.Sprintf("%s/%s", kind, name))
}
