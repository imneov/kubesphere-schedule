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

package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	cranev1alpha1 "github.com/gocrane/api/analysis/v1alpha1"
	cranev1informer "github.com/gocrane/api/pkg/generated/informers/externalversions/analysis/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	urlruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	v1 "k8s.io/client-go/informers/apps/v1"
	corev1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubesphere.io/scheduling/api/schedule/v1alpha1"
	schedulev1alpha1 "kubesphere.io/scheduling/api/schedule/v1alpha1"
	"kubesphere.io/scheduling/pkg/client/k8s"
	"kubesphere.io/scheduling/pkg/constants"
	"kubesphere.io/scheduling/pkg/models/schedule"
	"kubesphere.io/scheduling/pkg/utils/jsonpath"
	"kubesphere.io/scheduling/pkg/utils/sliceutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AnalysisTaskReconciler reconciles a Analysis object
type AnalysisTaskReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	ctx        context.Context
	Recorder   record.EventRecorder
	RestMapper meta.RESTMapper

	ClusterScheduleConfig *schedulev1alpha1.ClusterScheduleConfig

	K8SClient                  k8s.Client
	ScheduleClient             schedule.Operator
	AnalyticsInformer          cranev1informer.AnalyticsInformer
	RecommendationInformer     cranev1informer.RecommendationInformer
	DynamicInformer            dynamicinformer.DynamicSharedInformerFactory
	NamespaceInformer          corev1informer.NamespaceInformer
	DeploymentsInformer        v1.DeploymentInformer
	DaemonSetsInformer         v1.DaemonSetInformer
	StatefulSetsInformer       v1.StatefulSetInformer
	NameSpaceAnalysisTaskCache map[string]*v1alpha1.AnalysisTask // cache namespace analysis task for new workload
	AnalysisTaskReverseIndex   map[string]*v1alpha1.AnalysisTask // map[<namespace>]map[<taskName>]*v1alpha1.AnalysisTask
}

//+kubebuilder:rbac:groups=scheduling.kubesphere.io,resources=analysistasks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=scheduling.kubesphere.io,resources=analysistasks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=scheduling.kubesphere.io,resources=analysistasks/finalizers,verbs=update
//+kubebuilder:rbac:groups=installer.kubesphere.io,resources=clusterconfigurations,verbs= get;list;watch;patch;update

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get

