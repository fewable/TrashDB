package trashdb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/taimoorgit/trashdb/trashdb"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockKubernetesClient implements KubernetesClient
// MockKubernetesClient with embedded behavior as methods
type MockKubernetesClient struct {
	CreatePodFunc func(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error)
	ListPodsFunc  func(ctx context.Context, namespace string, listOptions metav1.ListOptions) (*v1.PodList, error)
	DeletePodFunc func(ctx context.Context, namespace, podName string) error
	GetPodFunc    func(ctx context.Context, namespace, podName string) (*v1.Pod, error)
}

// Implement the interface methods by delegating to the function fields
func (m *MockKubernetesClient) CreatePod(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
	if m.CreatePodFunc != nil {
		return m.CreatePodFunc(ctx, namespace, pod)
	}
	panic("CreatePod not implemented")
}

func (m *MockKubernetesClient) ListPods(ctx context.Context, namespace string, listOptions metav1.ListOptions) (*v1.PodList, error) {
	if m.ListPodsFunc != nil {
		return m.ListPodsFunc(ctx, namespace, listOptions)
	}
	panic("ListPods not implemented")
}

func (m *MockKubernetesClient) DeletePod(ctx context.Context, namespace, podName string) error {
	if m.DeletePodFunc != nil {
		return m.DeletePodFunc(ctx, namespace, podName)
	}
	panic("DeletePod not implemented")
}

func (m *MockKubernetesClient) GetPod(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
	if m.GetPodFunc != nil {
		return m.GetPodFunc(ctx, namespace, podName)
	}
	panic("GetPod not implemented")
}

// Option pattern for setting mock behaviors
type MockOption func(*MockKubernetesClient)

func WithCreatePodFunc(f func(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error)) MockOption {
	return func(m *MockKubernetesClient) {
		m.CreatePodFunc = f
	}
}

func WithListPodsFunc(f func(ctx context.Context, namespace string, listOptions metav1.ListOptions) (*v1.PodList, error)) MockOption {
	return func(m *MockKubernetesClient) {
		m.ListPodsFunc = f
	}
}

func WithDeletePodFunc(f func(ctx context.Context, namespace, podName string) error) MockOption {
	return func(m *MockKubernetesClient) {
		m.DeletePodFunc = f
	}
}

func WithGetPodFunc(f func(ctx context.Context, namespace, podName string) (*v1.Pod, error)) MockOption {
	return func(m *MockKubernetesClient) {
		m.GetPodFunc = f
	}
}

// Create a new mock client with options
func NewMockKubernetesClient(opts ...MockOption) *MockKubernetesClient {
	mock := &MockKubernetesClient{}
	for _, opt := range opts {
		opt(mock)
	}
	return mock
}

func TestGetPod(t *testing.T) {
	type testCase struct {
		Name        string
		Namespace   string
		PodName     string
		ExpectedPod *v1.Pod
		ExpectedErr string
		MockClient  trashdb.KubernetesClient
	}
	testCases := []testCase{
		{
			Name:      "Get pod success",
			Namespace: "namespace-123",
			PodName:   "pod-123",
			ExpectedPod: trashdb.NewPod(
				trashdb.WithNamespace("namespace-123"),
				trashdb.WithName("pod-123"),
			),
			ExpectedErr: "",
			MockClient: NewMockKubernetesClient(
				WithGetPodFunc(
					func(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
						return trashdb.NewPod(
							trashdb.WithNamespace(namespace),
							trashdb.WithName(podName),
						), nil
					},
				),
			),
		},
		{
			Name:        "Get pod error",
			Namespace:   "some error",
			ExpectedPod: nil,
			ExpectedErr: "some error",
			MockClient: NewMockKubernetesClient(
				WithGetPodFunc(
					func(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
						return nil, fmt.Errorf("some error")
					},
				),
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := trashdb.GetPod(context.Background(), tc.MockClient, tc.Namespace, tc.PodName)

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

func TestCreatePod(t *testing.T) {
	type testCase struct {
		Name        string
		Namespace   string
		PodName     string
		PodSecret   string
		Duration    time.Duration
		ExpectedPod *v1.Pod
		ExpectedErr string
		MockClient  trashdb.KubernetesClient
	}
	testCases := []testCase{
		{
			Name:      "Create pod success",
			Namespace: "namespace-123",
			PodName:   "pod-123",
			PodSecret: "pod-123-secret",
			Duration:  1 * time.Hour,
			ExpectedPod: trashdb.NewPod(
				trashdb.WithNamespace("namespace-123"),
				trashdb.WithName("pod-123"),
				trashdb.WithLabels(map[string]string{
					"app.kubernetes.io/instance": "redis-pod-123",
				}),
				trashdb.WithAnnotations(map[string]string{
					"app.trashdb/expiration": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					"app.trashdb/secret":     "pod-123-secret",
				}),
			),
			ExpectedErr: "",
			MockClient: NewMockKubernetesClient(
				WithCreatePodFunc(func(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
					return pod, nil
				}),
			),
		},
		{
			Name:        "Create pod failure - no namespace",
			Namespace:   "",
			PodName:     "pod-123",
			PodSecret:   "pod-123-secret",
			Duration:    1 * time.Hour,
			ExpectedPod: nil,
			ExpectedErr: "required: namespace, podName, podSecret",
			MockClient: NewMockKubernetesClient(
				WithCreatePodFunc(func(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
					return pod, nil
				}),
			),
		},
		{
			Name:        "Create pod failure - no podName",
			Namespace:   "namespace-123",
			PodName:     "",
			PodSecret:   "pod-123-secret",
			Duration:    1 * time.Hour,
			ExpectedPod: nil,
			ExpectedErr: "required: namespace, podName, podSecret",
			MockClient: NewMockKubernetesClient(
				WithCreatePodFunc(func(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
					return pod, nil
				}),
			),
		},
		{
			Name:        "Create pod failure - no podSecret",
			Namespace:   "namespace-123",
			PodName:     "pod-123",
			PodSecret:   "",
			Duration:    1 * time.Hour,
			ExpectedPod: nil,
			ExpectedErr: "required: namespace, podName, podSecret",
			MockClient: NewMockKubernetesClient(
				WithCreatePodFunc(func(ctx context.Context, namespace string, pod *v1.Pod) (*v1.Pod, error) {
					return pod, nil
				}),
			),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := trashdb.CreatePod(context.Background(), tc.MockClient, tc.Namespace, tc.PodName, tc.PodSecret, tc.Duration)

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
			Name:     "Has no expiration date",
			Pod:      trashdb.NewPod(trashdb.WithAnnotations(map[string]string{})),
			Expected: true,
		},
		{
			Name: "Has invalid expiration date",
			Pod: trashdb.NewPod(trashdb.WithAnnotations(map[string]string{
				"app.trashdb/expiration": "",
			})),
			Expected: true,
		},
		{
			Name: "Not expired",
			Pod: trashdb.NewPod(trashdb.WithAnnotations(map[string]string{
				"app.trashdb/expiration": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			})),
			Expected: false,
		},
		{
			Name: "Expired",
			Pod: trashdb.NewPod(trashdb.WithAnnotations(map[string]string{
				"app.trashdb/expiration": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			})),
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
