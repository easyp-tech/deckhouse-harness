package handler

import (
	"context"
	"fmt"
	"testing"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

func TestGetClusterStatus_Empty(t *testing.T) {
	h := NewDiagnosticsHandler(&mockClient{})
	resp, err := h.GetClusterStatus(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Nodes.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Nodes.Total)
	}
	if resp.Nodes.Ready != 0 {
		t.Errorf("expected ready=0, got %d", resp.Nodes.Ready)
	}
	if len(resp.NodeGroups) != 0 {
		t.Errorf("expected 0 node groups, got %d", len(resp.NodeGroups))
	}
	if len(resp.ErroredModules) != 0 {
		t.Errorf("expected 0 errored modules, got %d", len(resp.ErroredModules))
	}
	if len(resp.PendingReleases) != 0 {
		t.Errorf("expected 0 pending releases, got %d", len(resp.PendingReleases))
	}
}

func TestGetClusterStatus_Mixed(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{
				makeNode("node-1", true),
				makeNode("node-2", true),
				makeNode("node-3", false),
			}, nil
		},
		listNodeGroupsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeNodeGroup("workers", 2, 3),
			}, nil
		},
		listModuleConfigsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleConfig("mod-ok", true, ""),
				makeModuleConfig("mod-err", true, "something failed"),
			}, nil
		},
		listDeckhouseReleasesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeRelease("v1.60.0", "Deployed", "1.60.0"),
				makeRelease("v1.61.0", "Pending", "1.61.0"),
			}, nil
		},
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
				{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetClusterStatus(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Nodes.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Nodes.Total)
	}
	if resp.Nodes.Ready != 2 {
		t.Errorf("expected ready=2, got %d", resp.Nodes.Ready)
	}
	if resp.Nodes.NotReady != 1 {
		t.Errorf("expected notReady=1, got %d", resp.Nodes.NotReady)
	}
	if len(resp.NodeGroups) != 1 {
		t.Fatalf("expected 1 node group, got %d", len(resp.NodeGroups))
	}
	if resp.NodeGroups[0].Name != "workers" {
		t.Errorf("expected ng name=workers, got %s", resp.NodeGroups[0].Name)
	}
	if len(resp.ErroredModules) != 1 || resp.ErroredModules[0] != "mod-err" {
		t.Errorf("expected [mod-err], got %v", resp.ErroredModules)
	}
	if len(resp.PendingReleases) != 1 {
		t.Fatalf("expected 1 pending release, got %d", len(resp.PendingReleases))
	}
	if resp.DeckhouseVersion != "1.60.0" {
		t.Errorf("expected version 1.60.0, got %s", resp.DeckhouseVersion)
	}
	if resp.UnhealthyPodsCount != 1 {
		t.Errorf("expected unhealthy=1, got %d", resp.UnhealthyPodsCount)
	}
}

func TestListNodes_NoFilter(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{
				makeNode("node-1", true),
				makeNode("node-2", false),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListNodes(context.Background(), &pb.ListNodesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(resp.Nodes))
	}
}

