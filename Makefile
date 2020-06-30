PKGS := ./utils/... ./api/... ./controllers/...

# Image URL to use all building/pushing image targets
TAG ?= latest
IMG ?= keikoproj/kube-forensics-controller:${TAG}
WORKERIMG ?= keikoproj/kube-forensics-worker:${TAG}

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

export GO111MODULE = on

all: manager

# Run tests
test: generate fmt vet manifests
	go test ${PKGS} ${TESTARGS}

cover: TESTARGS=-coverprofile=cover.out
cover: test
	go tool cover -func=cover.out -o cover.txt
	go tool cover -html=cover.out -o cover.html
	@cat cover.txt
	@echo "Run 'open cover.html' to view coverage report."

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crd/bases

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crd/bases
	echo "# NOTE: This is a generated file; _DO NOT EDIT_" > config/samples/deploy.yaml
	kustomize build config/default >> config/samples/deploy.yaml
	kubectl apply -f config/samples/deploy.yaml

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Build the docker image
docker-build:
	docker build . -t ${IMG}
	docker build . -t ${WORKERIMG} --file Dockerfile.worker
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}
	docker push ${WORKERIMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.2
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
