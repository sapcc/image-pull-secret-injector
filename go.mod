module github.com/sapcc/pull-secrets-injector

go 1.14

require (
	github.com/docker/distribution v2.7.1+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/opencontainers/go-digest v1.0.0 // indirect
	go.uber.org/zap v1.10.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)
