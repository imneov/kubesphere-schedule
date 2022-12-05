/*
Copyright 2020 The KubeSphere Authors.

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

package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	cranealpha1 "github.com/gocrane/api/analysis/v1alpha1"
	ext "github.com/gocrane/api/pkg/generated/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"kubesphere.io/scheduling/api"
	"kubesphere.io/scheduling/api/schedule/v1alpha1"
	schedulev1alpha1 "kubesphere.io/scheduling/api/schedule/v1alpha1"
	"kubesphere.io/scheduling/pkg/apiserver/query"
	"kubesphere.io/scheduling/pkg/client/clientset/versioned"
	"kubesphere.io/scheduling/pkg/client/informers/externalversions"
	"kubesphere.io/scheduling/pkg/constants"
	ks_informers "kubesphere.io/scheduling/pkg/informers"
	resourcesV1alpha3 "kubesphere.io/scheduling/pkg/models/resources/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapPrefix      = "kubeconfig-"
	kubeconfigNameFormat = configMapPrefix + "%s"
	defaultClusterName   = "local"
	defaultNamespace     = "default"
	kubeconfigNotify     = "notify"
	configMapKind        = "ConfigMap"
	configMapAPIVersion  = "v1"
	privateKeyAnnotation = "kubesphere.io/private-key"
	residual             = 72 * time.Hour
)

type Operator interface {
	//AnalysisTask
	CreateAnalysisTask(ctx context.Context, namespace string, task *v1alpha1.AnalysisTask) (*v1alpha1.AnalysisTask, error)
	ListAnalysisTask(ctx context.Context, query *query.Query) (api.ListResult, error)
	ModifyAnalysisTask(ctx context.Context, namespace, id string, task *v1alpha1.AnalysisTask) error
	DescribeAnalysisTask(ctx context.Context, namespace, id string) (*v1alpha1.AnalysisTask, error)
	DeleteAnalysisTask(ctx context.Context, namespace, id string) error
	ModifyAnalysisTaskConfig(ctx context.Context, schedule *AnalysisTaskConfig) (*AnalysisTaskConfig, error)

	//Scheduler
	GetScheduleConfig(ctx context.Context) (*schedulev1alpha1.ClusterScheduleConfig, error)
	ModifySchedulerConfig(ctx context.Context, config *SchedulerConfig) (*SchedulerConfig, error)

	//Crane
	CreateCraneAnalysis(ctx context.Context, namespace string, analytics *cranealpha1.Analytics) error
	DeleteCraneAnalysis(ctx context.Context, namespace string, name string) error
}

func NewScheduleOperator(ksInformers ks_informers.InformerFactory,
	k8sClient kubernetes.Interface,
	scheduleClient versioned.Interface,
	resClient ext.Interface,
	stopCh <-chan struct{}) Operator {
	klog.Infof("start helm repo informer")

	return &scheduleOperator{
		informers:      ksInformers.ScheduleSharedInformerFactory(),
		k8sClient:      k8sClient,
		scheduleClient: scheduleClient,
		resClient:      resClient,
	}
}

type scheduleOperator struct {
	informers      externalversions.SharedInformerFactory
	scheduleClient versioned.Interface
	k8sClient      kubernetes.Interface
	resClient      ext.Interface
}

func (s *scheduleOperator) DescribeAnalysisTask(ctx context.Context, namespace, id string) (*v1alpha1.AnalysisTask, error) {
	return s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(namespace).Get(ctx, id, metav1.GetOptions{})
}

func (s *scheduleOperator) DeleteAnalysisTask(ctx context.Context, namespace, id string) error {
	return s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(namespace).Delete(ctx, id, metav1.DeleteOptions{})
}

func (s *scheduleOperator) ModifyAnalysisTaskConfig(ctx context.Context, config *AnalysisTaskConfig) (*AnalysisTaskConfig, error) {
	cfg, err := s.GetScheduleConfig(ctx)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("Failed to get cluster configurations for install ks in %s: %w", constants.KubeSphereNamespace, err)
		}
	}

	if config.EnableNotify != nil {
		cfg.Analysis.EnableNotify = *config.EnableNotify
	}
	if config.CPUNotifyPresent != nil {
		cfg.Analysis.NotifyThreshold.CPU = *config.CPUNotifyPresent
	}
	if config.MemNotifyPresent != nil {
		cfg.Analysis.NotifyThreshold.Mem = *config.MemNotifyPresent
	}

	return config, nil
}

func (s *scheduleOperator) ModifySchedulerConfig(ctx context.Context, config *SchedulerConfig) (*SchedulerConfig, error) {
	if config.Scheduler == nil {
		err := fmt.Errorf("Failed to update default scheduler, config.Scheduler is nil")
		return nil, err
	}

	cfg, err := s.GetScheduleConfig(ctx)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("Failed to get cluster configurations for install ks in %s: %w", constants.KubeSphereNamespace, err)
		}
	}

	cfg.DefaultScheduler = *config.Scheduler
	err = s.SaveScheduleConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch: %w", err)
	}

	return config, nil
}

func (s *scheduleOperator) CreateAnalysisTask(ctx context.Context, namespace string, task *v1alpha1.AnalysisTask) (*v1alpha1.AnalysisTask, error) {
	item, err := s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(namespace).Create(ctx, task, metav1.CreateOptions{})

	if err != nil {
		klog.Errorf("list helm repo failed: %s", err)
		return nil, err
	}

	return item, nil
}

func (s *scheduleOperator) GetAnalysisTask(ctx context.Context, namespace, id string) (*v1alpha1.AnalysisTask, error) {
	analysis, err := s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(namespace).Get(ctx, id, metav1.GetOptions{})

	if err != nil {
		klog.Error("get repo failed", err)
		return nil, err
	}
	return analysis, nil
}

func (s *scheduleOperator) ModifyAnalysisTask(ctx context.Context, namespace, id string, task *v1alpha1.AnalysisTask) error {
	analysis, err := s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(namespace).Get(ctx, id, metav1.GetOptions{})

	if err != nil {
		klog.Error("get analysis failed", err)
		return err
	}

	analysisCopy := analysis.DeepCopy()
	analysisCopy.Annotations = task.Annotations
	analysisCopy.Labels = task.Labels
	analysisCopy.Spec = task.Spec
	analysisCopy.Labels = task.Labels

	patch := client.MergeFrom(analysis)
	data, err := patch.Data(analysisCopy)
	if err != nil {
		klog.Error("create patch failed", err)
		return err
	}

	// data == "{}", need not to patch
	if len(data) == 2 {
		return nil
	}

	_, err = s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(namespace).Patch(ctx, id, patch.Type(), data, metav1.PatchOptions{})

	if err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

func (s *scheduleOperator) ListAnalysisTask(ctx context.Context, query *query.Query) (api.ListResult, error) {
	tasks, err := s.scheduleClient.ScheduleV1alpha1().AnalysisTasks(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return api.ListResult{}, err
	}
	var result = make([]runtime.Object, 0, len(tasks.Items))
	for _, item := range tasks.Items {
		item.Status.TargetDeployments = s.GetDeployments(item)
		item.Status.TargetStatefulSets = s.GetStatefulSets(item)
		if item.Spec.Type == v1alpha1.NamespaceResourceType {
			item.Status.TargetNamespaces = s.GetNamespaces(item)
		}
		result = append(result, item.DeepCopy())
	}
	return *resourcesV1alpha3.DefaultList(result, query, resourcesV1alpha3.DefaultCompare(), resourcesV1alpha3.DefaultFilter()), nil

}

func (s *scheduleOperator) GetDeployments(item v1alpha1.AnalysisTask) []corev1.ObjectReference {
	labelSelect := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
		client.MatchingLabels{constants.AnalysisTaskLabelKey: item.Name},
	})
	ret, err := s.k8sClient.AppsV1().Deployments(item.Namespace).List(context.Background(), *labelSelect.AsListOptions())
	if err != nil {
		klog.Error(err)
	} else {
		var result = make([]corev1.ObjectReference, 0, len(ret.Items))
		for _, workload := range ret.Items {
			result = append(result, corev1.ObjectReference{
				APIVersion: workload.APIVersion,
				Kind:       v1alpha1.DeploymentResource,
				Name:       workload.Name,
				Namespace:  workload.Namespace,
			})
		}
		return result
	}
	return []corev1.ObjectReference{}
}

func (s *scheduleOperator) GetStatefulSets(item v1alpha1.AnalysisTask) []corev1.ObjectReference {
	labelSelect := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
		client.MatchingLabels{constants.AnalysisTaskLabelKey: item.Name},
	})
	ret, err := s.k8sClient.AppsV1().StatefulSets(item.Namespace).List(context.Background(), *labelSelect.AsListOptions())
	if err != nil {
		klog.Error(err)
	} else {
		var result = make([]corev1.ObjectReference, 0, len(ret.Items))
		for _, workload := range ret.Items {
			result = append(result, corev1.ObjectReference{
				APIVersion: workload.APIVersion,
				Kind:       v1alpha1.StatefulSetResource,
				Name:       workload.Name,
				Namespace:  workload.Namespace,
			})
		}
		return result
	}
	return []corev1.ObjectReference{}
}

func (s *scheduleOperator) GetNamespaces(item v1alpha1.AnalysisTask) []corev1.ObjectReference {
	labelSelect := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
		client.MatchingLabels{constants.AnalysisTaskLabelKey: item.Name},
	})
	ret, err := s.k8sClient.CoreV1().Namespaces().List(context.Background(), *labelSelect.AsListOptions())
	if err != nil {
		klog.Error(err)
	} else {
		var result = make([]corev1.ObjectReference, 0, len(ret.Items))
		for _, workload := range ret.Items {
			result = append(result, corev1.ObjectReference{
				APIVersion: workload.APIVersion,
				Kind:       v1alpha1.NamespaceResourceType,
				Name:       workload.Name,
				Namespace:  workload.Namespace,
			})
		}
		return result
	}
	return []corev1.ObjectReference{}
}

// UpdateKubeconfig Update client key and client certificate after CertificateSigningRequest has been approved
func (o *scheduleOperator) GetScheduleConfig(ctx context.Context) (*schedulev1alpha1.ClusterScheduleConfig, error) {
	configMap, err := o.k8sClient.CoreV1().ConfigMaps(constants.KubesphereScheduleNamespace).Get(context.Background(), constants.KubesphereScheduleConfigMap, metav1.GetOptions{})
	if err != nil {
		klog.Errorln(err)
		return nil, err
	}

	config := schedulev1alpha1.ClusterScheduleConfig{}
	if err := json.Unmarshal([]byte(configMap.Data["config"]), &config); err != nil {
		klog.Errorln(err)
		return nil, err
	}
	return &config, nil
}

// UpdateKubeconfig Update client key and client certificate after CertificateSigningRequest has been approved
func (o *scheduleOperator) SaveScheduleConfig(ctx context.Context, config *schedulev1alpha1.ClusterScheduleConfig) error {
	configMap, err := o.k8sClient.CoreV1().ConfigMaps(constants.KubesphereScheduleNamespace).Get(context.Background(), constants.KubesphereScheduleConfigMap, metav1.GetOptions{})
	if err != nil {
		klog.Errorln(err)
		return err
	}

	data, err := json.Marshal(&config)
	if err != nil {
		klog.Errorln(err)
		return err
	}

	configMap.Data["config"] = string(data)
	_, err = o.k8sClient.CoreV1().ConfigMaps(constants.KubesphereScheduleNamespace).Update(context.Background(), configMap, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorln(err)
		return err
	}
	return nil
}
