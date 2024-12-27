package main

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var listPodsMutex sync.Mutex

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

func createRedisPod(ctx context.Context, podName string) error {
	logger := log.With().Str("podName", podName).Logger()
	logger.Info().Msg("Creating pod")

	data := redisPodTemplate.DeepCopy()
	data.SetName(podName)
	annotations := map[string]string{
		"expire": time.Now().Add(90 * time.Second).Format(time.RFC3339),
	}
	data.ObjectMeta.SetAnnotations(annotations)

	_, err := client.CoreV1().Pods(namespace).Create(ctx, data, metav1.CreateOptions{})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create pod")
		return err
	}

	logger.Info().Msg("Pod created")
	return nil
}

func deleteRedisPod(ctx context.Context, podName string) error {
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

func listRedisPods(ctx context.Context) error {
	listPodsMutex.Lock()
	defer listPodsMutex.Unlock()

	log.Info().Msg("Checking redis pods")

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "label=trashdb",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list pods")
		return err
	}

	log.Info().Msgf("Found %d pods", len(pods.Items))
	for _, pod := range pods.Items {
		logger := log.With().Str("podName", pod.Name).Logger()
		logger.Debug().Msg("Found pod")

		if isPodExpired(pod) {
			deleteRedisPod(ctx, pod.Name)
		}
	}
	return nil
}

func isPodExpired(pod v1.Pod) bool {
	logger := log.With().Str("podName", pod.Name).Logger()

	annotations := pod.GetAnnotations()
	if expire, found := annotations["expire"]; found {
		parsed, err := time.Parse(time.RFC3339, expire)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to parse expire time")
			return true
		}
		return time.Now().After(parsed)
	} else {
		logger.Info().Msg("Missing expire time annotation")
	}
	return true
}
