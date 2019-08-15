# Development reference

This document will walk you through setting up a basic testing environment and running unit tests.

## Deploy to an existing cluster

- Make sure you have KUBECONFIG and/or `kubectl config current-context` is properly set for access to your cluster

- Run `make deploy`

### Example

```bash
$ make deploy
/Users/tekenstam/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
kubectl apply -f config/crd/bases
customresourcedefinition.apiextensions.k8s.io/podcheckpoints.forensics.orkaproj.io configured
kustomize build config/default | kubectl apply -f -
namespace/forensics-system unchanged
customresourcedefinition.apiextensions.k8s.io/podcheckpoints.forensics.orkaproj.io configured
role.rbac.authorization.k8s.io/forensics-leader-election-role unchanged
clusterrole.rbac.authorization.k8s.io/forensics-manager-role configured
clusterrole.rbac.authorization.k8s.io/forensics-proxy-role unchanged
rolebinding.rbac.authorization.k8s.io/forensics-leader-election-rolebinding unchanged
clusterrolebinding.rbac.authorization.k8s.io/forensics-manager-rolebinding unchanged
clusterrolebinding.rbac.authorization.k8s.io/forensics-proxy-rolebinding unchanged
service/forensics-controller-manager-metrics-service unchanged
deployment.apps/forensics-controller-manager configured
```

## Running locally

Using the `Makefile` you can use `make run` to run kube-forensics locally on your machine, and it will try to reconcile PodCheckpoint resources in the cluster.
Make sure the context you are in is correct and that you scaled down the actual instance-manager controller pod.

### Example

```bash
# Disable deployed kube-forensics controller, if applicable
$ kubectl scale deployment -n forensics-system forensics-controller-manager --replicas 0
deployment.extensions/forensics-controller-manager scaled

$ make run
/Users/tekenstam/go/bin/controller-gen object:headerFile=./hack/boilerplate.go.txt paths=./api/...
go fmt ./...
go vet ./...
go run ./main.go
2019-07-17T17:15:09.752-0700	INFO	controller-runtime.controller	Starting EventSource	{"controller": "podcheckpoint", "source": "kind source: /, Kind="}
2019-07-17T17:15:09.752-0700	INFO	setup	starting manager
2019-07-17T17:15:09.853-0700	INFO	controller-runtime.controller	Starting Controller	{"controller": "podcheckpoint"}
2019-07-17T17:15:09.953-0700	INFO	controller-runtime.controller	Starting workers	{"controller": "podcheckpoint", "worker count": 1}
```

*NOTE:* Remember to scale the forensics-controller-manager back up once you are done running locally.

## Running unit tests

Using the `Makefile` you can run basic unit tests.

### Example

```bash
$ make test
/Users/tekenstam/go/bin/controller-gen object:headerFile=./hack/boilerplate.go.txt paths=./api/...
go fmt ./...
go vet ./...
/Users/tekenstam/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
go test ./utils/... ./api/... ./controllers/... -coverprofile cover.out
?   	github.com/orkaproj/kube-forensics/utils	[no test files]
ok  	github.com/orkaproj/kube-forensics/api/v1alpha1	14.008s	coverage: 1.6% of statements
ok  	github.com/orkaproj/kube-forensics/controllers	13.465s	coverage: 0.0% of statements
```
