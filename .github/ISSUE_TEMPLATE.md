**Is this a BUG REPORT or FEATURE REQUEST?**:

**What happened**:

**What you expected to happen**:

**How to reproduce it (as minimally and precisely as possible)**:

**Anything else we need to know?**:

**Environment**:
- kube-forensics version
- Kubernetes version :
```
$ kubectl version -o yaml
```

**Other debugging information (if applicable)**:
- PodCheckpoint status:
```
$ kubectl describe -n forensics-system podcheckpoint <checkpoint-name>
```
- controller logs:
```
$ kubectl logs -n forensics-system <forensics-controller-manager pod>
```