package main

import (
	"flag"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func main() {

	var imagePullSecretName, imagePullSecretNamespace string
	flag.StringVar(&imagePullSecretName, "image-pull-secret-name", "docker-hub-pull-secret", "Name of the pull secret to inject into pods")
	flag.StringVar(&imagePullSecretNamespace, "image-pull-secret-namespace", "kube-system", "Name of the pull secret to inject into pods")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	var log = logf.Log.WithName("pull-secrets-injector")

	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Port:                   9443,
		MetricsBindAddress:     ":8080",
		HealthProbeBindAddress: ":8081",
	})

	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}
	err = mgr.AddReadyzCheck("ready", func(_ *http.Request) error {
		return nil
		// use  once https://github.com/kubernetes-sigs/controller-runtime/pull/1124 is merged
		//if mgr.GetWebhookServer().Started {
		//  return nil
		//}
		//return errors.New("Webhook server not yet started")
	})
	if err != nil {
		log.Error(err, "could not add readiness check")
		os.Exit(1)
	}

	mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{Handler: &podMutator{
		ImagePullSecret: types.NamespacedName{Namespace: imagePullSecretNamespace, Name: imagePullSecretName},
		Client:          mgr.GetClient(),
		Log:             mgr.GetLogger()},
	})

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