//+kubebuilder:rbac:groups=analysis.crane.io,resources=analytics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=analysis.crane.io,resources=analytics/status,verbs=get
//+kubebuilder:rbac:groups=analysis.crane.io,resources=recommendations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=analysis.crane.io,resources=recommendations/status,verbs=get

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Analysis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *AnalysisTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("AnalysisTask", req.NamespacedName)
	instance := &v1alpha1.AnalysisTask{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AnalysisTask deleted")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(instance, constants.AnalysisTaskFinalizer) {
			controllerutil.AddFinalizer(instance, constants.AnalysisTaskFinalizer)
			instance.Status.Status = v1alpha1.UpdatingStatus
			if err := r.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}
	} else {
		if controllerutil.ContainsFinalizer(instance, constants.AnalysisTaskFinalizer) {
			if _, err := r.undoReconcile(ctx, instance); err != nil {
				return reconcile.Result{}, err
			}
			controllerutil.RemoveFinalizer(instance, constants.AnalysisTaskFinalizer)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	if _, err := r.doReconcile(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AnalysisTaskReconciler) doReconcile(ctx context.Context, instance *schedulev1alpha1.AnalysisTask) (ret ctrl.Result, err error) {
	log := r.Log.WithName("doReconcile")
	switch instance.Spec.Type {
	case v1alpha1.WorkloadResourceType:
		ret, err = r.doReconcileWorkloadAnalysis(ctx, instance)
	case v1alpha1.NamespaceResourceType:
		ret, err = r.doReconcileNamespaceAnalysis(ctx, instance)
	default:
		log.Info(fmt.Sprintf("not support resource type %s, should be Namespace/Workload", instance.Spec.Type))
		return ctrl.Result{}, nil
	}
	if err != nil {
		r.updateAnalysisTaskStatus(ctx, instance, schedulev1alpha1.ErrorStatus)
	}
	return
}

func (r *AnalysisTaskReconciler) undoReconcile(ctx context.Context, instance *schedulev1alpha1.AnalysisTask) (ctrl.Result, error) {
	log := r.Log.WithName("undoReconcile")
	switch instance.Spec.Type {
	case v1alpha1.NamespaceResourceType:
		return r.undoReconcileNamespaceAnalysis(ctx, instance)
	case v1alpha1.WorkloadResourceType:
		return r.undoReconcileWorkloadAnalysis(ctx, instance)
	default:
		log.Info(fmt.Sprintf("not support resource type %s", instance.Spec.Type))
		return ctrl.Result{}, nil
	}
}

func (r *AnalysisTaskReconciler) doReconcileWorkloadAnalysis(ctx context.Context, instance *schedulev1alpha1.AnalysisTask) (ctrl.Result, error) {
	log := r.Log.WithName("doReconcileWorkloadAnalysis")
	hasError := false
	namespace := instance.Namespace
	workloads := strings.Split(r.getNamespacesWorkloads(ctx, namespace), ",")
	for _, resource := range instance.Spec.ResourceSelectors {
		workload := corev1.ObjectReference{
			Kind:       resource.Kind,
			Name:       resource.Name,
			APIVersion: resource.APIVersion,
			Namespace:  namespace,
		}
		err := r.CreateCraneAnalysis(ctx, workload, instance)
		if err != nil {
			r.Recorder.Event(instance, corev1.EventTypeWarning, "CreateAnalysis", err.Error())
			hasError = true
			log.Error(err, "create crane analysis failed")
			continue
		}
		key := genWorkloadName(workload.Kind, workload.Name)
		if !sliceutil.HasString(workloads, key) {
			workloads = append(workloads, key)
		}
	}
	r.UpdateIndexCache(instance)
	if hasError {
		err := r.updateAnalysisTaskStatus(ctx, instance, schedulev1alpha1.ErrorStatus)
		return ctrl.Result{}, err
	}

	workloads = sliceutil.RemoveString(workloads, func(item string) bool { return item == "" })
	if err := r.updateNamespacesWorkloads(ctx, namespace, strings.Join(workloads, ",")); err != nil {
		log.Error(err, "get deployment error")
		return ctrl.Result{}, err
	}

	var err error
	if instance.Spec.CompletionStrategy.CompletionStrategyType == v1alpha1.CompletionStrategyOnce {
		err = r.updateAnalysisTaskStatus(ctx, instance, schedulev1alpha1.CompletedStatus)
	}
	if instance.Spec.CompletionStrategy.CompletionStrategyType == v1alpha1.CompletionStrategyPeriodical {
		err = r.updateAnalysisTaskStatus(ctx, instance, schedulev1alpha1.RunningStatus)
	}
	if err != nil {
		log.Error(err, "update analysis task status error")
	}

	return ctrl.Result{}, nil
}

func (r *AnalysisTaskReconciler) undoReconcileWorkloadAnalysis(ctx context.Context, instance *schedulev1alpha1.AnalysisTask) (ctrl.Result, error) {
	log := r.Log.WithName("undoReconcileWorkloadAnalysis")
	//删除 Analysis
	for _, resource := range instance.Spec.ResourceSelectors {
		if resource.Kind == schedulev1alpha1.NamespaceResourceType {
			log.Info(fmt.Sprintf("unknown kind %s/%s", resource.Kind, resource.Name))
			continue
		}

		if err := r.DeleteCraneAnalysis(ctx, corev1.ObjectReference{
			Kind:       resource.Kind,
			Name:       resource.Name,
			APIVersion: resource.APIVersion,
			Namespace:  instance.Namespace,
		}, instance); err != nil {
			log.Error(err, "delete analysis error")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *AnalysisTaskReconciler) doReconcileNamespaceAnalysis(ctx context.Context, instance *schedulev1alpha1.AnalysisTask) (ctrl.Result, error) {
	log := r.Log.WithName("doReconcileNamespaceAnalysis")
	if instance.Spec.Type != schedulev1alpha1.NamespaceResourceType {
		err := fmt.Errorf("not support resource type %s", instance.Spec.Type)
		return ctrl.Result{}, err
	}
	for _, resource := range instance.Spec.ResourceSelectors {
		if resource.Kind != schedulev1alpha1.NamespaceResourceType {
			log.Info(fmt.Sprintf("unknown kind %v", resource))
			continue
		}

		namespace := resource.Name
		if namespace == "" {
			log.Info(fmt.Sprintf("namespace is empty, %v", instance))
			continue
		}
		r.NameSpaceAnalysisTaskCache[namespace] = instance

		result, err := r.reconcileNamespaceAnalysis(ctx, namespace, instance)
		if err != nil {
			return result, err
		}

		r.UpdateIndexCache(instance)

		// overwrite Namespaces Annotations
		err = r.updateNamespacesWorkloads(ctx, namespace, "")
		if err != nil {
			log.Error(err, "get deployment error")
			return ctrl.Result{}, err
		}
	}

	var err error
	if instance.Spec.CompletionStrategy.CompletionStrategyType == v1alpha1.CompletionStrategyOnce {
		err = r.updateAnalysisTaskStatus(ctx, instance, schedulev1alpha1.CompletedStatus)
	}
	if instance.Spec.CompletionStrategy.CompletionStrategyType == v1alpha1.CompletionStrategyPeriodical {
		err = r.updateAnalysisTaskStatus(ctx, instance, schedulev1alpha1.RunningStatus)
	}
	if err != nil {
		log.Error(err, "update analysis task status error")
	}
	return ctrl.Result{}, nil
}

func (r *AnalysisTaskReconciler) reconcileNamespaceAnalysis(ctx context.Context, namespace string, instance *schedulev1alpha1.AnalysisTask) (ctrl.Result, error) {
	log := r.Log.WithName("reconcileNamespaceAnalysis")
	deployments, err := r.K8SClient.Kubernetes().AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "get deployment error: %s", err.Error())
		return ctrl.Result{}, err
	}
	for _, resource := range deployments.Items {
		workload, _ := getObjectReference(resource)
		workload.Namespace = namespace
		err := r.CreateCraneAnalysis(ctx, workload, instance)
		if err != nil {
			log.Error(err, "create crane analysis failed")
			return ctrl.Result{}, err
		}
	}

	statefulSets, err := r.K8SClient.Kubernetes().AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "get deployment error: %s", err.Error())
		return ctrl.Result{}, err
	}
	for _, resource := range statefulSets.Items {
		workload, _ := getObjectReference(resource)
		workload.Namespace = namespace
		err := r.CreateCraneAnalysis(ctx, workload, instance)
		if err != nil {
			log.Error(err, "create crane analysis failed")
			return ctrl.Result{}, err
		}
	}

	daemonSets, err := r.K8SClient.Kubernetes().AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "get deployment error: %s", err.Error())
		return ctrl.Result{}, err
	}
	for _, resource := range daemonSets.Items {
		workload, _ := getObjectReference(resource)
		workload.Namespace = namespace
		err := r.CreateCraneAnalysis(ctx, workload, instance)
		if err != nil {
			log.Error(err, "create crane analysis failed")
			return ctrl.Result{}, err
		}
	}

	_ = r.updateNamespaceTaskLabel(ctx, namespace, instance)
	return ctrl.Result{}, nil
}

