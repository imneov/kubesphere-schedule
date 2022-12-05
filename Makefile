# Image URL to use all building/pushing image targets
COMMIT := $(shell git rev-parse --short HEAD)
VERSION := dev-$(shell git describe --tags $(shell git rev-list --tags --max-count=1))

DOCKER_REPO ?= kubespheredev
CONTROLLER_IMG ?= ${DOCKER_REPO}/scheduling-controller:$(VERSION)-$(COMMIT)
APISERVER_IMG ?= ${DOCKER_REPO}/scheduling-apiserver:$(VERSION)-$(COMMIT)
TOOLS_IMG ?= ${DOCKER_REPO}/sechduling-tools:$(VERSION)-$(COMMIT)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
CONTAINER_CLI?=docker

GV="sechduling.kubesphere.io:v1alpha1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: test lint

# Run tests
test: fmt vet generate manifests
	go test ./... -coverprofile coverage.out

# Build manager binary
manager: generate fmt vet
	go build -a -o bin/controller-manager cmd/controller/main.go

tools-jwt: fmt vet
	go build -a -o bin/jwt cmd/tools/jwt/jwt_cmd.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run cmd/controller/main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Install golang-lint via https://golangci-lint.run/usage/install/#local-installation
lint:
	golangci-lint run ./...

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

clientset:
	./hack/generate_client.sh ${GV}

openapi: openapi-gen
	openapi-gen -O openapi_generated -i ./api/v1alpha1 -p kubesphere.io/api/sechduling/v1alpha1 -h ./hack/boilerplate.go.txt --report-filename ./api/violation_exceptions.list

generate-listers:
	lister-gen -v=2 --output-base=. --input-dirs kubesphere.io/sechduling/pkg/api/sechduling/v1alpha1  \
 		--output-package pkg/client/listers -h hack/boilerplate.go.txt

# Build the docker image of controller-manager
docker-build-controller:
	${CONTAINER_CLI} build . -f config/dockerfiles/controller-manager/Dockerfile --build-arg GOPROXY=${GOPROXY} -t ${CONTROLLER_IMG}
build-controller:
	buildctl build --frontend dockerfile.v0 --local dockerfile=config/dockerfiles/controller-manager/

# Push the docker image of controller-manager
docker-push-controller:
	${CONTAINER_CLI} push ${CONTROLLER_IMG}

# Build and push the docker image
docker-build-push-controller: docker-build-controller docker-push-controller

run-apiserver:
	go run cmd/apiserver/apiserver.go
# Build the docker image of apiserver
docker-build-apiserver:
	${CONTAINER_CLI} build . -f config/dockerfiles/apiserver/Dockerfile --build-arg GOPROXY=${GOPROXY} -t ${APISERVER_IMG}

# Push the docker image of controller-manager
docker-push-apiserver:
	${CONTAINER_CLI} push ${APISERVER_IMG}

# Build and push the docker image
docker-build-push-apiserver: docker-build-apiserver docker-push-apiserver

# Build the docker image of apiserver
docker-build-tools:
	${CONTAINER_CLI} build . -f config/dockerfiles/tools/Dockerfile --build-arg GOPROXY=${GOPROXY} -t ${TOOLS_IMG}

# Push the docker image of controller-manager
docker-push-tools:
	${CONTAINER_CLI} push ${TOOLS_IMG}

# Build and push the docker image
docker-build-push-tools: docker-build-tools docker-push-tools

docker-build-push: docker-build-push-apiserver docker-build-push-controller

atest:
	atest run -p 'test/api/*.yaml'

build-tpl:
	mkdir -p bin
	go build -o bin/tpl cmd/tools/tpl/*.go
copy-tpl: build-tpl
	cp bin/tpl /usr/local/bin/

swagger-ui:
	git clone https://github.com/swagger-api/swagger-ui -b v2.2.10 --depth 1 bin/swagger-ui

mock-gen:
	mockgen -source=cmd/tools/jwt/app/configmap_updater.go -destination ./cmd/tools/jwt/app/mock_app/configmap_updater.go
	mockgen -source=cmd/tools/jwt/app/kubernetes.go -destination ./cmd/tools/jwt/app/mock_app/kubernetes.go



# Install CRDs into a cluster
install: manifests kustomize
	kustomize build config/crd | kubectl apply -f -

install-crd:
	kubectl apply -f config/crd/bases

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && kustomize edit set image controller=${CONTROLLER_IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
#manifests: controller-gen
#	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=ks-schedule webhook paths="./api/..." output:crd:artifacts:config=config/crd/bases output:rbac:none
#	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=ks-schedule webhook paths="./pkg/..." output:crd:none
.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases




## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.5
CONTROLLER_TOOLS_VERSION ?= v0.9.2

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION) ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif


# find or download openapi-gen
# download openapi-gen if necessary
openapi-gen:
ifeq (, $(shell which openapi-gen))
	@{ \
	set -e ;\
	OPENAPI_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$OPENAPI_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install k8s.io/kube-openapi/cmd/openapi-gen ;\
	rm -rf $$OPENAPI_GEN_TMP_DIR ;\
	}
OPENAPI_GEN=$(GOBIN)/openapi-gen
else
OPENAPI_GEN=$(shell which openapi-gen)
endif


# find or download kustomize
# download kustomize if necessary
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	KUSTOMIZE_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_TMP_DIR ;\
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); } ;\
	mv kustomize $(GOBIN)/ ;\
	rm -rf $$KUSTOMIZE_TMP_DIR ;\
	KUSTOMIZE=$(GOBIN)/kustomize  ;\
	}
else
	@{ \
	KUSTOMIZE=$(shell which kustomize) ;\
	}
endif
