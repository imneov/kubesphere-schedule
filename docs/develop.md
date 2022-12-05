## 代码结构

### Operator代码目录结构

`cmd`是一个标准的Go main package目录。`controller`目录下是一些controller的实现，`apis`目录下是一些CRD的定义，`pkg`
目录下是一些Operator的功能实现。

| 目录             | 说明            |
|----------------|---------------|
| api            | CR 定义         |
| cmd/controller | controller入口  |
| pkg/apis       | CR 导入         |
| pkg/controller | controller 实现 |
| pkg/kapis      | API 的 HTTP 封装 |
| pkg/model      | API 的 业务实现    |


#### 目录示例

```
.
├── cmd
│  ├── apiserver
│  │  ├── apiserver.go
│  │  └── app
│  │      ├── options
│  │      │  ├── options.go
│  │      │  └── validation.go
│  │      └── server.go
│  └── controller-manager
│      ├── app
│      │  ├── controllers.go
│      │  ├── helper.go
│      │  ├── options
│      │  │  ├── options.go
│      │  │  └── options_test.go
│      │  └── server.go
│      └── controller-manager.go
├── go.mod
├── go.sum
├── pkg
│  ├── api
│  │  ├── application
│  │  │  ├── OWNERS
│  │  │  ├── crdinstall
│  │  │  │  └── install.go
│  │  │  ├── group.go
│  │  │  └── v1alpha1
│  │  │      ├── constants.go
│  │  │      ├── doc.go
│  │  │      ├── helmapplication_types.go
│  │  │      ├── helmapplicationversion_types.go
│  │  │      ├── helmcategory_types.go
│  │  │      ├── helmrelease_types.go
│  │  │      ├── helmrepo_types.go
│  │  │      ├── register.go
│  │  │      └── zz_generated.deepcopy.go
│  │  ├── cluster
│  │  │  ├── group.go
│  │  │  └── v1alpha1
│  │  │      ├── cluster_types.go
│  │  │      ├── doc.go
│  │  │      ├── register.go
│  │  │      └── zz_generated.deepcopy.go
│  │  ├── types.go
│  │  └── utils.go
│  ├── apis
│  │  ├── addtoscheme_application_v1alpha1.go
│  │  ├── addtoscheme_cluster_v1alpha1.go
│  │  └── apis.go
│  ├── apiserver
│  │  ├── apiserver.go
│  │  ├── filters
│  │  │  ├── authentication.go
│  │  │  ├── kubeapiserver.go
│  │  │  └── requestinfo.go
│  │  ├── query
│  │  │  ├── field.go
│  │  │  ├── types.go
│  │  │  └── types_test.go
│  │  ├── request
│  │  │  ├── context.go
│  │  │  ├── context_test.go
│  │  │  ├── requestinfo.go
│  │  │  └── requestinfo_test.go
│  │  └── runtime
│  │      ├── runtime.go
│  │      └── runtime_test.go
│  ├── client
│  │  ├── clientset
│  │  │  └── versioned
│  │  │      └── .....  
│  │  ├── informers
│  │  │  └── externalversions
│  │  │      ├── application
│  │  │      │  ├── interface.go
│  │  │      │  └── v1alpha1
│  │  │      │      ├── helmapplication.go
│  │  │      │      ├── helmapplicationversion.go
│  │  │      │      ├── helmcategory.go
│  │  │      │      ├── helmrelease.go
│  │  │      │      ├── helmrepo.go
│  │  │      │      └── interface.go
│  │  │      ├── cluster
│  │  │      │  ├── interface.go
│  │  │      │  └── v1alpha1
│  │  │      │      ├── cluster.go
│  │  │      │      └── interface.go
│  │  │      ├── factory.go
│  │  │      ├── generic.go
│  │  │      └── internalinterfaces
│  │  │          └── factory_interfaces.go
│  │  ├── k8s
│  │  │  ├── fake_client.go
│  │  │  ├── kubernetes.go
│  │  │  ├── kubernetes_test.go
│  │  │  ├── options.go
│  │  │  └── options_test.go
│  │  ├── listers
│  │  │  ├── application
│  │  │  │  └── v1alpha1
│  │  │  │      ├── expansion_generated.go
│  │  │  │      ├── helmapplication.go
│  │  │  │      ├── helmapplicationversion.go
│  │  │  │      ├── helmcategory.go
│  │  │  │      ├── helmrelease.go
│  │  │  │      └── helmrepo.go
│  │  │  └── cluster
│  │  │      └── v1alpha1
│  │  │          ├── cluster.go
│  │  │          └── expansion_generated.go
│  │  ├── schedule
│  │  │  ├── OWNERS
│  │  │  ├── helmrepoindex
│  │  │  │  ├── interface.go
│  │  │  │  ├── load_chart.go
│  │  │  │  ├── load_package.go
│  │  │  │  ├── repo_index.go
│  │  │  │  └── repo_index_test.go
│  │  │  ├── helmwrapper
│  │  │  │  ├── helm_wrapper.go
│  │  │  │  ├── helm_wrapper_darwin.go
│  │  │  │  ├── helm_wrapper_linux.go
│  │  │  │  └── helm_wrapper_test.go
│  │  │  └── options.go
│  │  └── s3
│  │      ├── fake
│  │      │  ├── fakes3.go
│  │      │  └── fakes3_test.go
│  │      ├── interface.go
│  │      ├── options.go
│  │      ├── s3.go
│  │      └── s3_test.go
│  ├── config
│  │  └── config.go
│  ├── constants
│  │  └── constants.go
│  ├── controller
│  │  └── schedule
│  │      ├── OWNERS
│  │      ├── helmapplication
│  │      │  ├── helm_application_controller.go
│  │      │  ├── helm_application_controller_suite_test.go
│  │      │  ├── helm_application_controller_test.go
│  │      │  ├── helm_application_version_controller.go
│  │      │  └── metrics.go
│  │      ├── helmcategory
│  │      │  ├── helm_category_controller.go
│  │      │  ├── helm_category_controller_suite_test.go
│  │      │  └── helm_category_controller_test.go
│  │      ├── helmrelease
│  │      │  ├── get_chart_data.go
│  │      │  └── helm_release_controller.go
│  │      └── helmrepo
│  │          ├── helm_repo_controller.go
│  │          ├── helm_repo_controller_suite_test.go
│  │          └── helm_repo_controller_test.go
│  ├── informers
│  │  ├── informers.go
│  │  └── null_informers.go
│  ├── kapis
│  │  └── schedule
│  │      ├── OWNERS
│  │      ├── v1
│  │      │  ├── handler.go
│  │      │  └── register.go
│  │      └── v2alpha1
│  │          ├── handler.go
│  │          └── register.go
│  ├── models
│  │  ├── schedule
│  │  │  ├── OWNERS
│  │  │  ├── applications.go
│  │  │  ├── applications_test.go
│  │  │  ├── applicationversions.go
│  │  │  ├── attachments.go
│  │  │  ├── categories.go
│  │  │  ├── category_test.go
│  │  │  ├── errors.go
│  │  │  ├── interface.go
│  │  │  ├── release.go
│  │  │  ├── release_test.go
│  │  │  ├── repo_test.go
│  │  │  ├── repos.go
│  │  │  ├── types.go
│  │  │  ├── utils.go
│  │  │  └── v2alpha1
│  │  │      ├── applications.go
│  │  │      ├── applicationsversions.go
│  │  │      ├── categories.go
│  │  │      ├── interface.go
│  │  │      ├── release.go
│  │  │      └── repos.go
│  │  ├── resources
│  │  │  └── v1alpha3
│  │  │      ├── application
│  │  │      │  ├── applications.go
│  │  │      │  └── applications_test.go
│  │  │      ├── interface.go
│  │  │      ├── interface_test.go
│  │  │      └── schedule
│  │  │          ├── OWNERS
│  │  │          ├── application
│  │  │          │  └── applications.go
│  │  │          ├── applicationversion
│  │  │          │  └── applicationsversions.go
│  │  │          ├── category
│  │  │          │  └── category.go
│  │  │          ├── helmrelease
│  │  │          │  └── releases.go
│  │  │          └── repo
│  │  │              └── repo.go
│  │  └── types.go
│  ├── server
│  │  ├── errors
│  │  │  └── errors.go
│  │  ├── options
│  │  │  └── options.go
│  │  └── params
│  │      ├── params.go
│  │      └── params_test.go
│  └── utils
│      ├── clusterclient
│      │  └── clusterclient.go
│      ├── idutils
│      │  ├── id_utils.go
│      │  └── id_utils_test.go
│      ├── net
│      │  ├── net.go
│      │  └── net_test.go
│      ├── reflectutils
│      │  ├── deep.go
│      │  └── reflect.go
│      ├── reposcache
│      │  └── repo_cahes.go
│      ├── resourceparse
│      │  └── resource_parse.go
│      ├── sliceutil
│      │  └── sliceutils.go
│      └── stringutils
              └── string.go
└── plugins
└── crds
└── apiservice.yaml
```


