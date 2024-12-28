package trashdb

import (
	"context"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
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
