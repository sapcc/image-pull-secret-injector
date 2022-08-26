package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestInject(t *testing.T) {
	type test struct {
		name            string
		secretNamespace string
		secretName      string
		secretData      map[string][]byte
		request         admission.Request
		compare         func(*testing.T, admission.Response, test)
		checkResources  func(*testing.T, client.Client, test)
	}

	tests := []test{
		{
			name:            "Normal same namespace",
			secretNamespace: "default",
			secretName:      "secret",
			request: admission.Request{
				AdmissionRequest: v1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Version: "v1",
						Kind:    "pods",
					},
					Operation: v1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`{
		    "apiVersion": "v1",
		    "kind": "Pod",
		    "metadata": {
		        "name": "foo",
		        "namespace": "default"
		    },
		    "spec": {
		        "containers": [
		            {
		                "image": "bar:v2",
		                "name": "bar"
		            }
		        ]
		    }
		}`),
					},
				},
			},
			compare: func(t *testing.T, r admission.Response, tc test) {
				for _, p := range r.Patches {
					fmt.Println(p)
					if p.Path == "/spec/imagePullSecrets" {
						assert.Equalf(t, tc.secretName, p.Value.([]interface{})[0].(map[string]interface{})["name"].(string), "Test case %v failed", tc.name)
						return
					}
				}
				assert.Failf(t, "Test case %v failed. Expected patch not found", tc.name)
			},
		},
		{
			name:            "Ghcr.io same namespace",
			secretNamespace: "default",
			secretName:      "secret",
			request: admission.Request{
				AdmissionRequest: v1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Version: "v1",
						Kind:    "pods",
					},
					Operation: v1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`{
		    "apiVersion": "v1",
		    "kind": "Pod",
		    "metadata": {
		        "name": "foo",
		        "namespace": "default"
		    },
		    "spec": {
		        "containers": [
		            {
		                "image": "ghcr.io/bar:v2",
		                "name": "bar"
		            }
		        ]
		    }
		}`),
					},
				},
			},
			compare: func(t *testing.T, r admission.Response, tc test) {
				for _, p := range r.Patches {
					fmt.Println(p)
					if p.Path == "/spec/imagePullSecrets" {
						assert.Equalf(t, tc.secretName, p.Value.([]interface{})[0].(map[string]interface{})["name"].(string), "Test case %v failed", tc.name)
						return
					}
				}
				assert.Failf(t, "Test case %v failed. Expected patch not found", tc.name)
			},
		},
		{
			name:            "not specified same namespace",
			secretNamespace: "default",
			secretName:      "secret",
			request: admission.Request{
				AdmissionRequest: v1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Version: "v1",
						Kind:    "pods",
					},
					Operation: v1.Create,
					Object: runtime.RawExtension{
						Raw: []byte(`{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "name": "foo",
        "namespace": "default"
    },
    "spec": {
        "containers": [
            {
                "image": "fake-registry.com/bar:v2",
                "name": "bar"
            }
        ]
    }
}`),
					},
				},
			},
			compare: func(t *testing.T, r admission.Response, tc test) {
				for _, p := range r.Patches {
					fmt.Println(p)
					if p.Path == "/spec/imagePullSecrets" {
						assert.Failf(t, "Test case %v failed. Unexpected patch found", tc.name)
					}
				}
			},
		},
		{
			name:            "Normal different namespace",
			secretNamespace: "default",
			secretName:      "secret",
			secretData: map[string][]byte{
				"secret": []byte("data"),
			},
			request: admission.Request{
				AdmissionRequest: v1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Version: "v1",
						Kind:    "pods",
					},
					Operation: v1.Create,
					Namespace: "non-default",
					Object: runtime.RawExtension{
						Raw: []byte(`{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "name": "foo",
        "namespace": "non-default"
    },
    "spec": {
        "containers": [
            {
                "image": "bar:v2",
                "name": "bar"
            }
        ]
    }
}`),
					},
				},
			},
			compare: func(t *testing.T, r admission.Response, tc test) {
				for _, p := range r.Patches {
					fmt.Println(p)
					if p.Path == "/spec/imagePullSecrets" {
						assert.Equalf(t, tc.secretName, p.Value.([]interface{})[0].(map[string]interface{})["name"].(string), "Test case %v failed", tc.name)
						return
					}
				}
				assert.Failf(t, "Test case %v failed. Expected patch not found", tc.name)
			},
			checkResources: func(t *testing.T, c client.Client, tc test) {
				// Assert that the secret got created
				secret := new(corev1.Secret)
				err := c.Get(context.TODO(), types.NamespacedName{Namespace: "non-default", Name: "secret"}, secret)
				assert.Equalf(t, tc.secretData, secret.Data, "Test case %v failed", tc.name)
				assert.Equalf(t, nil, err, "Test case %v failed", tc.name)
			},
		},
		{
			name:            "not specified different namespace",
			secretNamespace: "default",
			secretName:      "secret",
			secretData: map[string][]byte{
				"secret": []byte("data"),
			},
			request: admission.Request{
				AdmissionRequest: v1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Version: "v1",
						Kind:    "pods",
					},
					Operation: v1.Create,
					Namespace: "non-default",
					Object: runtime.RawExtension{
						Raw: []byte(`{
    "apiVersion": "v1",
    "kind": "Pod",
    "metadata": {
        "name": "foo",
        "namespace": "non-default"
    },
    "spec": {
        "containers": [
            {
                "image": "fake-registry.com/bar:v2",
                "name": "bar"
            }
        ]
    }
}`),
					},
				},
			},
			compare: func(t *testing.T, r admission.Response, tc test) {
				for _, p := range r.Patches {
					fmt.Println(p)
					if p.Path == "/spec/imagePullSecrets" {
						assert.Failf(t, "Test case %v failed. Unexpected patch found", tc.name)
					}
				}
			},
			checkResources: func(t *testing.T, c client.Client, tc test) {
				// Assert that the secret got created
				secret := new(corev1.Secret)
				err := c.Get(context.TODO(), types.NamespacedName{Namespace: "non-default", Name: "secret"}, secret)
				assert.Truef(t, apierrors.IsNotFound(err), "Test case %v failed", tc.name)
			},
		},
	}

	for _, tc := range tests {
		client := fake.NewClientBuilder().Build()
		mtr := &PodMutator{
			ImagePullSecret: types.NamespacedName{Namespace: tc.secretNamespace, Name: tc.secretName},
			Client:          client,
			Log:             logf.Log.WithName("pull-secrets-injector"),
			Registries:      []string{"docker.io", "ghcr.io"},
		}

		client.Create(context.TODO(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: tc.secretName, Namespace: tc.secretNamespace}, Data: tc.secretData})

		decoder, err := admission.NewDecoder(runtime.NewScheme())
		assert.Equalf(t, nil, err, "Test case %v failed", tc.name)
		mtr.InjectDecoder(decoder)

		resp := mtr.Handle(context.TODO(), tc.request)
		tc.compare(t, resp, tc)

		if tc.checkResources != nil {
			tc.checkResources(t, client, tc)
		}

		// Cleanup
		client.DeleteAllOf(context.TODO(), &corev1.Namespace{})
	}
}
