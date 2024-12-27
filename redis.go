package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var podsList *v1.PodList

var listPodsMutex sync.Mutex
var createPodMutex sync.Mutex
var deletePodMutex sync.Mutex

var redisPodTemplate = &v1.Pod{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Pod",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name: "redis",
		Labels: map[string]string{
			"label": "trashdb",
		},
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			v1.Container{
				Name:  "redis",
				Image: "redis:7.4",
				Ports: []v1.ContainerPort{
					v1.ContainerPort{
						ContainerPort: 6379,
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("500m"),
						v1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("1"),
						v1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
		},
	},
}

func getPod(ctx context.Context, podName string) (*v1.Pod, error) {
	logger := log.With().Str("podName", podName).Logger()
	logger.Debug().Msg("Getting pod")

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get pod")
		return nil, err
	}

	return pod, nil

}

func createPod(ctx context.Context, podName string, podSecret string) error {
	createPodMutex.Lock()
	defer createPodMutex.Unlock()

	logger := log.With().Str("podName", podName).Logger()
	logger.Info().Msg("Creating pod")

	data := redisPodTemplate.DeepCopy()
	data.SetName(podName)
	data.SetLabels(map[string]string{
		"label":     "trashdb",
		"podSecret": podSecret,
	})
	data.SetAnnotations(map[string]string{
		"expiration": time.Now().Add(90 * time.Second).Format(time.RFC3339),
	})

	_, err := client.CoreV1().Pods(namespace).Create(ctx, data, metav1.CreateOptions{})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create pod")
		return err
	}

	logger.Info().Msg("Pod created")
	return nil
}

func deletePod(ctx context.Context, podName string) error {
	deletePodMutex.Lock()
	defer deletePodMutex.Unlock()

	logger := log.With().Str("podName", podName).Logger()
	logger.Info().Msg("Deleting pod")

	err := client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to delete pod")
		return err
	}

	logger.Info().Msg("Pod deleted")
	return nil
}

func deleteExpiredPods(ctx context.Context) error {
	log.Info().Msg("Checking for expired pods")

	var deleteErrors error

	if podsList != nil {
		for _, pod := range podsList.Items {
			if podExpired(pod) {
				if err := deletePod(ctx, pod.Name); err != nil {
					errors.Join(deleteErrors, err)
				}
			}
		}
	}

	return deleteErrors
}

func podExpiration(pod v1.Pod) (*time.Time, error) {
	logger := log.With().Str("podName", pod.Name).Logger()

	expiration, ok := pod.Annotations["expiration"]
	if ok {
		if expirationTime, err := time.Parse(time.RFC3339, expiration); err != nil {
			logger.Error().Err(err).Msg("Failed to parse expiration time")
			return &expirationTime, nil
		} else {
			return &expirationTime, nil
		}
	} else {
		logger.Debug().Msg("Pod has no expiration")
		return nil, fmt.Errorf("Pod has no expiration")
	}
}

func podExpired(pod v1.Pod) bool {
	logger := log.With().Str("podName", pod.Name).Logger()

	expiration, err := podExpiration(pod)
	if err != nil {
		return true
	}

	if time.Now().After(*expiration) {
		return true
	} else {
		logger.Debug().Msg("Pod has expired")
		return false
	}
}

func listPods(ctx context.Context) error {
	listPodsMutex.Lock()
	defer listPodsMutex.Unlock()

	log.Info().Msg("Listing redis pods")

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "label=trashdb",
		// FieldSelector: "status.phase=Running",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list pods")
		return err
	}

	log.Info().Msgf("Found %d pods", len(pods.Items))
	for _, pod := range pods.Items {
		logger := log.With().Str("podName", pod.Name).Logger()
		logger.Debug().Msg("Found pod")
	}

	podsList = pods
	return nil
}
