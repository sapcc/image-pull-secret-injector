package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type podMutator struct {
	Log             logr.Logger
	Client          client.Client
	ImagePullSecret types.NamespacedName
	decoder         *admission.Decoder
}

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create,versions=v1,name=mpod.kb.io

func (a *podMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	name := req.Name
	if name == "" {
		name = pod.Name
	}
	if name == "" {
		name = pod.GenerateName + "[SERVER GENERATED]"
	}

	a.injectImagePullSecret(ctx, pod, req.Namespace, name)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (a *podMutator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

func (a *podMutator) injectImagePullSecret(ctx context.Context, pod *corev1.Pod, namespace, name string) {

	//if the pod already has an imagePullSecret we have nothing todo
	if pod.Spec.ImagePullSecrets != nil && len(pod.Spec.ImagePullSecrets) > 0 {
		return
	}

	dockerHubImageFound := false
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if MatchImageHostname(container.Image, "docker.io") {
			dockerHubImageFound = true
			break
		}
	}

	if dockerHubImageFound {
		if err := a.ensurePullSecretInNamespace(ctx, namespace); err != nil {
			a.Log.Error(err, "Failed to ensure image pull secret", "namespace", namespace)
			return
		}
		a.Log.Info("injecting image pull secret into pod", "namespace", namespace, "name", name)
		pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: a.ImagePullSecret.Name}}
	}
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update

func (a *podMutator) ensurePullSecretInNamespace(ctx context.Context, namespace string) error {
	//skip if we are in the namespace containing the original image pull secret
	if a.ImagePullSecret.Namespace == namespace {
		return nil
	}

	pullSecret := new(corev1.Secret)
	if err := a.Client.Get(ctx, a.ImagePullSecret, pullSecret); err != nil {
		return fmt.Errorf("Failed to get original pull secret %s: %w", a.ImagePullSecret, err)
	}

	localPullSecret := new(corev1.Secret)
	if err := a.Client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: pullSecret.Name}, localPullSecret); err != nil {
		if apierrors.IsNotFound(err) {

			pullSecret.Namespace = namespace
			pullSecret.ResourceVersion = ""
			a.Log.Info("Creating pull secret", "namespace", namespace, "name", pullSecret.Name)
			return a.Client.Create(ctx, pullSecret)
		}
		return fmt.Errorf("Failed to get pull secret %s/%s: %w", namespace, pullSecret.Name, err)
	}
	if !reflect.DeepEqual(pullSecret.Data, localPullSecret.Data) {
		a.Log.Info("Update pull secret", "namespace", namespace, "name", pullSecret.Name)
		localPullSecret.Data = pullSecret.Data
		return a.Client.Update(ctx, localPullSecret)
	}
	return nil
}

func MatchImageHostname(image, hostname string) bool {
	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return false
	}
	named, err := reference.ParseNamed(ref.String())
	if err != nil {
		return false
	}
	if hostname == "index.docker.io" {
		hostname = "docker.io"
	}
	return reference.Domain(named) == hostname
}
