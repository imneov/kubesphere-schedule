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

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	servererr "kubesphere.io/scheduling/pkg/server/errors"

	restful "github.com/emicklei/go-restful"
	ext "github.com/gocrane/api/pkg/generated/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"kubesphere.io/scheduling/api"
	"kubesphere.io/scheduling/api/schedule/v1alpha1"
	"kubesphere.io/scheduling/pkg/apiserver/query"
	"kubesphere.io/scheduling/pkg/client/clientset/versioned"
	scheduleoptions "kubesphere.io/scheduling/pkg/client/schedule"
	"kubesphere.io/scheduling/pkg/constants"
	"kubesphere.io/scheduling/pkg/informers"
	"kubesphere.io/scheduling/pkg/models/schedule"
)

var (
	ErrScheduleClientNil = fmt.Errorf("schedule client is nil")
	ErrScheduleTypeNil   = fmt.Errorf("analysis task type error")
)

type scheduleHandler struct {
	schedule schedule.Operator
}

func NewScheduleClient(ksInformers informers.InformerFactory,
	k8sClient kubernetes.Interface,
	scheduleClient versioned.Interface,
	resourcesClient ext.Interface,
	dynamicClient dynamic.Interface, option *scheduleoptions.Options, stopCh <-chan struct{}) schedule.Operator {

	return schedule.NewScheduleOperator(ksInformers,
		k8sClient,
		scheduleClient,
		resourcesClient,
		stopCh)
}

func (h *scheduleHandler) ListScheduler(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	objs, err := h.schedule.GetScheduleConfig(ctx)
	handleResponse(request, response, objs, err)
}

func (h *scheduleHandler) ModifyScheduler(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	var schedulerConfig *schedule.SchedulerConfig
	err := request.ReadEntity(&schedulerConfig)
	if err != nil {
		klog.V(4).Infoln(err)
		api.HandleBadRequest(response, nil, err)
		return
	}

	objs, err := h.schedule.ModifySchedulerConfig(ctx, schedulerConfig)
	handleResponse(request, response, objs, err)
}

func (h *scheduleHandler) ListAnalysisTask(request *restful.Request, response *restful.Response) {
	//analysis := request.PathParameter("analysis")
	ctx := request.Request.Context()
	query := query.ParseQueryParameter(request)
	objs, err := h.schedule.ListAnalysisTask(ctx, query)
	handleResponse(request, response, objs, err)
}

func (h *scheduleHandler) CreateWorkloadAnalysis(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	namespace := request.PathParameter("namespace")
	var analysisTask *v1alpha1.AnalysisTask
	err := request.ReadEntity(&analysisTask)
	if err != nil {
		klog.V(4).Infoln(err)
		api.HandleBadRequest(response, nil, err)
		return
	}

	if analysisTask.Spec.Type != v1alpha1.WorkloadResourceType {
		klog.V(4).Infoln(ErrScheduleTypeNil)
		api.HandleBadRequest(response, nil, ErrScheduleTypeNil)
		return
	}

	ret, err := h.schedule.CreateAnalysisTask(ctx, namespace, analysisTask)
	handleResponse(request, response, ret, err)
}

func (h *scheduleHandler) CreateNamespaceAnalysis(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	var analysisTask *v1alpha1.AnalysisTask
	err := request.ReadEntity(&analysisTask)
	if err != nil {
		klog.V(4).Infoln(err)
		api.HandleBadRequest(response, nil, err)
		return
	}

	if analysisTask.Spec.Type != v1alpha1.NamespaceResourceType {
		klog.V(4).Infoln(ErrScheduleTypeNil)
		api.HandleBadRequest(response, nil, ErrScheduleTypeNil)
		return
	}

	ret, err := h.schedule.CreateAnalysisTask(ctx, constants.KubesphereScheduleNamespace, analysisTask)
	handleResponse(request, response, ret, err)
}

func (h *scheduleHandler) ModifyAnalysisTask(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	analysis := request.PathParameter("analysis")
	namespace := request.PathParameter("namespace")
	var analysisTask *v1alpha1.AnalysisTask
	err := request.ReadEntity(&analysisTask)
	if err != nil {
		klog.V(4).Infoln(err)
		api.HandleBadRequest(response, nil, err)
		return
	}

	err = h.schedule.ModifyAnalysisTask(ctx, namespace, analysis, analysisTask)
	handleResponse(request, response, servererr.None, err)
}

func (h *scheduleHandler) DescribeAnalysisTask(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	analysis := request.PathParameter("analysis")
	namespace := request.PathParameter("namespace")

	ret, err := h.schedule.DescribeAnalysisTask(ctx, namespace, analysis)
	handleResponse(request, response, ret, err)
}

func (h *scheduleHandler) DeleteAnalysisTask(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	analysis := request.PathParameter("analysis")
	namespace := request.PathParameter("namespace")

	err := h.schedule.DeleteAnalysisTask(ctx, namespace, analysis)
	handleResponse(request, response, servererr.None, err)
}

func (h *scheduleHandler) ModifyAnalysisTaskConfig(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()
	var analysisConfig *schedule.AnalysisTaskConfig
	err := request.ReadEntity(&analysisConfig)
	if err != nil {
		klog.V(4).Infoln(err)
		api.HandleBadRequest(response, nil, err)
		return
	}

	ret, err := h.schedule.ModifyAnalysisTaskConfig(ctx, analysisConfig)
	handleResponse(request, response, ret, err)
}

func handleResponse(req *restful.Request, resp *restful.Response, obj interface{}, err error) {
	if err != nil {
		klog.Error(err)
		if errors.IsNotFound(err) {
			api.HandleNotFound(resp, req, err)
			return
		} else if errors.IsConflict(err) {
			api.HandleConflict(resp, req, err)
			return
		}
		api.HandleBadRequest(resp, req, err)
		return
	}

	_ = resp.WriteEntity(obj)
}
