# kube-forensics

[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)][GithubMaintainedUrl]
[![PR](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)][GithubPrsUrl]
[![slack](https://img.shields.io/badge/slack-join%20the%20conversation-ff69b4.svg)][SlackUrl]

![version](https://img.shields.io/badge/version-0.1.0-blue.svg?cacheSeconds=2592000)
[![Build Status][BuildStatusImg]][BuildMasterUrl]
[![codecov][CodecovImg]][CodecovUrl]
[![Go Report Card][GoReportImg]][GoReportUrl]

> Create checkpoint snapshots of the state of running pods for later off-line analysis.

kube-forensics allows a cluster administrator to dump the current state of a running pod and all its containers so that security professionals can perform off-line forensic analysis.

<!-- ![kube-forensics](docs/kube-forensics.png) -->

In the event of a security breach, members of the Security Team need to examine the state of the Pod and perform a detailed forensics analysis to determine the mode of attack. However, the business would like to terminate the Pod and get back to normal processing as quickly as possible. kube-forensics was developed to allow a cluster administrator to dump the state of a running Pod for offline analysis.

The forensics-controller-manager manages a PodCheckpoint custom resource definition (CRD). The PodCheckpoint resource runs a Kubernetes Job on the same node as the target pod and performs the equivalent of the following operations on the indicated pod/containers:

``` bash
docker inspect
docker diff
docker export
```

In addition, it collects some meta-data about the target pod. The output is uploaded to the destination S3 bucket.

## Installation

You must have cluster administrator access to deploy kube-forensics to a running cluster.

1. Insure your `KUBECONFIG` and current context correctly points to the desired cluster.
1. Checkout kube-forensics repository
1. Change directory into the root of the repository
1. Run `make deploy`

For example:

```sh
$ cd kube-forensics
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
deployment.apps/forensics-controller-manager unchanged
```

## Usage example

Once the kube-forensics controller is installed, a `PodCheckpoint` spec can be submitted for processing.

### Sample spec

Save the following `yaml` file to `example.yaml` and modify the `destination`, `pod` and `namespace` to valid values for your cluster.

``` yaml
apiVersion: forensics.orkaproj.io/v1alpha1
kind: PodCheckpoint
metadata:
  name: podcheckpoint-sample
  namespace: forensics-system
spec:
  destination: s3://my-bucket-123456789000-us-west-2
  subpath: forensics
  pod: bad-pod-1234567890-dead1
  namespace: default
```

### Submit & Verify

``` sh
$ kubectl apply -f ./config/samples/forensics_v1alpha1_podcheckpoint.yaml
podcheckpoint.forensics.orkaproj.io/podcheckpoint-sample created

$ kubectl get -n forensics-system PodCheckpoint
NAME                   AGE
podcheckpoint-sample   33s
```

Check the state of the PodCheckpoint.

```sh
$ kubectl describe PodCheckpoint -n forensics-system podcheckpoint-sample
Name:         podcheckpoint-sample
Namespace:    forensics-system
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration:
                {"apiVersion":"forensics.orkaproj.io/v1alpha1","kind":"PodCheckpoint","metadata":{"annotations":{},"name":"podcheckpoint-sample","namespac...
API Version:  forensics.orkaproj.io/v1alpha1
Kind:         PodCheckpoint
Metadata:
  Creation Timestamp:  2019-08-14T23:19:13Z
  Generation:          2
  Resource Version:    595318
  Self Link:           /apis/forensics.orkaproj.io/v1alpha1/namespaces/forensics-system/podcheckpoints/podcheckpoint-sample
  UID:                 edbe3bd6-bee9-11e9-a5c6-0afa5b77e74c
Spec:
  Destination:  s3://my-bucket-123456789000-us-west-2
  Namespace:    default
  Pod:          bad-pod-1234567890-dead1
  Subpath:      forensics
Status:
  Completion Time:  2019-08-14T23:19:13Z
  Conditions:
    Last Probe Time:       2019-08-14T23:19:13Z
    Last Transition Time:  2019-08-14T23:19:13Z
    Message:               The specified Pod 'bad-pod-1234567890-dead1' was not found in the 'default' namespace.
    Reason:                NotFound
    Status:                True
    Type:                  Failed
  Start Time:              2019-08-14T23:19:13Z
Events:                    <none>
```

In the above output you can see the PodCheckpoint failed due to the Pod name not being found in the system.

### Bucket Configuration

The S3 bucket indicated in the `destination` spec must allow the worker pod created by kube-forensics to put objects into the bucket. For example, you may use the `nodes` role of the cluster to provide the needed access.

``` yaml
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::<AWS_ACCOUNT>:role/nodes.<CLUSTER_NAME>.cluster.k8s.local"
            },
            "Action": "s3:PutObject",
            "Resource": "arn:aws:s3:::kops-state-store-<AWS_ACCOUNT>-us-west-2/*"
        }
    ]
}
```

## Release History

* 0.1.0
  * Release alpha version of kube-forensics

## ❤ Contributing ❤

Please see [CONTRIBUTING.md](.github/CONTRIBUTING.md).

## Developer Guide

Please see [DEVELOPER.md](.github/DEVELOPER.md).

<!-- Markdown links -->

[BuildStatusImg]: https://travis-ci.org/orkaproj/kube-forensics.svg?branch=master
[BuildMasterUrl]: https://travis-ci.org/orkaproj/kube-forensics

[GithubMaintainedUrl]: https://github.com/orkaproj/kube-forensics/graphs/commit-activity
[GithubPrsUrl]: https://github.com/orkaproj/kube-forensics/pulls
[SlackUrl]: https://orkaproj.slack.com/app_redirect?channel=kube-forensics

[CodecovImg]: https://codecov.io/gh/orkaproj/kube-forensics/branch/master/graph/badge.svg
[CodecovUrl]: https://codecov.io/gh/orkaproj/kube-forensics

[GoReportImg]: https://goreportcard.com/badge/github.com/orkaproj/kube-forensics
[GoReportUrl]: https://goreportcard.com/report/github.com/orkaproj/kube-forensics