func (r *AnalysisTaskReconciler) undoReconcileNamespaceAnalysis(ctx context.Context, instance *schedulev1alpha1.AnalysisTask) (ctrl.Result, error) {
	log := r.Log.WithName("undoReconcileNamespaceAnalysis")

	for _, resource := range instance.Spec.ResourceSelectors {
		if resource.Kind != schedulev1alpha1.NamespaceResourceType ||
			resource.Name == "" {
			log.Info(fmt.Sprintf("unknown kind %v", resource))
			continue
		}
		namespace := resource.Name
		r.NameSpaceAnalysisTaskCache[namespace] = instance

		options := &client.MatchingLabels{
			constants.AnalysisTaskLabelKey: instance.Name,
		}
		deployments := &appsv1.DeploymentList{}
		if err := r.List(ctx, deployments, options); err != nil {
			log.Error(err, "Failed to Get Deployments")
			return ctrl.Result{}, err
		}
		for _, item := range deployments.Items {
			if err := r.DeleteCraneAnalysis(ctx, corev1.ObjectReference{
				Kind:       item.Kind,
				Name:       item.Name,
				APIVersion: item.APIVersion,
				Namespace:  namespace,
			}, instance); err != nil {
				log.Error(err, "delete analysis error")
			}
		}

		statefulSets := &appsv1.StatefulSetList{}
		if err := r.List(ctx, statefulSets, options); err != nil {
			log.Error(err, "Failed to Get Deployments")
			return ctrl.Result{}, err
		}
		for _, item := range statefulSets.Items {
			if err := r.DeleteCraneAnalysis(ctx, corev1.ObjectReference{
				Kind:       item.Kind,
				Name:       item.Name,
				APIVersion: item.APIVersion,
				Namespace:  namespace,
			}, instance); err != nil {
				log.Error(err, "delete analysis error")
				return ctrl.Result{}, err
			}
		}

		daemonSets := &appsv1.DaemonSetList{}
		if err := r.List(ctx, daemonSets, options); err != nil {
			log.Error(err, "Failed to Get Deployments")
			return ctrl.Result{}, err
		}
		for _, item := range daemonSets.Items {
			if err := r.DeleteCraneAnalysis(ctx, corev1.ObjectReference{
				Kind:       item.Kind,
				Name:       item.Name,
				APIVersion: item.APIVersion,
				Namespace:  namespace,
			}, instance); err != nil {
				log.Error(err, "delete analysis error")
				return ctrl.Result{}, err
			}
		}

		_ = r.cleanNamespaceTaskLabel(ctx, namespace)
	}
	return ctrl.Result{}, nil
}