### 启动时报错

```
I1029 10:30:59.032622   98782 controller.go:165] controller/analysis "msg"="Starting EventSource" "reconciler group"="scheduling.kubesphere.io" "reconciler kind"="Analysis" "source"={"Type":{"metadata":{"creationTimestamp":null},"spec":{"resourceSelectors":null,"completionStrategy":{}},"status":{}}}
I1029 10:30:59.032668   98782 controller.go:173] controller/analysis "msg"="Starting Controller" "reconciler group"="scheduling.kubesphere.io" "reconciler kind"="Analysis" 
E1029 10:30:59.241859   98782 deleg.go:144] controller-runtime/source "msg"="if kind is a CRD, it should be installed before calling Start" "error"="no matches for kind \"Analysis\" in version \"scheduling.kubesphere.io/v1alpha1\""  "kind"={"Group":"scheduling.kubesphere.io","Kind":"Analysis"}
E1029 10:30:59.241932   98782 controller.go:190] controller/analysis "msg"="Could not wait for Cache to sync" "error"="failed to wait for analysis caches to sync: no matches for kind \"Analysis\" in version \"scheduling.kubesphere.io/v1alpha1\"" "reconciler group"="scheduling.kubesphere.io" "reconciler kind"="Analysis" 
F1029 10:30:59.242041   98782 server.go:198] unable to run the manager: failed to wait for analysis caches to sync: no matches for kind "Analysis" in version "scheduling.kubesphere.io/v1alpha1"
Exiting.
```