func TestListNodes_ByNodeGroup(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{
				makeNodeWithGroup("node-1", true, "workers"),
				makeNodeWithGroup("node-2", true, "masters"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	ng := "workers"
	resp, err := h.ListNodes(context.Background(), &pb.ListNodesRequest{NodeGroup: &ng})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].Name != "node-1" {
		t.Errorf("expected node-1, got %s", resp.Nodes[0].Name)
	}
}

func TestListNodes_ByStatus(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{
				makeNode("node-1", true),
				makeNode("node-2", false),
				makeNode("node-3", true),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	status := pb.NodeStatusFilter_NODE_STATUS_FILTER_READY
	resp, err := h.ListNodes(context.Background(), &pb.ListNodesRequest{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Nodes) != 2 {
		t.Errorf("expected 2 ready nodes, got %d", len(resp.Nodes))
	}
}

func TestListNodes_ByRole(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{
				makeNodeWithRole("node-1", true, "master"),
				makeNodeWithRole("node-2", true, "worker"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	role := "worker"
	resp, err := h.ListNodes(context.Background(), &pb.ListNodesRequest{Role: &role})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].Name != "node-2" {
		t.Errorf("expected node-2, got %s", resp.Nodes[0].Name)
	}
}

func TestListNodes_Combined(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			n1 := makeNodeFull("node-1", true, "workers", "worker")
			n2 := makeNodeFull("node-2", false, "workers", "worker")
			n3 := makeNodeFull("node-3", true, "masters", "master")
			return []corev1.Node{n1, n2, n3}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	ng := "workers"
	status := pb.NodeStatusFilter_NODE_STATUS_FILTER_READY
	role := "worker"
	resp, err := h.ListNodes(context.Background(), &pb.ListNodesRequest{
		NodeGroup: &ng,
		Status:    &status,
		Role:      &role,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(resp.Nodes))
	}
	if resp.Nodes[0].Name != "node-1" {
		t.Errorf("expected node-1, got %s", resp.Nodes[0].Name)
	}
}

func TestListNodes_Empty(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{makeNode("node-1", true)}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	ng := "nonexistent"
	resp, err := h.ListNodes(context.Background(), &pb.ListNodesRequest{NodeGroup: &ng})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(resp.Nodes))
	}
}

func TestListNodeGroups_All(t *testing.T) {
	mc := &mockClient{
		listNodeGroupsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeNodeGroup("workers", 3, 3),
				makeNodeGroup("masters", 1, 1),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListNodeGroups(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.NodeGroups) != 2 {
		t.Errorf("expected 2 node groups, got %d", len(resp.NodeGroups))
	}
}

func TestListNodeGroups_WithError(t *testing.T) {
	mc := &mockClient{
		listNodeGroupsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			ng := makeNodeGroup("broken", 0, 1)
			ng.Object["status"].(map[string]interface{})["error"] = "some error"
			return []unstructured.Unstructured{ng}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListNodeGroups(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeGroups[0].Error != "some error" {
		t.Errorf("expected error 'some error', got %q", resp.NodeGroups[0].Error)
	}
}

func TestListNodeGroups_Empty(t *testing.T) {
	mc := &mockClient{
		listNodeGroupsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListNodeGroups(context.Background(), &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.NodeGroups) != 0 {
		t.Errorf("expected 0, got %d", len(resp.NodeGroups))
	}
}

func TestListStaticInstances_NoFilter(t *testing.T) {
	mc := &mockClient{
		listStaticInstancesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeStaticInstance("si-1", "10.0.0.1", "Running", "workers"),
				makeStaticInstance("si-2", "10.0.0.2", "Pending", "workers"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListStaticInstances(context.Background(), &pb.ListStaticInstancesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Instances) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Instances))
	}
}

func TestListStaticInstances_ByPhase(t *testing.T) {
	mc := &mockClient{
		listStaticInstancesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeStaticInstance("si-1", "10.0.0.1", "Running", "workers"),
				makeStaticInstance("si-2", "10.0.0.2", "Pending", "workers"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	phase := pb.StaticInstancePhase_STATIC_INSTANCE_PHASE_RUNNING
	resp, err := h.ListStaticInstances(context.Background(), &pb.ListStaticInstancesRequest{Phase: &phase})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Instances) != 1 {
		t.Errorf("expected 1, got %d", len(resp.Instances))
	}
	if resp.Instances[0].Name != "si-1" {
		t.Errorf("expected si-1, got %s", resp.Instances[0].Name)
	}
}

func TestListStaticInstances_ByNodeGroup(t *testing.T) {
	mc := &mockClient{
		listStaticInstancesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeStaticInstance("si-1", "10.0.0.1", "Running", "workers"),
				makeStaticInstance("si-2", "10.0.0.2", "Running", "masters"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	ng := "workers"
	resp, err := h.ListStaticInstances(context.Background(), &pb.ListStaticInstancesRequest{NodeGroup: &ng})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Instances) != 1 {
		t.Errorf("expected 1, got %d", len(resp.Instances))
	}
}

func TestListStaticInstances_Combined(t *testing.T) {
	mc := &mockClient{
		listStaticInstancesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeStaticInstance("si-1", "10.0.0.1", "Running", "workers"),
				makeStaticInstance("si-2", "10.0.0.2", "Pending", "workers"),
				makeStaticInstance("si-3", "10.0.0.3", "Running", "masters"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	ng := "workers"
	phase := pb.StaticInstancePhase_STATIC_INSTANCE_PHASE_RUNNING
	resp, err := h.ListStaticInstances(context.Background(), &pb.ListStaticInstancesRequest{
		NodeGroup: &ng,
		Phase:     &phase,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Instances) != 1 {
		t.Errorf("expected 1, got %d", len(resp.Instances))
	}
	if resp.Instances[0].Name != "si-1" {
		t.Errorf("expected si-1, got %s", resp.Instances[0].Name)
	}
}

func TestListUnhealthyPods_All(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makePod("pod-1", "ns-1", corev1.PodRunning),
				makePod("pod-2", "ns-1", corev1.PodFailed),
				makePod("pod-3", "ns-2", corev1.PodPending),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListUnhealthyPods(context.Background(), &pb.ListUnhealthyPodsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Pods) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Pods))
	}
}

func TestListUnhealthyPods_Namespace(t *testing.T) {
	called := ""
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, namespace string) ([]corev1.Pod, error) {
			called = namespace
			return []corev1.Pod{
				makePod("pod-1", "ns-1", corev1.PodFailed),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	ns := "ns-1"
	_, err := h.ListUnhealthyPods(context.Background(), &pb.ListUnhealthyPodsRequest{Namespace: &ns})
	if err != nil {
		t.Fatal(err)
	}
	if called != "ns-1" {
		t.Errorf("expected namespace ns-1, got %q", called)
	}
}

func TestListUnhealthyPods_ExcludeCompleted(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makePod("pod-1", "ns-1", corev1.PodFailed),
				makePod("pod-2", "ns-1", corev1.PodSucceeded),
				makePod("pod-3", "ns-1", corev1.PodPending),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	exclude := true
	resp, err := h.ListUnhealthyPods(context.Background(), &pb.ListUnhealthyPodsRequest{ExcludeCompleted: &exclude})
	if err != nil {
		t.Fatal(err)
	}
	// PodFailed and PodPending are unhealthy. PodSucceeded is filtered by the base filter (not in Running/Succeeded).
	// Wait — Succeeded is filtered by base filter already. Let me check: Running/Succeeded are filtered out.
	// So PodFailed and PodPending remain = 2.
	if len(resp.Pods) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Pods))
	}
}

func TestListUnhealthyPods_AllHealthy(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makePod("pod-1", "ns-1", corev1.PodRunning),
				makePod("pod-2", "ns-1", corev1.PodSucceeded),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.ListUnhealthyPods(context.Background(), &pb.ListUnhealthyPodsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Pods) != 0 {
		t.Errorf("expected 0, got %d", len(resp.Pods))
	}
}

func TestGetNode_Found(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			n := makeNodeFull(name, true, "workers", "worker")
			n.Status.Capacity = corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewMilliQuantity(4000, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(8*1024*1024*1024, resource.BinarySI),
			}
			n.Status.Allocatable = n.Status.Capacity
			n.Status.Addresses = []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
			}
			return &n, nil
		},
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return nil, nil
		},
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			si := makeStaticInstance("worker-01", "10.0.0.1", "Running", "workers")
			return &si, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetNode(context.Background(), &pb.GetNodeRequest{Name: "worker-01"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Node.Name != "worker-01" {
		t.Errorf("expected worker-01, got %s", resp.Node.Name)
	}
	if resp.StaticInstancePhase == nil || *resp.StaticInstancePhase != "Running" {
		t.Errorf("expected StaticInstancePhase=Running, got %v", resp.StaticInstancePhase)
	}
}

func TestGetNode_NotFound(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return nil, fmt.Errorf("node %q not found", name)
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetNode(context.Background(), &pb.GetNodeRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetNode_NoStaticInstance(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			n := makeNode(name, true)
			return &n, nil
		},
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return nil, nil
		},
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("not found")
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetNode(context.Background(), &pb.GetNodeRequest{Name: "cloud-node"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StaticInstancePhase != nil {
		t.Errorf("expected nil StaticInstancePhase for cloud node, got %v", resp.StaticInstancePhase)
	}
}

func TestGetNode_WithEvents(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			n := makeNode(name, true)
			return &n, nil
		},
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("not found")
		},
		listNodeEventsFunc: func(_ context.Context, nodeName string) ([]corev1.Event, error) {
			return []corev1.Event{
				{
					Reason:  "NodeReady",
					Message: "node is ready",
					Type:    "Normal",
					Count:   1,
				},
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetNode(context.Background(), &pb.GetNodeRequest{Name: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Events))
	}
	if resp.Events[0].Reason != "NodeReady" {
		t.Errorf("expected reason NodeReady, got %s", resp.Events[0].Reason)
	}
	if resp.Events[0].Count != 1 {
		t.Errorf("expected count 1, got %d", resp.Events[0].Count)
	}
}

func TestGetNodeGroup_Found(t *testing.T) {
	mc := &mockClient{
		getNodeGroupFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			ng := makeNodeGroup(name, 2, 3)
			return &ng, nil
		},
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{
				makeNodeWithGroup("node-1", true, "workers"),
				makeNodeWithGroup("node-2", true, "workers"),
				makeNodeWithGroup("node-3", false, "other"),
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetNodeGroup(context.Background(), &pb.GetNodeGroupRequest{Name: "workers"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "workers" {
		t.Errorf("expected workers, got %s", resp.Name)
	}
	if len(resp.NodeNames) != 2 {
		t.Errorf("expected 2 nodes in group, got %d", len(resp.NodeNames))
	}
}

func TestGetNodeGroup_NotFound(t *testing.T) {
	mc := &mockClient{
		getNodeGroupFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("node group %q not found", name)
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetNodeGroup(context.Background(), &pb.GetNodeGroupRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDeckhouseLogs_Success(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, ns string) ([]corev1.Pod, error) {
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deckhouse-abc",
					Namespace: "d8-system",
					Labels:    map[string]string{"app": "deckhouse"},
				},
			}
			return []corev1.Pod{pod}, nil
		},
		getPodLogsFunc: func(_ context.Context, _, _, _ string, _ *int64, _ *string) (string, error) {
			return "line1\nline2\nline3\n", nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetDeckhouseLogs(context.Background(), &pb.GetDeckhouseLogsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Logs == "" {
		t.Error("expected non-empty logs")
	}
}

func TestGetDeckhouseLogs_NoPod(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetDeckhouseLogs(context.Background(), &pb.GetDeckhouseLogsRequest{})
	if err == nil {
		t.Fatal("expected error for missing deckhouse pod")
	}
}

func TestGetNodeEvents_Happy(t *testing.T) {
	mc := &mockClient{
		listNodeEventsFunc: func(_ context.Context, nodeName string) ([]corev1.Event, error) {
			if nodeName != "worker-01" {
				t.Errorf("expected node worker-01, got %q", nodeName)
			}
			return []corev1.Event{
				{Reason: "NodeReady", Message: "ready", Type: "Normal", Count: 1, LastTimestamp: metav1.Now()},
				{Reason: "NodeNotReady", Message: "not ready", Type: "Warning", Count: 2, LastTimestamp: metav1.Now()},
			}, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetNodeEvents(context.Background(), &pb.GetNodeEventsRequest{Name: "worker-01"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(resp.Events))
	}
	if resp.Events[0].Reason != "NodeReady" {
		t.Errorf("expected first reason=NodeReady, got %q", resp.Events[0].Reason)
	}
	if resp.Events[1].Type != "Warning" {
		t.Errorf("expected second type=Warning, got %q", resp.Events[1].Type)
	}
}

func TestGetNodeEvents_NoEvents(t *testing.T) {
	mc := &mockClient{
		listNodeEventsFunc: func(_ context.Context, _ string) ([]corev1.Event, error) {
			return nil, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetNodeEvents(context.Background(), &pb.GetNodeEventsRequest{Name: "quiet-node"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(resp.Events))
	}
}

func TestGetNodeEvents_NotFound(t *testing.T) {
	mc := &mockClient{
		listNodeEventsFunc: func(_ context.Context, nodeName string) ([]corev1.Event, error) {
			return nil, fmt.Errorf("listing events for node %q: not found", nodeName)
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetNodeEvents(context.Background(), &pb.GetNodeEventsRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetStaticInstance_Happy(t *testing.T) {
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			if name != "worker-01" {
				t.Errorf("expected name worker-01, got %q", name)
			}
			si := makeStaticInstance(name, "10.0.0.1", "Running", "workers")
			// Add credentialsRef and nodeRef for the assertion below.
			si.Object["spec"].(map[string]any)["credentialsRef"] = map[string]any{"name": "worker-01-ssh"}
			si.Object["status"].(map[string]any)["nodeRef"] = map[string]any{"name": "worker-01"}
			return &si, nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	resp, err := h.GetStaticInstance(context.Background(), &pb.GetStaticInstanceRequest{Name: "worker-01"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "worker-01" {
		t.Errorf("expected name worker-01, got %q", resp.Name)
	}
	if resp.Address != "10.0.0.1" {
		t.Errorf("expected address 10.0.0.1, got %q", resp.Address)
	}
	if resp.Phase != "Running" {
		t.Errorf("expected phase Running, got %q", resp.Phase)
	}
	if resp.CredentialsRef != "worker-01-ssh" {
		t.Errorf("expected credentialsRef worker-01-ssh, got %q", resp.CredentialsRef)
	}
	if resp.NodeRef != "worker-01" {
		t.Errorf("expected nodeRef worker-01, got %q", resp.NodeRef)
	}
	if resp.Labels["node.deckhouse.io/group"] != "workers" {
		t.Errorf("expected label workers, got %q", resp.Labels["node.deckhouse.io/group"])
	}
}

func TestGetStaticInstance_NotFound(t *testing.T) {
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("static instance %q not found", name)
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetStaticInstance(context.Background(), &pb.GetStaticInstanceRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPodLogs_Happy(t *testing.T) {
	var (
		capturedNS, capturedPod, capturedContainer string
		capturedTail                               *int64
		capturedSince                              *string
	)
	mc := &mockClient{
		getPodLogsFunc: func(_ context.Context, ns, pod, container string, tail *int64, since *string) (string, error) {
			capturedNS = ns
			capturedPod = pod
			capturedContainer = container
			capturedTail = tail
			capturedSince = since
			return "line1\nline2\n", nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	container := "deckhouse"
	tail := int64(50)
	since := "30m"
	resp, err := h.GetPodLogs(context.Background(), &pb.GetPodLogsRequest{
		Namespace: "d8-system",
		Pod:       "deckhouse-abc",
		Container: &container,
		Tail:      ptr(int32(50)),
		Since:     &since,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Logs != "line1\nline2\n" {
		t.Errorf("expected logs, got %q", resp.Logs)
	}
	if capturedNS != "d8-system" {
		t.Errorf("expected ns d8-system, got %q", capturedNS)
	}
	if capturedPod != "deckhouse-abc" {
		t.Errorf("expected pod deckhouse-abc, got %q", capturedPod)
	}
	if capturedContainer != "deckhouse" {
		t.Errorf("expected container deckhouse, got %q", capturedContainer)
	}
	if capturedTail == nil || *capturedTail != tail {
		t.Errorf("expected tail %d, got %v", tail, capturedTail)
	}
	if capturedSince == nil || *capturedSince != since {
		t.Errorf("expected since %q, got %v", since, capturedSince)
	}
}

func TestGetPodLogs_NotFound(t *testing.T) {
	mc := &mockClient{
		getPodLogsFunc: func(_ context.Context, _, _, _ string, _ *int64, _ *string) (string, error) {
			return "", fmt.Errorf("opening log stream: pod not found")
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetPodLogs(context.Background(), &pb.GetPodLogsRequest{
		Namespace: "ns",
		Pod:       "missing",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ptr is a small helper for getting a pointer to a value (used for int32 → *int32).
func ptr[T any](v T) *T {
	return &v
}

func TestGetDeckhouseLogs_Grep(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "deckhouse-xyz",
					Labels: map[string]string{"app": "deckhouse"},
				},
			}
			return []corev1.Pod{pod}, nil
		},
		getPodLogsFunc: func(_ context.Context, _, _, _ string, _ *int64, _ *string) (string, error) {
			return "INFO: starting\nERROR: something failed\nINFO: done\n", nil
		},
	}

	h := NewDiagnosticsHandler(mc)
	grep := "ERROR"
	resp, err := h.GetDeckhouseLogs(context.Background(), &pb.GetDeckhouseLogsRequest{Grep: &grep})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Logs != "ERROR: something failed\n" {
		t.Errorf("expected filtered log, got %q", resp.Logs)
	}
}

// Test helpers

func makeNode(name string, ready bool) corev1.Node {
	condStatus := corev1.ConditionFalse
	if ready {
		condStatus = corev1.ConditionTrue
	}
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{}, CreationTimestamp: metav1.Now()},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: condStatus},
			},
		},
	}
}

func makeNodeWithGroup(name string, ready bool, group string) corev1.Node {
	n := makeNode(name, ready)
	n.Labels["node.deckhouse.io/group"] = group
	return n
}

func makeNodeWithRole(name string, ready bool, role string) corev1.Node {
	n := makeNode(name, ready)
	n.Labels["node-role.kubernetes.io/"+role] = ""
	return n
}

func makeNodeFull(name string, ready bool, group, role string) corev1.Node {
	n := makeNode(name, ready)
	n.Labels["node.deckhouse.io/group"] = group
	n.Labels["node-role.kubernetes.io/"+role] = ""
	return n
}

func makeNodeGroup(name string, ready, total int64) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "deckhouse.io/v1",
			"kind":       "NodeGroup",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"nodeType": "Static",
			},
			"status": map[string]interface{}{
				"ready": ready,
				"nodes": total,
			},
		},
	}
}

func makeStaticInstance(name, address, phase, nodeGroup string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "deckhouse.io/v1alpha2",
			"kind":       "StaticInstance",
			"metadata": map[string]interface{}{
				"name": name,
				"labels": map[string]interface{}{
					"node.deckhouse.io/group": nodeGroup,
				},
			},
			"spec": map[string]interface{}{
				"address": address,
			},
			"status": map[string]interface{}{
				"currentStatus": map[string]interface{}{
					"phase": phase,
				},
			},
		},
	}
}

func makeModuleConfig(name string, enabled bool, statusMsg string) unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": "deckhouse.io/v1alpha1",
		"kind":       "ModuleConfig",
		"metadata": map[string]interface{}{
			"name": name,
		},
		"spec": map[string]interface{}{
			"enabled": enabled,
		},
		"status": map[string]interface{}{},
	}
	if statusMsg != "" {
		obj["status"].(map[string]interface{})["message"] = statusMsg
	}
	return unstructured.Unstructured{Object: obj}
}

func makeRelease(name, phase, version string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "deckhouse.io/v1alpha1",
			"kind":       "DeckhouseRelease",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"version": version,
			},
			"status": map[string]interface{}{
				"phase": phase,
			},
		},
	}
}

func makePod(name, namespace string, phase corev1.PodPhase) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, CreationTimestamp: metav1.Now()},
		Status:     corev1.PodStatus{Phase: phase},
	}
}
