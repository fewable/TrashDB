package trashdb

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var client *kubernetes.Clientset

func SetClient(c *kubernetes.Clientset) {
	log.Warn().Msg("Setting up a real Kubernetes Client!!!")
	client = c
}

var podsCache *v1.PodList

// TODO: a diff output would be nice here, but can't do it easily because of metadata changing constantly...
func UpdatePodsCache(pods *v1.PodList) {
	if podsCache == nil {
		log.Info().Msg("Initializing pod cache")
		podsCache = pods
		return
	}
	log.Info().Msg("Updating pod cache")
	podsCache = pods
}

var redisPodTemplate = &v1.Pod{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Pod",
	},
	ObjectMeta: metav1.ObjectMeta{
		Labels: map[string]string{
			"app.kubernetes.io/name":       "redis",
			"app.kubernetes.io/version":    "7.4",
			"app.kubernetes.io/component":  "cache",
			"app.kubernetes.io/part-of":    "trashdb",
			"app.kubernetes.io/managed-by": "trashdb",
		},
		Annotations: map[string]string{},
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

func CreatePod(ctx context.Context, client KubernetesClient, namespace, podName, podSecret string, duration time.Duration) (*v1.Pod, error) {
	if client == nil {
		client = &RealKubernetesClient{}
	}

	if namespace == "" || podName == "" || podSecret == "" {
		return nil, fmt.Errorf("required: namespace, podName, podSecret")
	}

	data := redisPodTemplate.DeepCopy()
	data.SetNamespace(namespace)
	data.SetName(podName)
	data.Labels["app.kubernetes.io/instance"] = "redis-" + podName
	data.Annotations["app.trashdb/expiration"] = time.Now().Add(duration).Format(time.RFC3339)
	data.Annotations["app.trashdb/secret"] = podSecret

	return client.CreatePod(ctx, namespace, data)
}

func ListPods(ctx context.Context, client KubernetesClient, namespace string) (*v1.PodList, error) {
	if client == nil {
		client = &RealKubernetesClient{}
	}

	newPods, err := client.ListPods(ctx, namespace, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=trashdb",
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to list pods")
		return nil, err
	}

	log.Info().Msgf("Found %d pods", len(newPods.Items))

	UpdatePodsCache(newPods)
	return newPods, err
}

func DeletePod(ctx context.Context, client KubernetesClient, namespace, podName string) error {
	if client == nil {
		client = &RealKubernetesClient{}
	}

	return client.DeletePod(ctx, namespace, podName)
}

// used for when the user wants to delete a pod that they have the secret for
func DeletePodWithSecret(ctx context.Context, client KubernetesClient, namespace, podName, podSecret string) error {
	if client == nil {
		client = &RealKubernetesClient{}
	}

	pod, err := client.GetPod(ctx, namespace, podName)
	if err != nil {
		return err
	}

	if actualSecret, ok := pod.Annotations["app.trashdb/secret"]; ok && podSecret != actualSecret {
		return fmt.Errorf("Wrong secret")
	}

	return client.DeletePod(ctx, namespace, podName)
}

func GetPod(ctx context.Context, client KubernetesClient, namespace, podName string) (*v1.Pod, error) {
	if client == nil {
		client = &RealKubernetesClient{}
	}

	return client.GetPod(ctx, namespace, podName)
}

func PodExpiration(pod v1.Pod) (*time.Time, error) {
	if expiration, ok := pod.Annotations["app.trashdb/expiration"]; ok {
		if expirationTime, err := time.Parse(time.RFC3339, expiration); err != nil {
			return nil, fmt.Errorf("failed to parse expiration timestamp")
		} else {
			return &expirationTime, nil
		}
	}
	return nil, fmt.Errorf("pod has no expiration")
}

func IsExpired(pod v1.Pod) bool {
	expiration, err := PodExpiration(pod)
	if err != nil {
		return true
	}
	return time.Now().After(*expiration)
}

// TODO: should this be a method of KubernetesClient?
func DeleteExpiredPods(ctx context.Context, namespace string) {
	if podsCache == nil {
		return
	}

	var success, failures int
	for _, pod := range podsCache.Items {
		if IsExpired(pod) {
			if err := DeletePod(ctx, nil, namespace, pod.Name); err != nil {
				failures++
			} else {
				success++
			}
		}
	}
	total := success + failures
	if total > 0 {
		log.Info().Msgf("Attempted to clean up %d pods, %d success, %d failures", total, success, failures)
	}
}

type KubernetesClient interface {
	CreatePod(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error)
	ListPods(ctx context.Context, namespace string, listOptions metav1.ListOptions) (*v1.PodList, error)
	DeletePod(ctx context.Context, namespace, podName string) error
	GetPod(ctx context.Context, namespace, podName string) (*v1.Pod, error)
}

type RealKubernetesClient struct{}

func (c *RealKubernetesClient) CreatePod(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
	return client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
}

func (c *RealKubernetesClient) ListPods(ctx context.Context, namespace string, listOptions metav1.ListOptions) (*v1.PodList, error) {
	return client.CoreV1().Pods(namespace).List(ctx, listOptions)
}

func (c *RealKubernetesClient) DeletePod(ctx context.Context, namespace, podName string) error {
	return client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
}

func (c *RealKubernetesClient) GetPod(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
	return client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
}
