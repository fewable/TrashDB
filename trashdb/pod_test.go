package trashdb_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/taimoorgit/trashdb/trashdb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var examplePod = &v1.Pod{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Pod",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "testing-123",
		Namespace: "namespace-123",
		Labels: map[string]string{
			"app.kubernetes.io/name":       "redis",
			"app.kubernetes.io/instance":   "redis-testing-123",
			"app.kubernetes.io/version":    "7.4",
			"app.kubernetes.io/component":  "cache",
			"app.kubernetes.io/part-of":    "trashdb",
			"app.kubernetes.io/managed-by": "trashdb",
		},
		Annotations: map[string]string{
			"app.trashdb/secret":     "X1-HIBJYLfir7HltuoiunHybtpDT39",
			"app.trashdb/expiration": "2044-01-16T09:23:34-05:00",
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

type MockKubernetesClient struct{}

func (c *MockKubernetesClient) CreatePod(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
	if strings.Contains(namespace, "error") {
		return nil, fmt.Errorf(namespace)
	}
	return examplePod.DeepCopy(), nil
}

func (c *MockKubernetesClient) ListPods(ctx context.Context, namespace string, listOptions metav1.ListOptions) (*v1.PodList, error) {
	if strings.Contains(namespace, "error") {
		return nil, fmt.Errorf(namespace)
	}

	// TODO: make these different from each other, could use options to make them different easily
	pod1 := examplePod.DeepCopy()
	pod2 := examplePod.DeepCopy()
	pod3 := examplePod.DeepCopy()

	mockList := &v1.PodList{
		Items: []v1.Pod{*pod1, *pod2, *pod3},
	}
	return mockList, nil
}

func (c *MockKubernetesClient) DeletePod(ctx context.Context, namespace, podName string) error {
	if strings.Contains(namespace, "error") {
		return fmt.Errorf(namespace)
	}
	return nil
}

func (c *MockKubernetesClient) GetPod(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
	if strings.Contains(namespace, "error") {
		return nil, fmt.Errorf(namespace)
	}
	return examplePod.DeepCopy(), nil
}

func TestGetPod(t *testing.T) { // TODO: these tests need to actually build something using mock client given args
	type testCase struct {
		Name        string
		Namespace   string
		PodName     string
		ExpectedPod *v1.Pod
		ExpectedErr string
		TestClient  MockKubernetesClient
	}
	testCases := []testCase{
		{
			Name:        "Get pod success",
			Namespace:   "namespace-123",
			PodName:     examplePod.Name,
			ExpectedPod: examplePod.DeepCopy(),
			ExpectedErr: "",
		},
		{
			Name:        "Get pod error",
			Namespace:   "some error",
			ExpectedPod: nil,
			ExpectedErr: "some error",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := trashdb.GetPod(context.Background(), &MockKubernetesClient{}, tc.Namespace, tc.PodName)

			// Check for error match
			if tc.ExpectedErr != "" {
				if err == nil || err.Error() != tc.ExpectedErr {
					t.Errorf("Expected error %q, got %v", tc.ExpectedErr, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check pod equality
			if diff := cmp.Diff(tc.ExpectedPod, got); diff != "" {
				t.Errorf("Pod mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestPodIsExpired(t *testing.T) {
	type testCase struct {
		Name     string
		Pod      *v1.Pod
		Expected bool
	}
	testCases := []testCase{
		{
			Name: "Has no expiration date",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			Expected: true,
		},
		{
			Name: "Has invalid expiration date",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.trashdb/expiration": "",
					},
				},
			},
			Expected: true,
		},
		{
			Name: "Not expired",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.trashdb/expiration": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					},
				},
			},
			Expected: false,
		},
		{
			Name: "Expired",
			Pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.trashdb/expiration": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					},
				},
			},
			Expected: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			actual := trashdb.IsExpired(*tc.Pod)
			if actual != tc.Expected {
				t.Errorf("Expected %v, got %v", tc.Expected, actual)
			}
		})
	}
}
