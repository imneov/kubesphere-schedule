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

package k8s

import (
	"errors"

	cranev1 "github.com/gocrane/api/pkg/generated/clientset/versioned"
	ext "github.com/gocrane/api/pkg/generated/clientset/versioned"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	schedule "kubesphere.io/scheduling/pkg/client/clientset/versioned"
)

type Client interface {
	Kubernetes() kubernetes.Interface
	Dynamic() dynamic.Interface
	Schedule() schedule.Interface
	ApiExtensions() apiextensionsclient.Interface
	CraneResources() cranev1.Interface
	ExtResources() ext.Interface
	Discovery() discovery.DiscoveryInterface
	Master() string
	Config() *rest.Config
}

type kubernetesClient struct {
	// kubernetes client interface
	k8s kubernetes.Interface
	// discovery client
	discoveryClient *discovery.DiscoveryClient

	// generated clientset
	schedule      schedule.Interface
	crane         cranev1.Interface
	apiextensions apiextensionsclient.Interface
	ext           ext.Interface
	master        string
	config        *rest.Config
	dynamic       dynamic.Interface
}

func (k *kubernetesClient) CraneResources() cranev1.Interface {
	return k.crane
}

func (k *kubernetesClient) Dynamic() dynamic.Interface {
	return k.dynamic
}

func (k *kubernetesClient) Schedule() schedule.Interface {
	return k.schedule
}

// NewKubernetesClientWithConfig creates a k8s client with the rest config
func NewKubernetesClientWithConfig(config *rest.Config) (client Client, err error) {
	if config == nil {
		return
	}

	var k kubernetesClient
	if k.k8s, err = kubernetes.NewForConfig(config); err != nil {
		return
	}

	if k.dynamic, err = dynamic.NewForConfig(config); err != nil {
		return
	}

	//gvr := schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	//_, err = vClusterDynamicCli.Resource(gvr).Get(ctx, ksClusterConfigurationCRDName, metav1.GetOptions{})
	//if err != nil {
	//	if !apierrors.IsNotFound(err) {
	//		logger.Error(err, "Failed to get customresourcedefinitions crd in dolphincluster")
	//		return err
	//	}
	//	customresourcedefinitionCRD := r.clusterconfigurationCRDForKs(dolphinCluster)
	//	_, err = vClusterDynamicCli.Resource(gvr).Create(ctx, customresourcedefinitionCRD, metav1.CreateOptions{})
	//	if err != nil {
	//		logger.Error(err, "Failed to create customresourcedefinitions crd in dolphincluster")
	//		return err
	//	}
	//}

	if k.schedule, err = schedule.NewForConfig(config); err != nil {
		return
	}

	if k.crane, err = cranev1.NewForConfig(config); err != nil {
		return
	}

	if k.discoveryClient, err = discovery.NewDiscoveryClientForConfig(config); err != nil {
		return
	}

	if k.apiextensions, err = apiextensionsclient.NewForConfig(config); err != nil {
		return
	}

	if k.ext, err = ext.NewForConfig(config); err != nil {
		return
	}

	k.config = config
	client = &k
	return
}

// NewKubernetesClientWithToken creates a k8s client with a bearer token
func NewKubernetesClientWithToken(token string, master string) (client Client, err error) {
	if token == "" {
		return nil, errors.New("token required")
	}

	return NewKubernetesClientWithConfig(&rest.Config{
		BearerToken: token,
		Host:        master,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	})
}

// NewKubernetesClient creates a KubernetesClient
func NewKubernetesClient(options *KubernetesOptions) (client Client, err error) {
	if options == nil {
		return
	}

	var config *rest.Config
	if config, err = clientcmd.BuildConfigFromFlags("", options.KubeConfig); err != nil {
		return
	}

	config.QPS = options.QPS
	config.Burst = options.Burst

	if client, err = NewKubernetesClientWithConfig(config); err == nil {
		if k8sClient, ok := client.(*kubernetesClient); ok {
			k8sClient.config = config
			k8sClient.master = options.Master
		}
	}
	return
}

func (k *kubernetesClient) Kubernetes() kubernetes.Interface {
	return k.k8s
}

func (k *kubernetesClient) Discovery() discovery.DiscoveryInterface {
	return k.discoveryClient
}

func (k *kubernetesClient) ApiExtensions() apiextensionsclient.Interface {
	return k.apiextensions
}

func (k *kubernetesClient) ExtResources() ext.Interface {
	return k.ext
}

// Master address used to generate kubeconfig for downloading
func (k *kubernetesClient) Master() string {
	return k.master
}

func (k *kubernetesClient) Config() *rest.Config {
	return k.config
}