func (r *AnalysisTaskReconciler) handleWorkloadUpdateEvent(workload interface{}) {
	log := r.Log.WithName("handleWorkloadUpdateEvent")
	objectRef, oldTaskKey := getObjectReference(workload)
	namespace := objectRef.Namespace
	if analysisTask, ok := r.NameSpaceAnalysisTaskCache[namespace]; ok {
		oldTask := r.GetAnalysisTask(namespace, oldTaskKey)
		if oldTask != nil {
			return
		}
		err := r.CreateCraneAnalysis(context.Background(), objectRef, analysisTask)
		if err != nil {
			log.Error(err, "create crane analysis failed")
			return
		}
	} else {
		log.Info(fmt.Sprintf("skip create workload analysis %s", objectRef.Name))
	}
}

func (r *AnalysisTaskReconciler) handleDeploymentRemoveEvent(deployment *appsv1.Deployment) {
	log := r.Log.WithName("handleDeploymentRemoveEvent")
	log.Info(fmt.Sprintf("handle deployment remove event %s, do nothing", deployment.Name))
}

func (r *AnalysisTaskReconciler) WorkloadEventHandler() cache.ResourceEventHandler {
	log := r.Log.WithName("WorkloadEventHandler")
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			r.handleWorkloadUpdateEvent(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			r.handleWorkloadUpdateEvent(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			log.Info(fmt.Sprintf("delete deployment delete event %s", deployment.Name))
		},
	}
}

func (r *AnalysisTaskReconciler) UpdateIndexCache(instance *schedulev1alpha1.AnalysisTask) {
	log := r.Log.WithName("UpdateIndexCache")
	//r.IndexCache[key] = instance
	key := genAnalysisTaskIndexKey(instance.Namespace, instance.Name)
	if o, ok := r.AnalysisTaskReverseIndex[key]; ok {
		log.Info(fmt.Sprintf("analysis task %s %s will overwrite", key, o.CreationTimestamp))
	}
	r.AnalysisTaskReverseIndex[key] = instance
}

func (r *AnalysisTaskReconciler) RemoveIndexCache(namespace, name string) {
	key := genAnalysisTaskIndexKey(namespace, name)
	delete(r.AnalysisTaskReverseIndex, key)
}

func (r *AnalysisTaskReconciler) GetAnalysisTask(namespace string, taskID string) *schedulev1alpha1.AnalysisTask {
	if task, ok := r.AnalysisTaskReverseIndex[genAnalysisTaskIndexKey(namespace, taskID)]; ok {
		return task
	}
	return nil
}

func (r *AnalysisTaskReconciler) ConfigEventHandler() cache.ResourceEventHandler {
	log := r.Log.WithName("ConfigEventHandler")
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			r.UpdateScheduleConfig(obj)
			log.Info(fmt.Sprintf("receiver Installer add event %v", r.ClusterScheduleConfig))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			r.UpdateScheduleConfig(newObj)
			log.Info(fmt.Sprintf("receiver Installer update event %v", r.ClusterScheduleConfig))
		},
		DeleteFunc: func(obj interface{}) {
			//r.UpdateScheduleConfig(obj)
			log.Info(fmt.Sprintf("delete Installer delete event %v", obj, r.ClusterScheduleConfig))
		},
	}
}

