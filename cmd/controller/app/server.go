/*
Copyright 2019 The KubeSphere Authors.

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

package app

import (
	"context"
	"fmt"
	"github.com/google/gops/agent"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	"kubesphere.io/schedule/pkg/client/k8s"
	"kubesphere.io/schedule/pkg/constants"
	"kubesphere.io/schedule/pkg/informers"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"kubesphere.io/schedule/cmd/controller/app/options"
	"kubesphere.io/schedule/pkg/apis"
	controllerconfig "kubesphere.io/schedule/pkg/config"
)

func NewControllerManagerCommand() *cobra.Command {
	s := options.NewKubeSphereControllerManagerOptions()
	conf, err := controllerconfig.TryLoadFromDisk()
	if err == nil {
		// make sure LeaderElection is not nil
		s.KubernetesOptions = conf.KubernetesOptions
		s.ScheduleOptions = conf.ScheduleOptions
	} else {
		klog.Fatal("Failed to load configuration from disk", err)
	}

	cmd := &cobra.Command{
		Use:  "controller-manager",
		Long: `Schedule controller manager is a daemon that embeds the control loops shipped with KubeSphere.`,
		Run: func(cmd *cobra.Command, args []string) {
			if errs := s.Validate(allControllers); len(errs) != 0 {
				klog.Error(utilerrors.NewAggregate(errs))
				os.Exit(1)
			}

			if s.GOPSEnabled {
				// Add agent to report additional information such as the current stack trace, Go version, memory stats, etc.
				// Bind to a random port on address 127.0.0.1
				if err := agent.Listen(agent.Options{}); err != nil {
					klog.Fatal(err)
				}
			}

			if err = Run(s, controllerconfig.WatchConfigChange(), signals.SetupSignalHandler()); err != nil {
				klog.Error(err)
				os.Exit(1)
			}
		},
		SilenceUsage: true,
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags(allControllers)

	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})

	return cmd
}

func Run(s *options.KubeSphereControllerManagerOptions, configCh <-chan controllerconfig.Config, ctx context.Context) error {
	ictx, cancelFunc := context.WithCancel(context.TODO())
	errCh := make(chan error)
	defer close(errCh)
	go func() {
		if err := run(s, ictx); err != nil {
			errCh <- err
		}
	}()

	// The ctx (signals.SetupSignalHandler()) is to control the entire program life cycle,
	// The ictx(internal context)  is created here to control the life cycle of the controller-manager(all controllers, sharedInformer, webhook etc.)
	// when config changed, stop server and renew context, start new server
	for {
		select {
		case <-ctx.Done():
			cancelFunc()
			return nil
		case cfg := <-configCh:
			cancelFunc()
			s.MergeConfig(&cfg)
			ictx, cancelFunc = context.WithCancel(context.TODO())
			go func() {
				if err := run(s, ictx); err != nil {
					errCh <- err
				}
			}()
		case err := <-errCh:
			cancelFunc()
			return err
		}
	}
}

func run(s *options.KubeSphereControllerManagerOptions, ctx context.Context) error {

	kubernetesClient, err := k8s.NewKubernetesClient(s.KubernetesOptions)
	if err != nil {
		klog.Errorf("Failed to create kubernetes clientset %v", err)
		return err
	}

	informerFactory := informers.NewInformerFactories(
		kubernetesClient.Kubernetes(),
		kubernetesClient.Schedule(),
		kubernetesClient.ExtResources(),
		kubernetesClient.ExtResources(),
		kubernetesClient.ApiExtensions(),
		kubernetesClient.Dynamic())

	mgrOptions := manager.Options{
		CertDir:                s.WebhookCertDir,
		Port:                   8443,
		MetricsBindAddress:     s.GenericServerRunOptions.MetricsBindAddress,
		HealthProbeBindAddress: s.GenericServerRunOptions.HealthzBindAddress,
	}

	if s.LeaderElect {
		mgrOptions.LeaderElection = s.LeaderElect
		mgrOptions.LeaderElectionNamespace = constants.KubesphereScheduleNamespace
		mgrOptions.LeaderElectionID = constants.KubesphereScheduleElectionID
		mgrOptions.LeaseDuration = &s.LeaderElection.LeaseDuration
		mgrOptions.RetryPeriod = &s.LeaderElection.RetryPeriod
		mgrOptions.RenewDeadline = &s.LeaderElection.RenewDeadline
	}

	/*

		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "fa68b2a3.kubesphere.io",
	*/

	klog.V(0).Info("setting up manager")
	ctrl.SetLogger(klogr.New())
	// Use 8443 instead of 443 cause we need root permission to bind port 443
	mgr, err := manager.New(kubernetesClient.Config(), mgrOptions)
	if err != nil {
		klog.Fatalf("unable to set up overall controller manager: %v", err)
	}

	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Fatalf("unable add APIs to scheme: %v", err)
	}

	// register common meta types into schemas.
	metav1.AddToGroupVersion(mgr.GetScheme(), metav1.SchemeGroupVersion)

	// TODO(jeff): refactor config with CRD
	// install all controllers
	if err = addAllControllers(mgr,
		kubernetesClient,
		informerFactory,
		s,
		ctx.Done()); err != nil {
		klog.Fatalf("unable to register controllers to the manager: %v", err)
	}

	// Start cache data after all informer is registered
	klog.V(0).Info("Starting cache resource from apiserver...")

	informerFactory.Start(ctx.Done())

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	klog.V(0).Info("Starting the controllers.")
	if err = mgr.Start(ctx); err != nil {
		klog.Fatalf("unable to run the manager: %v", err)
	}

	return nil
}