```
cd /Users/neov/src/CNCF/kubesphere-schedule\n
kubectl apply -f config/samples/schedule_v1alpha1_analysis.yaml
```
安装 CR 后修复

### Crane 安装后不生效

charts/crane/templates/scheduler-prometheus-rules.yaml
需要修改为 
```
namespace: {{ .Values.global.prometheusNameSpace }}
```
该值应该为：
```
global:
  prometheusAddr: http://prometheus-k8s.kubesphere-monitoring-system.svc.cluster.local:9090   # prometheus address
  prometheusNameSpace: kubesphere-monitoring-system
```


### 找不到新增的 CR
```
E1030 12:24:15.916419   79293 controller.go:304] controller/analysis "msg"="Reconciler error" "error"="no kind is registered for the type v1alpha1.NamespaceAnalysis in scheme \"k8s.io/client-go/kubernetes/scheme/register.go:72\"" "name"="analysis-sample" "namespace"="default" "reconciler group"="scheduling.kubesphere.io" "reconciler kind"="Analysis" 
I1030 12:24:18.477386   79293 namespace_analysis_controller.go:69] [+]---4---
E1030 12:24:18.477581   79293 controller.go:304] controller/analysis "msg"="Reconciler error" "error"="no kind is registered for the type v1alpha1.NamespaceAnalysis in scheme \"k8s.io/client-go/kubernetes/scheme/register.go:72\"" "name"="analysis-sample" "namespace"="default" "reconciler group"="scheduling.kubesphere.io" "reconciler kind"="Analysis" 
```

Add `api/scheduling/v1alpha1/types.go:126`
```go
func init() {
	...
	SchemeBuilder.Register(&NamespaceAnalysis{}, &NamespaceAnalysisList{})
}
```


### 生成的 CR 缺少字段

缺少 `DeepCopyInto`

增加
```
// +kubebuilder:resource:shortName=analysis
```

完整的如下：
```go
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=analysis
// +kubebuilder:subresource:status

// Analysis is the Schema for the analyses API
type Analysis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnalysisSpec   `json:"spec,omitempty"`
	Status AnalysisStatus `json:"status,omitempty"`
}
```

### 通过 Object 获取 BVK

// GVKForObject finds the GroupVersionKind associated with the given object, if there is only a single such GVK.
func GVKForObject(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
// TODO(directxman12): do we want to generalize this to arbitrary container types?
// I think we'd need a generalized form of scheme or something.  It's a
// shame there's not a reliable "GetGVK" interface that works by default
// for unpopulated static types and populated "dynamic" types
// (unstructured, partial, etc)


### IDEA Http

```
> {%
client.get("
// client.global.set("auth_token", response.body.result.token);
%}


Authorization: Bearer {{auth_token}}
```


### Config

clusterconfigurations/ks-installer
```yaml

	scheduler:
		enable: false
		defaultScheduler: crane-scheduler
		schedulers:
		- kube-scheduler
		- crane-scheduler
		analysis:
		    notifyThreshold:
		        cpu: 20
		        mem: 20

```