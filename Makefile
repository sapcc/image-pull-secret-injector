IMAGE:= sapcc/pull-secret-injector
VERSION:=0.3.0

manifests: controller-gen
	$(CONTROLLER_GEN) paths="./..." webhook rbac:roleName=webhook-server

deploy: 
	cd config/mutator && kustomize edit set image controller=$(IMAGE):$(VERSION)
	kustomize build config/default | kubectl apply -f -

uninstall:
	kustomize build config/default | kubectl delete -f -

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

docker:
	docker build -t $(IMAGE):$(VERSION) .
push:
	docker push $(IMAGE):$(VERSION)
