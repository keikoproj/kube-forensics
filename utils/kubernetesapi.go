package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8s client can be overridden for unit testing
type KubernetesAPI struct {
	Suffix string
	Client kubernetes.Interface
}

func (k KubernetesAPI) GetContainerIdsForPod(podName string, ns string) ([]string, string, error) {
	var options metav1.GetOptions
	pod, err := k.Client.CoreV1().Pods(ns).Get(podName, options)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Pod Name: %s\n", pod.GetName())
	fmt.Printf("Node Name: %s\n", pod.Spec.NodeName)

	// spew.Dump(pod)
	cnt := len(pod.Status.ContainerStatuses)
	s := make([]string, 0, cnt)
	if !(cnt > 0) {
		panic("This is highly unusual; pod doesn't have any containers")
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		// fmt.Printf("\nContainer ID: %s\n\n", containerStatus.ContainerID)
		tok := strings.SplitAfter(containerStatus.ContainerID, "://")

		if len(tok) != 2 || tok[0] != "docker://" {
			err = CommandError{1, fmt.Sprintf("Unexpected ContainerID (%s)", containerStatus.ContainerID)}
			break
		} else {
			id := tok[1]
			s = append(s, id)
		}
	}
	return s, pod.Spec.NodeName, err
}

func (k KubernetesAPI) CreateJobFromJson(jobAsJson string, ns string) error {

	var p batchv1.Job
	err := json.Unmarshal([]byte(jobAsJson), &p)
	if err != nil {
		return err
	}

	//spew.Dump(p)

	job, joberr := k.Client.BatchV1().Jobs(ns).Create(&p)
	// job, joberr := kubernetesConfig().BatchV1().Jobs(ns).Create(&p)
	if joberr != nil {
		return joberr
	}
	fmt.Printf("Created Job %q.\n", job.GetObjectMeta().GetName())
	return nil
}
