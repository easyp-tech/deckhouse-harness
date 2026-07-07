package handler

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

// GetClusterStatus must degrade gracefully (empty node groups, no error) when the
// nodegroups CRD is absent — e.g. the node-manager module is disabled.
func TestGetClusterStatus_NodeGroupsCRDAbsent(t *testing.T) {
	mc := &mockClient{
		listNodeGroupsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, errors.New("nodegroups.deckhouse.io: the server could not find the requested resource")
		},
	}

	h := NewDiagnosticsHandler(mc)

	resp, err := h.GetClusterStatus(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if len(resp.GetNodeGroups()) != 0 {
		t.Errorf("expected empty node groups, got %d", len(resp.GetNodeGroups()))
	}
}

// ListModuleConfigs must render the integer spec.version as a string ("2"),
// not drop it because the raw value is not a string.
func TestListModuleConfigs_VersionFromInt(t *testing.T) {
	mc := &mockClient{
		listModuleConfigsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{{
				Object: map[string]any{
					"apiVersion": "deckhouse.io/v1alpha1",
					"kind":       "ModuleConfig",
					"metadata":   map[string]any{"name": "cert-manager"},
					"spec":       map[string]any{"enabled": true, "version": int64(2)},
				},
			}}, nil
		},
	}

	h := NewModulesHandler(mc)

	resp, err := h.ListModuleConfigs(context.Background(), &pb.ListModuleConfigsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetModules()) != 1 {
		t.Fatalf("expected 1 module, got %d", len(resp.GetModules()))
	}
	if got := resp.GetModules()[0].GetVersion(); got != "2" {
		t.Errorf("expected version %q, got %q", "2", got)
	}
}

// A CrashLoopBackOff pod has phase=Running (a container is waiting), so
// ListUnhealthyPods must inspect container state — not just pod phase — or it
// misses the most common unhealthy case.
func TestListUnhealthyPods_CrashLoopBackOff(t *testing.T) {
	crashing := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "crasher", Namespace: "e2e", CreationTimestamp: metav1.Now()},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning, // <-- Running, yet the container is crash-looping
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:         "app",
				RestartCount: 5,
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
				},
			}},
		},
	}
	healthy := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "e2e", CreationTimestamp: metav1.Now()},
		Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "nginx", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}},
		},
	}

	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{crashing, healthy}, nil
		},
	}
	h := NewDiagnosticsHandler(mc)

	resp, err := h.ListUnhealthyPods(context.Background(), &pb.ListUnhealthyPodsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetPods()) != 1 {
		t.Fatalf("expected only the crashing pod, got %d", len(resp.GetPods()))
	}
	p := resp.GetPods()[0]
	if p.GetName() != "crasher" {
		t.Errorf("expected crasher, got %q", p.GetName())
	}
	if p.GetStatus() != "CrashLoopBackOff" {
		t.Errorf("expected status CrashLoopBackOff, got %q", p.GetStatus())
	}
	if p.GetRestartCount() != 5 {
		t.Errorf("expected restartCount 5, got %d", p.GetRestartCount())
	}
}
