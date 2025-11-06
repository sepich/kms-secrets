/*


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

package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"

	secretv1beta1 "github.com/sepich/kms-secrets/api/v1beta1"
	"github.com/sepich/kms-secrets/controllers"
	"github.com/spf13/pflag"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = secretv1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var namespaced bool
	pflag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.BoolVar(&namespaced, "namespaced", false, "Only watch KMSSecret in the current namespace")
	var logLevel = pflag.StringP("log-level", "l", "info", "Log level to use (debug, info)")
	pflag.Parse()

	logger := getLogger(*logLevel)
	ctrl.SetLogger(logr.FromSlogHandler(logger.Handler()))

	ns := ""
	if namespaced {
		ns = os.Getenv("POD_NAMESPACE")
		if len(ns) == 0 {
			// Fall back to the namespace associated with the service account token, if available
			if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
				ns = strings.TrimSpace(string(data))
			}
		}
		if len(ns) == 0 {
			logger.Error("Mode is namespaced, but unable to determine own namespace name, please set env 'POD_NAMESPACE'")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "e976aec6.h3poteto.dev",
		Namespace:          ns,
	})
	if err != nil {
		logger.Error("unable to start manager", "error", err)
		os.Exit(1)
	}

	if err = (&controllers.KMSSecretReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("KMSSecret"),
		Recorder: mgr.GetEventRecorderFor("kms-secret"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		logger.Error("unable to create controller", "controller", "KMSSecret", "error", err)
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error("problem running manager", "error", err)
		os.Exit(1)
	}
}

func getLogger(logLevel string) *slog.Logger {
	var l = slog.LevelInfo
	if logLevel == "debug" {
		l = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     l,
		AddSource: logLevel == "debug",
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			if a.Key == slog.SourceKey {
				s := a.Value.String()
				i := strings.LastIndex(s, "/")
				j := strings.LastIndex(s, " ")
				a.Value = slog.StringValue(s[i+1:j] + ":" + s[j+1:len(s)-1])
			}
			if a.Key == slog.LevelKey {
				a.Value = slog.StringValue(strings.ToLower(a.Value.String()))
			}
			return a
		},
	}))
}
