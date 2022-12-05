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
	"context"
	"encoding/json"
	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	schedulev1alpha1 "kubesphere.io/scheduling/api/schedule/v1alpha1"
	"kubesphere.io/scheduling/pkg/constants"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Mutating struct {
	*AnalysisTaskReconciler
	decoder *admission.Decoder
}

func (m *Mutating) Handle(ctx context.Context, request admission.Request) admission.Response {
	return m.AnalysisTaskReconciler.Mutating(ctx, request, m.decoder)
}

// InjectDecoder injects the decoder.
func (m *Mutating) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

type Validating struct {
	*AnalysisTaskReconciler
	decoder *admission.Decoder
}

// InjectDecoder injects the decoder.
func (v Validating) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

func (m *Validating) Handle(ctx context.Context, request admission.Request) admission.Response {
	return m.AnalysisTaskReconciler.Validating(ctx, request)
}

// set scheduler for workloads
func (r *AnalysisTaskReconciler) Mutating(ctx context.Context, request admission.Request, decoder *admission.Decoder) admission.Response {
	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Namespace: constants.KubesphereScheduleNamespace, Name: constants.KubesphereScheduleConfigMap}
	err := r.Get(ctx, key, cm)
	if err != nil {
		klog.Errorf("get ConfigMap error, %s", err.Error())
		if apierrors.IsNotFound(err) {
			return admission.Allowed("ConfigMap not found, skip")
		}
		return admission.Allowed("Get ConfigMap error, skip")
	}
	scheduleConfig := &schedulev1alpha1.ClusterScheduleConfig{}

	if data, ok := cm.Data["config"]; ok {
		err = json.Unmarshal([]byte(data), scheduleConfig)
		if err != nil {
			klog.Errorf("Unmarshal schedule config error, %s, skip", err.Error())
			return admission.Allowed("Unmarshal schedule config error, skip")
		}
	} else {
		klog.Errorf("schedule config do not exists, %s, skip")
		return admission.Allowed("Schedule config do not exists, skip")
	}

	defaultScheduler := scheduleConfig.DefaultScheduler
	if defaultScheduler == "kube-scheduler" {
		defaultScheduler = "default-scheduler"
	}
	klog.Info("Mutating webhook request ", " defaultScheduler ", defaultScheduler, " scheduleConfig ", scheduleConfig)

	var workload client.Object
	switch request.Kind.Kind {
	case "Deployment":
		var deploy = &appsv1.Deployment{}
		err := decoder.Decode(request, deploy)
		if err != nil {
			klog.Errorf("Decode Deployment error, %s", err.Error())
			return admission.Allowed("Decode Deployment error, skip.")
		}
		if deploy.Spec.Template.Spec.SchedulerName == "default-scheduler" {
			deploy.Spec.Template.Spec.SchedulerName = defaultScheduler
		}
		workload = deploy
	case "StatefulSet":
		var state = &appsv1.StatefulSet{}
		err := decoder.Decode(request, state)
		if err != nil {
			klog.Errorf("Decode StatefulSet error, %s", err.Error())
			return admission.Allowed("Decode StatefulSet error, skip.")
		}
		if state.Spec.Template.Spec.SchedulerName == "default-scheduler" {
			state.Spec.Template.Spec.SchedulerName = defaultScheduler
		}
		workload = state
	case "DaemonSet":
		var daemon = &appsv1.DaemonSet{}
		err := decoder.Decode(request, daemon)
		if err != nil {
			klog.Errorf("Decode DaemonSet error, %s", err.Error())
			return admission.Allowed("Decode DaemonSet error, skip.")
		}
		if daemon.Spec.Template.Spec.SchedulerName == "default-scheduler" {
			daemon.Spec.Template.Spec.SchedulerName = defaultScheduler
		}
		workload = daemon
	default:
		klog.Info("Kind do not match, pass.")
		return admission.Allowed("Kind do not match, pass.")
	}

	marshaled, err := json.Marshal(workload)
	if err != nil {
		klog.Error("marshal workload error", "error", err, "namespace", request.Namespace, "name", request.Name)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	klog.Info("patch success", string(marshaled))
	return admission.PatchResponseFromRaw(request.Object.Raw, marshaled)
}
