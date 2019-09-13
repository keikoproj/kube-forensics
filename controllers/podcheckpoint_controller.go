/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	forensicsv1alpha1 "github.com/keikoproj/kube-forensics/api/v1alpha1"
	"github.com/keikoproj/kube-forensics/utils"
)

// PodCheckpointReconciler reconciles a PodCheckpoint object
type PodCheckpointReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// Clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=forensics.keikoproj.io,resources=podcheckpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=forensics.keikoproj.io,resources=podcheckpoints/status,verbs=get;update;patch

// Reconcile is the main entry point for comparing current state of custom resource with desired state and converge to the desired state
func (r *PodCheckpointReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("podcheckpoint", req.NamespacedName)

	// Fetch the PodCheckpoint instance
	var podCheckpoint forensicsv1alpha1.PodCheckpoint
	if err := r.Get(ctx, req.NamespacedName, &podCheckpoint); err != nil {
		log.Error(err, "unable to fetch PodCheckpoint")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}

	pod := &corev1.Pod{}
	err := r.Get(ctx,
		types.NamespacedName{Name: podCheckpoint.Spec.Pod, Namespace: podCheckpoint.Spec.Namespace}, pod)
	if err != nil {
		if apierrs.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			log.Info("PodCheckpoint: Specified pod not found",
				"name", podCheckpoint.Name,
				"namespace", podCheckpoint.Namespace,
				"instance.Status", podCheckpoint.Status,
				"instance.Spec", podCheckpoint.Spec,
			)
			// Set the PodCheckpoint status based on the status of the owned Job
			status := &forensicsv1alpha1.PodCheckpointStatus{
				StartTime:      &metav1.Time{Time: time.Now()},
				CompletionTime: &metav1.Time{Time: time.Now()},
				Conditions: []forensicsv1alpha1.PodCheckpointCondition{
					{
						Type:               forensicsv1alpha1.PodCheckpointFailed,
						Status:             corev1.ConditionTrue,
						LastProbeTime:      metav1.Time{Time: time.Now()},
						LastTransitionTime: metav1.Time{Time: time.Now()},
						Reason:             "NotFound",
						Message: fmt.Sprintf("The specified Pod '%s' was not found in the '%s' namespace.",
							podCheckpoint.Spec.Pod, podCheckpoint.Spec.Namespace),
					},
				},
			}
			status.DeepCopyInto(&podCheckpoint.Status)
			err = r.Update(ctx, &podCheckpoint)
			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	cnt := len(pod.Status.ContainerStatuses)
	s := make([]string, 0, cnt)
	if !(cnt > 0) {
		// No container; requeue and try again
		log.Info("PodCheckpoint: Specified pod does not have any containers")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
		// panic("This is highly unusual; pod doesn't have any containers")
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		tok := strings.SplitAfter(containerStatus.ContainerID, "://")

		if len(tok) != 2 || tok[0] != "docker://" {
			// Return error if invalid containerId is provided.
			err = utils.CommandError{ID: 1, Result: fmt.Sprintf("Unexpected ContainerID (%s)", containerStatus.ContainerID)}
			return ctrl.Result{}, err
		}
		id := tok[1]
		s = append(s, id)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podCheckpoint.Name + "-job",
			Namespace: "forensics-system",
			Labels: map[string]string{
				"env": "security",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"job": podCheckpoint.Name + "-job"}},
				Spec: corev1.PodSpec{
					NodeName:    pod.Spec.NodeName,
					HostNetwork: true,
					Containers: []corev1.Container{
						{
							Name:            "kube-forensics-worker",
							Image:           "keikoproj/kube-forensics-worker:latest",
							ImagePullPolicy: "Always",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "dockersock",
									MountPath: "/var/run/docker.sock",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "DEST_BUCKET",
									Value: podCheckpoint.Spec.Destination,
								},
								{
									Name:  "SUBPATH",
									Value: podCheckpoint.Spec.Subpath,
								}, {
									Name:  "NAMESPACE",
									Value: podCheckpoint.Spec.Namespace,
								},
								{
									Name:  "POD_NAME",
									Value: podCheckpoint.Spec.Pod,
								},
								{
									Name:  "CONTAINER_ID",
									Value: s[0],
								},
							},
						},
					},
					ServiceAccountName: "forensics-worker",
					RestartPolicy:      "Never",
					Volumes: []corev1.Volume{
						{
							Name: "dockersock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/docker.sock",
								},
							},
						},
					},
				},
			},
		},
	}

	// Set this controller to own the job.
	if err := ctrl.SetControllerReference(&podCheckpoint, job, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Check if the job already exists
	found := &batchv1.Job{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && apierrs.IsNotFound(err) {
		log.Info("Creating Job", "namespace", job.Namespace, "name", job.Name)
		err = r.Create(context.TODO(), job)
		if err != nil {
			return ctrl.Result{}, err
		}
		podCheckpoint.Status.StartTime = &metav1.Time{Time: time.Now()}

	} else if err != nil {
		return ctrl.Result{}, err
	}

	podCheckpoint.Status.CompletionTime = found.Status.CompletionTime
	podCheckpoint.Status.Active = found.Status.Active
	podCheckpoint.Status.Succeeded = found.Status.Succeeded
	podCheckpoint.Status.Failed = found.Status.Failed

	err = r.Update(ctx, &podCheckpoint)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("PodCheckpoint",
		"name", podCheckpoint.Name,
		"namespace", podCheckpoint.Namespace,
		"instance.Status", podCheckpoint.Status,
	)

	return ctrl.Result{}, nil
}

// SetupWithManager will configure the controller manager
func (r *PodCheckpointReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&forensicsv1alpha1.PodCheckpoint{}).
		Complete(r)
}
