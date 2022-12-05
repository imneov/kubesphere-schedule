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

package options

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/core/v1"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	"kubesphere.io/scheduling/pkg/apis"
	"kubesphere.io/scheduling/pkg/apiserver"
	"kubesphere.io/scheduling/pkg/client/clientset/versioned/scheme"
	"kubesphere.io/scheduling/pkg/client/k8s"
	apiserverconfig "kubesphere.io/scheduling/pkg/config"
	"kubesphere.io/scheduling/pkg/informers"
	schedulev1 "kubesphere.io/scheduling/pkg/kapis/schedule/v1alpha1"
	genericoptions "kubesphere.io/scheduling/pkg/server/options"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ServerRunOptions struct {
	ConfigFile              string
	GenericServerRunOptions *genericoptions.ServerRunOptions
	*apiserverconfig.Config

	DebugMode bool
}

func NewServerRunOptions() *ServerRunOptions {
	s := &ServerRunOptions{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
		Config:                  apiserverconfig.New(),
	}

	return s
}

func (s *ServerRunOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("generic")
	fs.BoolVar(&s.DebugMode, "debug", false, "Don't enable this if you don't know what it means.")
	s.GenericServerRunOptions.AddFlags(fs, s.GenericServerRunOptions)

	fs = fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		fs.AddGoFlag(fl)
	})

	return fss
}

// const fakeInterface string = "FAKE"

// NewAPIServer creates an APIServer instance using given options
func (s *ServerRunOptions) NewAPIServer(stopCh <-chan struct{}) (*apiserver.APIServer, error) {
	apiServer := &apiserver.APIServer{
		Config: s.Config,
	}

	kubernetesClient, err := k8s.NewKubernetesClient(s.KubernetesOptions)
	if err != nil {
		return nil, err
	}
	apiServer.KubernetesClient = kubernetesClient

	informerFactory := informers.NewInformerFactories(kubernetesClient.Kubernetes(), kubernetesClient.Schedule(),
		kubernetesClient.ExtResources(),
		kubernetesClient.CraneResources(),
		kubernetesClient.ApiExtensions(),
		kubernetesClient.Dynamic())
	apiServer.InformerFactory = informerFactory

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", s.GenericServerRunOptions.InsecurePort),
	}

	apiServer.ScheduleClient = schedulev1.NewScheduleClient(informerFactory,
		apiServer.KubernetesClient.Kubernetes(),
		apiServer.KubernetesClient.Schedule(),
		apiServer.KubernetesClient.ExtResources(),
		apiServer.KubernetesClient.Dynamic(),
		s.ScheduleOptions, stopCh)

	if s.GenericServerRunOptions.SecurePort != 0 {
		certificate, err := tls.LoadX509KeyPair(s.GenericServerRunOptions.TlsCertFile, s.GenericServerRunOptions.TlsPrivateKey)
		if err != nil {
			return nil, err
		}

		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{certificate},
		}
		server.Addr = fmt.Sprintf(":%d", s.GenericServerRunOptions.SecurePort)
	}

	sch := scheme.Scheme
	_ = v1.SchemeBuilder.AddToScheme(sch)
	if err := apis.AddToScheme(sch); err != nil {
		klog.Fatalf("unable add APIs to scheme: %v", err)
	}

	// we create a manager for getting client and cache, although the manager is for creating controller. At last, we
	// won't start it up.
	m, err := manager.New(kubernetesClient.Config(), manager.Options{
		Scheme: sch,
		// disable metrics server needed by controller only
		MetricsBindAddress: "0",
	})
	if err != nil {
		klog.Errorf("unable to create manager for getting client and cache, err = %v", err)
		return nil, err
	}
	apiServer.Client = m.GetClient()
	apiServer.RuntimeCache = m.GetCache()
	apiServer.Server = server
	return apiServer, nil
}