func (r *AnalysisTaskReconciler) AnalyticsEventHandler() cache.ResourceEventHandler {
	log := r.Log.WithName("AnalyticsEventHandler")
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			o := obj.(*cranev1alpha1.Analytics)
			log.Info(fmt.Sprintf("reciver Analytics add event %v", o))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			//o := newObj.(*cranev1alpha1.Analytics)
			//o.Labels @TODO
			//log.Info("reciver Analytics update event %v:%v", o.ResourceVersion, o.Status.LastUpdateTime)
			//for _, comm := range o.Status.Recommendations {
			//	log.Info("reciver Analytics update event %s: %s", comm.Name, comm.Message)
			//}
		},
		DeleteFunc: func(obj interface{}) {
			o := obj.(*cranev1alpha1.Analytics)
			log.Info(fmt.Sprintf("reciver Analytics delete event %v", o))
		},
	}
}

func (r *AnalysisTaskReconciler) RecommendationsEventHandler() cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
		},
		DeleteFunc: func(obj interface{}) {
		},
	}
}

func (r *AnalysisTaskReconciler) UpdateScheduleConfig(newObj interface{}) {
	log := r.Log.WithName("UpdateScheduleConfig")
	var config *schedulev1alpha1.ClusterScheduleConfig
	object := jsonpath.New(newObj)
	err := object.DataAs("spec.scheduler", &config)
	if err != nil {
		log.Error(err, "parse ClusterConfig is error", object)
		return
	}
	r.ClusterScheduleConfig = config
	log.Info(fmt.Sprintf("update cluster schedule config: %v", config))
}

// SetupWithManager sets up the controller with the Manager.
func (r *AnalysisTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	urlruntime.Must(NotNil(r.K8SClient))
	urlruntime.Must(NotNil(r.ScheduleClient))
	urlruntime.Must(NotNil(r.DeploymentsInformer))
	urlruntime.Must(NotNil(r.DaemonSetsInformer))
	urlruntime.Must(NotNil(r.StatefulSetsInformer))
	urlruntime.Must(NotNil(r.NamespaceInformer))
	urlruntime.Must(NotNil(r.DynamicInformer))
	urlruntime.Must(NotNil(r.ClusterScheduleConfig))
	urlruntime.Must(NotNil(r.AnalyticsInformer))
	urlruntime.Must(NotNil(r.RecommendationInformer))
	urlruntime.Must(NotNil(r.NameSpaceAnalysisTaskCache))
	urlruntime.Must(NotNil(r.AnalysisTaskReverseIndex))

	r.Log = mgr.GetLogger().WithName("controllers").WithName("AnalysisTask")
	r.DeploymentsInformer.Informer().AddEventHandler(r.WorkloadEventHandler())
	r.DaemonSetsInformer.Informer().AddEventHandler(r.WorkloadEventHandler())
	r.StatefulSetsInformer.Informer().AddEventHandler(r.WorkloadEventHandler())
	r.AnalyticsInformer.Informer().AddEventHandler(r.AnalyticsEventHandler())
	r.RecommendationInformer.Informer().AddEventHandler(r.RecommendationsEventHandler())
	//r.DynamicInformer.ForResource(schema.GroupVersionResource{
	//	Group: "installer.kubesphere.io", Version: "v1alpha1", Resource: "clusterconfigurations",
	//}).Informer().AddEventHandler(r.ConfigEventHandler())

	return ctrl.NewControllerManagedBy(mgr).
		For(&schedulev1alpha1.AnalysisTask{}).
		Complete(r)
}

func (r *AnalysisTaskReconciler) Validating(ctx context.Context, request admission.Request) admission.Response {
	return admission.Allowed("Validating")
}

func genAnalysis(resource corev1.ObjectReference, instance *schedulev1alpha1.AnalysisTask) *cranev1alpha1.Analytics {
	name := genAnalysisName(instance.Spec.Type, resource.Kind, resource.Name)
	analytics := convertAnalytics(name, resource, instance.Spec.CompletionStrategy)
	analytics = labelAnalyticsWithAnalysisName(analytics, instance.Name)
	return analytics
}

func (r *AnalysisTaskReconciler) CreateCraneAnalysis(ctx context.Context, resource corev1.ObjectReference, instance *v1alpha1.AnalysisTask) error {
	log := r.Log.WithName("CreateCraneAnalysis")

	if err := r.updateWorkloadTaskLabel(ctx, resource, instance); err != nil {
		log.Error(err, "update workload error")
		return err
	}

	analytics := genAnalysis(resource, instance)
	if err := r.ScheduleClient.CreateCraneAnalysis(ctx, resource.Namespace, analytics); err != nil {
		log.Error(err, "creat analysis error")
		return err
	}

	r.Recorder.Event(instance, corev1.EventTypeNormal, "CreateCraneAnalysis", fmt.Sprintf("Create CraneAnalysis %s/%s", resource.Namespace, resource.Name))
	return nil
}

func (r *AnalysisTaskReconciler) DeleteCraneAnalysis(ctx context.Context, resource corev1.ObjectReference, instance *v1alpha1.AnalysisTask) (err error) {
	log := r.Log.WithName("DeleteCraneAnalysis")
	if err = r.cleanWorkloadTaskLabel(ctx, resource, instance); err != nil {
		log.Error(err, "clean workload label error")
		return err
	}

	name := genAnalysisName(instance.Spec.Type, resource.Kind, resource.Name)
	err = r.ScheduleClient.DeleteCraneAnalysis(ctx, resource.Namespace, name)
	if err != nil {
		log.Error(err, "delete analysis error")
		return err
	}
	r.Recorder.Event(instance, corev1.EventTypeNormal, "DeleteCraneAnalysis", fmt.Sprintf("Delete CraneAnalysis %s/%s", resource.Namespace, name))
	return
}

func (r *AnalysisTaskReconciler) updateWorkloadTaskLabel(ctx context.Context, resource corev1.ObjectReference, instance *v1alpha1.AnalysisTask) error {
	log := r.Log.WithName("updateWorkloadTaskLabel")

	gvkr, err := ParseGVKR(r.RestMapper, resource.APIVersion, resource.Kind)
	if err != nil {
		log.Error(err, "Failed to parse Group, Version, Kind, Resource", "apiVersion", resource.APIVersion, "kind", resource.Kind)
	}
	gvkString := gvkr.GVKString()
	log.Info("Parsed Group, Version, Kind, Resource", "Name", resource.Name, "GVK", gvkString, "Resource", gvkr.Resource)

	unstruct := &unstructured.Unstructured{}
	unstruct.SetGroupVersionKind(gvkr.GroupVersionKind())
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: resource.Namespace, Name: resource.Name}, unstruct); err != nil {
		// resource doesn't exist
		log.Error(err, "Target resource doesn't exist", "resource", gvkString, "name", resource.Name)
		return client.IgnoreNotFound(err)
	}

	usCopy := unstruct.DeepCopy()
	labels := unstruct.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[constants.AnalysisTaskLabelKey] = instance.Name
	usCopy.SetLabels(labels)
	patch := client.MergeFrom(unstruct)
	if err = r.Client.Patch(ctx, usCopy, patch); err != nil {
		log.Error(err, "Failed to patch target workload", "apiVersion", resource.APIVersion, "kind", resource.Kind)
		return err
	}

	return nil
}

func (r *AnalysisTaskReconciler) cleanWorkloadTaskLabel(ctx context.Context, resource corev1.ObjectReference, instance *v1alpha1.AnalysisTask) error {
	log := r.Log.WithName("cleanWorkloadTaskLabel")

	gvkr, err := ParseGVKR(r.RestMapper, resource.APIVersion, resource.Kind)
	if err != nil {
		log.Error(err, "Failed to parse Group, Version, Kind, Resource", "apiVersion", resource.APIVersion, "kind", resource.Kind)
	}
	gvkString := gvkr.GVKString()
	log.Info("Parsed Group, Version, Kind, Resource", "Name", resource.Name, "GVK", gvkString, "Resource", gvkr.Resource)

	unstruct := &unstructured.Unstructured{}
	unstruct.SetGroupVersionKind(gvkr.GroupVersionKind())
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: resource.Namespace, Name: resource.Name}, unstruct); err != nil {
		// resource doesn't exist
		log.Error(err, "Target resource doesn't exist", "resource", gvkString, "name", resource.Name)
		return client.IgnoreNotFound(err)
	}

	labels := unstruct.GetLabels()
	if labels == nil {
		return nil
	}

	delete(labels, constants.AnalysisTaskLabelKey)
	unstruct.SetLabels(labels)
	if err := r.Client.Update(ctx, unstruct); err != nil {
		log.Error(err, "Failed to clean target workload", "apiVersion", resource.APIVersion, "kind", resource.Kind)
		return err
	}

	return nil
}

func (r *AnalysisTaskReconciler) updateNamespaceTaskLabel(ctx context.Context, namespace string, instance *v1alpha1.AnalysisTask) error {
	log := r.Log.WithName("updateNamespaceTaskLabel")
	ns, err := r.K8SClient.Kubernetes().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nsCopy := ns.DeepCopy()
	if nsCopy.Labels == nil {
		nsCopy.Labels = make(map[string]string, 0)
	}
	nsCopy.Labels[constants.AnalysisTaskLabelKey] = instance.Name
	patch := client.MergeFrom(ns)
	data, err := patch.Data(nsCopy)
	if err != nil {
		log.Error(err, "create patch failed")
		return err
	}

	// data == "{}", need not to patch
	if len(data) == 2 {
		return nil
	}

	_, err = r.K8SClient.Kubernetes().CoreV1().Namespaces().Patch(ctx, ns.Name, patch.Type(), data, metav1.PatchOptions{})
	if err != nil {
		log.Error(err, "patch namespace failed")
		return err
	}
	return nil
}

func (r *AnalysisTaskReconciler) cleanNamespaceTaskLabel(ctx context.Context, namespace string) error {
	log := r.Log.WithName("cleanNamespaceTaskLabel")
	ns, err := r.K8SClient.Kubernetes().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	nsCopy := ns.DeepCopy()
	if nsCopy.Labels == nil {
		nsCopy.Labels = make(map[string]string, 0)
	}
	delete(nsCopy.Labels, constants.AnalysisTaskLabelKey)
	patch := client.MergeFrom(ns)
	data, err := patch.Data(nsCopy)
	if err != nil {
		log.Error(err, "create patch failed")
		return err
	}

	// data == "{}", need not to patch
	if len(data) == 2 {
		return nil
	}

	_, err = r.K8SClient.Kubernetes().CoreV1().Namespaces().Patch(ctx, ns.Name, patch.Type(), data, metav1.PatchOptions{})
	if err != nil {
		log.Error(err, "patch namespace failed")
		return err
	}
	return nil
}

func (r *AnalysisTaskReconciler) updateNamespacesWorkloads(ctx context.Context, namespace string, value string) error {
	log := r.Log.WithName("updateNamespacesWorkloads")
	ns, err := r.K8SClient.Kubernetes().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	oldValue := Label(ns.Annotations, constants.AnalysisTaskWorkloadAnnotationKey)
	log.Info(fmt.Sprintf("overwrite namespace %s annotation %s,old =%s,new = %s", namespace, constants.AnalysisTaskWorkloadAnnotationKey, oldValue, value))
	nsCopy := ns.DeepCopy()
	if nsCopy.Annotations == nil {
		nsCopy.Annotations = make(map[string]string, 0)
	}
	if value != "" {
		nsCopy.Annotations[constants.AnalysisTaskWorkloadAnnotationKey] = value
	} else {
		delete(nsCopy.Annotations, constants.AnalysisTaskWorkloadAnnotationKey)
	}
	patch := client.MergeFrom(ns)
	data, err := patch.Data(nsCopy)
	if err != nil {
		log.Error(err, "create patch failed")
		return err
	}

	// data == "{}", need not to patch
	if len(data) == 2 {
		return nil
	}

	_, err = r.K8SClient.Kubernetes().CoreV1().Namespaces().Patch(ctx, namespace, patch.Type(), data, metav1.PatchOptions{})
	if err != nil {
		log.Error(err, "patch namespace failed")
		return err
	}
	return nil
}

func (r *AnalysisTaskReconciler) getNamespacesWorkloads(ctx context.Context, namespace string) string {
	ns, err := r.K8SClient.Kubernetes().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return ""
	}

	value := Label(ns.Annotations, constants.AnalysisTaskWorkloadAnnotationKey)
	return value
}

func (r *AnalysisTaskReconciler) updateAnalysisTaskStatus(ctx context.Context, instance *schedulev1alpha1.AnalysisTask, status schedulev1alpha1.Status) error {
	instance.Status.Status = status
	r.Recorder.Eventf(instance, corev1.EventTypeNormal, "Update", "update analysis task %s status to %s", instance.Name, status)
	err := r.Status().Update(ctx, instance)
	return err
}
