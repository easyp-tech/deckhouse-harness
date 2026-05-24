package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

func TestCreateSSHCredentials_OK(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.CreateSSHCredentials(context.Background(), &pb.CreateSSHCredentialsRequest{
		Name:       "test-creds",
		User:       "ubuntu",
		PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "test-creds" {
		t.Errorf("expected test-creds, got %s", resp.Name)
	}

	// Verify base64 encoding.
	spec := captured.Object["spec"].(map[string]interface{})
	encoded := spec["privateSSHKey"].(string)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if string(decoded) != "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----" {
		t.Errorf("round-trip failed: got %q", string(decoded))
	}
}

func TestCreateSSHCredentials_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, errors.NewAlreadyExists(schema.GroupResource{Group: "deckhouse.io", Resource: "sshcredentials"}, "test-creds")
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateSSHCredentials(context.Background(), &pb.CreateSSHCredentialsRequest{
		Name:       "test-creds",
		User:       "ubuntu",
		PrivateKey: "key",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateSSHCredentials_Defaults(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateSSHCredentials(context.Background(), &pb.CreateSSHCredentialsRequest{
		Name:       "test-creds",
		User:       "ubuntu",
		PrivateKey: "key",
	})
	if err != nil {
		t.Fatal(err)
	}

	spec := captured.Object["spec"].(map[string]interface{})
	port := spec["sshPort"]
	if port != int64(22) {
		t.Errorf("expected default port 22, got %v", port)
	}
}

func TestCreateStaticInstance_OK(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createStaticInstanceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.CreateStaticInstance(context.Background(), &pb.CreateStaticInstanceRequest{
		Name:           "node-1",
		Address:        "10.0.0.1",
		CredentialsRef: "creds-1",
		Labels: map[string]string{
			"node.deckhouse.io/group": "workers",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "node-1" {
		t.Errorf("expected node-1, got %s", resp.Name)
	}

	// Verify labels propagated.
	labels := captured.GetLabels()
	if labels["node.deckhouse.io/group"] != "workers" {
		t.Errorf("expected label workers, got %q", labels["node.deckhouse.io/group"])
	}
}

func TestCreateStaticInstance_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		createStaticInstanceFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, errors.NewAlreadyExists(schema.GroupResource{Group: "deckhouse.io", Resource: "staticinstances"}, "node-1")
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateStaticInstance(context.Background(), &pb.CreateStaticInstanceRequest{
		Name:           "node-1",
		Address:        "10.0.0.1",
		CredentialsRef: "creds-1",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAddWorkerNode_HappyPath(t *testing.T) {
	pollCount := 0
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		createStaticInstanceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		getStaticInstanceFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			pollCount++
			phase := "Pending"
			if pollCount >= 2 {
				phase = "Running"
			}
			return &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"currentStatus": map[string]interface{}{
							"phase": phase,
						},
					},
				},
			}, nil
		},
	}

	h := NewNodesHandler(mc)
	timeout := int32(5) // short timeout for test
	resp, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNodeRequest{
		Address:        "10.0.0.1",
		SshUser:        "ubuntu",
		PrivateKey:     "key",
		NodeGroup:      "workers",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeName != "10-0-0-1" {
		t.Errorf("expected 10-0-0-1, got %s", resp.NodeName)
	}
	if resp.SshCredentialsName != "10-0-0-1-creds" {
		t.Errorf("expected 10-0-0-1-creds, got %s", resp.SshCredentialsName)
	}
	if resp.Phase != "Running" {
		t.Errorf("expected Running, got %s", resp.Phase)
	}
	if resp.TimedOut {
		t.Error("expected timedOut=false")
	}
}

func TestAddWorkerNode_Timeout(t *testing.T) {
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		createStaticInstanceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"currentStatus": map[string]interface{}{
							"phase": "Bootstrapping",
						},
					},
				},
			}, nil
		},
	}

	h := NewNodesHandler(mc)
	timeout := int32(1)
	resp, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNodeRequest{
		Address:        "10.0.0.1",
		SshUser:        "ubuntu",
		PrivateKey:     "key",
		NodeGroup:      "workers",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.TimedOut {
		t.Error("expected timedOut=true")
	}
	if resp.Phase != "Bootstrapping" {
		t.Errorf("expected Bootstrapping, got %s", resp.Phase)
	}
}

func TestAddWorkerNode_SSHCredsError(t *testing.T) {
	siCreated := false
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("ssh error")
		},
		createStaticInstanceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			siCreated = true
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNodeRequest{
		Address:    "10.0.0.1",
		SshUser:    "ubuntu",
		PrivateKey: "key",
		NodeGroup:  "workers",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if siCreated {
		t.Error("StaticInstance should NOT be created when SSHCredentials fails")
	}
}

func TestAddWorkerNode_StaticInstanceError(t *testing.T) {
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		createStaticInstanceFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("si error")
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNodeRequest{
		Address:    "10.0.0.1",
		SshUser:    "ubuntu",
		PrivateKey: "key",
		NodeGroup:  "workers",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// Error should mention SSHCredentials was already created.
	if !containsSubstring(err.Error(), "already created") {
		t.Errorf("error should mention SSHCredentials was already created: %v", err)
	}
}

func TestAddWorkerNode_NoWait(t *testing.T) {
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		createStaticInstanceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	waitReady := false
	resp, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNodeRequest{
		Address:    "10.0.0.1",
		SshUser:    "ubuntu",
		PrivateKey: "key",
		NodeGroup:  "workers",
		WaitReady:  &waitReady,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Phase != "Pending" {
		t.Errorf("expected Pending (no polling), got %s", resp.Phase)
	}
	if resp.TimedOut {
		t.Error("expected timedOut=false")
	}
}

func TestAddWorkerNode_GeneratedName(t *testing.T) {
	mc := &mockClient{
		createSSHCredentialsFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
		createStaticInstanceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	waitReady := false
	resp, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNodeRequest{
		Address:    "192.168.1.100",
		SshUser:    "ubuntu",
		PrivateKey: "key",
		NodeGroup:  "workers",
		WaitReady:  &waitReady,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeName != "192-168-1-100" {
		t.Errorf("expected 192-168-1-100, got %s", resp.NodeName)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure metav1 import is used.
var _ = metav1.Now()

func TestDeleteStaticInstance_Success(t *testing.T) {
	deleted := false
	mc := &mockClient{
		deleteStaticInstanceFunc: func(_ context.Context, name string) error {
			deleted = true
			return nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.DeleteStaticInstance(context.Background(), &pb.DeleteStaticInstanceRequest{Name: "node-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if !deleted {
		t.Error("expected deleteStaticInstance to be called")
	}
}

func TestDeleteStaticInstance_NotFound(t *testing.T) {
	mc := &mockClient{
		deleteStaticInstanceFunc: func(_ context.Context, name string) error {
			return fmt.Errorf("static instance %q not found", name)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.DeleteStaticInstance(context.Background(), &pb.DeleteStaticInstanceRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRemoveNode_DrainAndDelete(t *testing.T) {
	cordoned := false
	deleted := false
	podDeleted := false
	si := makeStaticInstance("node-1", "10.0.0.1", "Running", "workers")
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &si, nil
		},
		getNodeFunc: func(_ context.Context, _ string) (*corev1.Node, error) {
			n := makeNode("node-1", true)
			return &n, nil
		},
		cordonNodeFunc: func(_ context.Context, _ string) error {
			cordoned = true
			return nil
		},
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "app-pod", Namespace: "default"},
				Spec:       corev1.PodSpec{NodeName: "node-1"},
			}
			return []corev1.Pod{pod}, nil
		},
		deletePodFunc: func(_ context.Context, _, _ string) error {
			podDeleted = true
			return nil
		},
		deleteStaticInstanceFunc: func(_ context.Context, _ string) error {
			deleted = true
			return nil
		},
	}

	h := NewNodesHandler(mc)
	drain := true
	resp, err := h.RemoveNode(context.Background(), &pb.RemoveNodeRequest{Name: "node-1", Drain: &drain})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Deleted {
		t.Error("expected deleted=true")
	}
	if !cordoned {
		t.Error("expected cordon to be called when drain=true")
	}
	if !podDeleted {
		t.Error("expected pod deletion when drain=true")
	}
	if !deleted {
		t.Error("expected deleteStaticInstance to be called")
	}
}

func TestRemoveNode_NoDrain(t *testing.T) {
	cordoned := false
	si := makeStaticInstance("node-1", "10.0.0.1", "Running", "workers")
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &si, nil
		},
		getNodeFunc: func(_ context.Context, _ string) (*corev1.Node, error) {
			n := makeNode("node-1", true)
			return &n, nil
		},
		cordonNodeFunc: func(_ context.Context, _ string) error {
			cordoned = true
			return nil
		},
		deleteStaticInstanceFunc: func(_ context.Context, _ string) error {
			return nil
		},
	}

	h := NewNodesHandler(mc)
	drain := false
	_, err := h.RemoveNode(context.Background(), &pb.RemoveNodeRequest{Name: "node-1", Drain: &drain})
	if err != nil {
		t.Fatal(err)
	}
	if cordoned {
		t.Error("expected cordon NOT called when drain=false")
	}
}

func TestRemoveNode_NoStaticInstance(t *testing.T) {
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("not found: %s", name)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.RemoveNode(context.Background(), &pb.RemoveNodeRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateNodeGroup_Success(t *testing.T) {
	mc := &mockClient{
		createNodeGroupFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.CreateNodeGroup(context.Background(), &pb.CreateNodeGroupRequest{
		Name:     "workers",
		NodeType: "Static",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "workers" {
		t.Errorf("expected workers, got %s", resp.Name)
	}
}

func TestCreateNodeGroup_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		createNodeGroupFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, errors.NewAlreadyExists(schema.GroupResource{Group: "deckhouse.io", Resource: "nodegroups"}, "workers")
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateNodeGroup(context.Background(), &pb.CreateNodeGroupRequest{
		Name:     "workers",
		NodeType: "Static",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWaitNodeReady_Success(t *testing.T) {
	pollCount := 0
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			pollCount++
			phase := "Bootstrapping"
			if pollCount >= 2 {
				phase = "Running"
			}
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"currentStatus": map[string]interface{}{
							"phase": phase,
						},
					},
				},
			}
			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	timeout := int32(5)
	resp, err := h.WaitNodeReady(context.Background(), &pb.WaitNodeReadyRequest{Name: "node-1", TimeoutSeconds: &timeout})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Phase != "Running" {
		t.Errorf("expected Running, got %s", resp.Phase)
	}
	if resp.TimedOut {
		t.Error("expected timedOut=false")
	}
}

func TestWaitNodeReady_Timeout(t *testing.T) {
	mc := &mockClient{
		getStaticInstanceFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &unstructured.Unstructured{
				Object: map[string]interface{}{
					"status": map[string]interface{}{
						"currentStatus": map[string]interface{}{
							"phase": "Bootstrapping",
						},
					},
				},
			}, nil
		},
	}

	h := NewNodesHandler(mc)
	timeout := int32(1)
	resp, err := h.WaitNodeReady(context.Background(), &pb.WaitNodeReadyRequest{Name: "node-1", TimeoutSeconds: &timeout})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.TimedOut {
		t.Error("expected timedOut=true")
	}
	if resp.Phase != "Bootstrapping" {
		t.Errorf("expected Bootstrapping, got %s", resp.Phase)
	}
}

func TestCordonNode_Happy(t *testing.T) {
	cordoned := false
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       corev1.NodeSpec{Unschedulable: false},
			}, nil
		},
		cordonNodeFunc: func(_ context.Context, name string) error {
			if name != "worker-01" {
				t.Errorf("expected name worker-01, got %q", name)
			}
			cordoned = true
			return nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.CordonNode(context.Background(), &pb.CordonNodeRequest{Name: "worker-01"})
	if err != nil {
		t.Fatal(err)
	}
	if !cordoned {
		t.Error("expected CordonNode to be called")
	}
	if resp.PreviousState {
		t.Error("expected previousState=false (was not cordoned)")
	}
}

func TestCordonNode_AlreadyCordoned(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       corev1.NodeSpec{Unschedulable: true},
			}, nil
		},
		cordonNodeFunc: func(_ context.Context, _ string) error {
			return nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.CordonNode(context.Background(), &pb.CordonNodeRequest{Name: "worker-01"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.PreviousState {
		t.Error("expected previousState=true (already cordoned)")
	}
}

func TestCordonNode_NotFound(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return nil, fmt.Errorf("node %q not found", name)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CordonNode(context.Background(), &pb.CordonNodeRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUncordonNode_Happy(t *testing.T) {
	uncordoned := false
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       corev1.NodeSpec{Unschedulable: true},
			}, nil
		},
		uncordonNodeFunc: func(_ context.Context, _ string) error {
			uncordoned = true
			return nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.UncordonNode(context.Background(), &pb.UncordonNodeRequest{Name: "node-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetPreviousState() {
		t.Error("expected previousState=true (was cordoned)")
	}
	if !uncordoned {
		t.Error("expected UncordonNode to be called")
	}
}

func TestUncordonNode_AlreadyUncordoned(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: name},
				Spec:       corev1.NodeSpec{Unschedulable: false},
			}, nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.UncordonNode(context.Background(), &pb.UncordonNodeRequest{Name: "node-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetPreviousState() {
		t.Error("expected previousState=false (was already uncordoned)")
	}
}

func TestUncordonNode_NotFound(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			return nil, fmt.Errorf("node %q not found", name)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.UncordonNode(context.Background(), &pb.UncordonNodeRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteSSHCredentials_Happy(t *testing.T) {
	deleted := ""
	mc := &mockClient{
		deleteSSHCredentialsFunc: func(_ context.Context, name string) error {
			deleted = name
			return nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.DeleteSSHCredentials(context.Background(), &pb.DeleteSSHCredentialsRequest{Name: "creds-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetDeleted() {
		t.Error("expected deleted=true")
	}
	if deleted != "creds-1" {
		t.Errorf("expected deletion of creds-1, got %q", deleted)
	}
}

func TestDeleteSSHCredentials_NotFound(t *testing.T) {
	mc := &mockClient{
		deleteSSHCredentialsFunc: func(_ context.Context, name string) error {
			return errors.NewNotFound(schema.GroupResource{Group: "deckhouse.io", Resource: "sshcredentials"}, name)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.DeleteSSHCredentials(context.Background(), &pb.DeleteSSHCredentialsRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteNodeGroup_Happy(t *testing.T) {
	deleted := ""
	mc := &mockClient{
		deleteNodeGroupFunc: func(_ context.Context, name string) error {
			deleted = name
			return nil
		},
	}

	h := NewNodesHandler(mc)
	resp, err := h.DeleteNodeGroup(context.Background(), &pb.DeleteNodeGroupRequest{Name: "workers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetDeleted() {
		t.Error("expected deleted=true")
	}
	if deleted != "workers" {
		t.Errorf("expected deletion of 'workers', got %q", deleted)
	}
}

func TestDeleteNodeGroup_NotFound(t *testing.T) {
	mc := &mockClient{
		deleteNodeGroupFunc: func(_ context.Context, name string) error {
			return errors.NewNotFound(schema.GroupResource{Group: "deckhouse.io", Resource: "nodegroups"}, name)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.DeleteNodeGroup(context.Background(), &pb.DeleteNodeGroupRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// makeDrainPod returns a pod scheduled on nodeName with optional ownerRef and annotations.
func makeDrainPod(namespace, name, nodeName string, ownerRefs []metav1.OwnerReference, mirrorAnnotation bool) corev1.Pod {
	annotations := map[string]string{}
	if mirrorAnnotation {
		annotations["kubernetes.io/config.mirror"] = "true"
	}

	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       namespace,
			Name:            name,
			OwnerReferences: ownerRefs,
			Annotations:     annotations,
		},
		Spec: corev1.PodSpec{NodeName: nodeName},
	}
}

func dsOwner() []metav1.OwnerReference {
	c := true
	return []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
			Name:       "ds-1",
			Controller: &c,
		},
	}
}

func TestDrainNode_Happy(t *testing.T) {
	cordoned := false
	evicted := map[string]bool{}
	mc := &mockClient{
		cordonNodeFunc: func(_ context.Context, _ string) error {
			cordoned = true
			return nil
		},
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makeDrainPod("default", "web-1", "node-1", nil, false),
				makeDrainPod("default", "web-2", "node-1", nil, false),
				makeDrainPod("default", "elsewhere", "node-2", nil, false),
			}, nil
		},
		evictPodFunc: func(_ context.Context, namespace, name string) error {
			evicted[namespace+"/"+name] = true
			return nil
		},
	}

	timeout := int32(30)
	h := NewNodesHandler(mc)

	resp, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{
		Name:           "node-1",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cordoned {
		t.Error("expected cordon to be called")
	}
	if !resp.GetCordoned() {
		t.Error("expected cordoned=true in response")
	}
	if resp.GetEvictedCount() != 2 {
		t.Errorf("expected evictedCount=2, got %d", resp.GetEvictedCount())
	}
	if !evicted["default/web-1"] || !evicted["default/web-2"] {
		t.Errorf("expected web-1 and web-2 evicted, got %v", evicted)
	}
	if evicted["default/elsewhere"] {
		t.Error("must not evict pod on a different node")
	}
}

func TestDrainNode_SkipsDaemonSet(t *testing.T) {
	evicted := map[string]bool{}
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makeDrainPod("kube-system", "ds-pod", "node-1", dsOwner(), false),
				makeDrainPod("default", "web-1", "node-1", nil, false),
			}, nil
		},
		evictPodFunc: func(_ context.Context, namespace, name string) error {
			evicted[namespace+"/"+name] = true
			return nil
		},
	}

	timeout := int32(30)
	h := NewNodesHandler(mc)

	resp, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{
		Name:           "node-1",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetEvictedCount() != 1 {
		t.Errorf("expected evictedCount=1, got %d", resp.GetEvictedCount())
	}
	if evicted["kube-system/ds-pod"] {
		t.Error("DaemonSet pod must NOT be evicted")
	}
	if !evicted["default/web-1"] {
		t.Error("regular pod must be evicted")
	}
}

func TestDrainNode_SkipsMirror(t *testing.T) {
	evicted := map[string]bool{}
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makeDrainPod("kube-system", "static-apiserver", "node-1", nil, true),
				makeDrainPod("default", "web-1", "node-1", nil, false),
			}, nil
		},
		evictPodFunc: func(_ context.Context, namespace, name string) error {
			evicted[namespace+"/"+name] = true
			return nil
		},
	}

	timeout := int32(30)
	h := NewNodesHandler(mc)

	resp, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{
		Name:           "node-1",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetEvictedCount() != 1 {
		t.Errorf("expected evictedCount=1, got %d", resp.GetEvictedCount())
	}
	if evicted["kube-system/static-apiserver"] {
		t.Error("mirror pod must NOT be evicted")
	}
}

func TestDrainNode_CordonFails(t *testing.T) {
	listed := false
	mc := &mockClient{
		cordonNodeFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("cordon failure")
		},
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			listed = true
			return nil, nil
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{Name: "node-1"})
	if err == nil {
		t.Fatal("expected error from cordon failure, got nil")
	}
	if listed {
		t.Error("ListPods must NOT be called when cordon fails")
	}
}

func TestDrainNode_PodAlreadyGone(t *testing.T) {
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makeDrainPod("default", "web-1", "node-1", nil, false),
			}, nil
		},
		evictPodFunc: func(_ context.Context, _, name string) error {
			return errors.NewNotFound(schema.GroupResource{Resource: "pods"}, name)
		},
	}

	timeout := int32(30)
	h := NewNodesHandler(mc)

	resp, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{
		Name:           "node-1",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// IsNotFound counts as evicted (pod already gone).
	if resp.GetEvictedCount() != 1 {
		t.Errorf("expected evictedCount=1 (gone counts as evicted), got %d", resp.GetEvictedCount())
	}
	if len(resp.GetFailedPods()) != 0 {
		t.Errorf("expected no failed_pods, got %v", resp.GetFailedPods())
	}
}

func TestDrainNode_PDBBlocksThenSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("polling test, run with full -test.timeout")
	}

	var calls int32
	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makeDrainPod("default", "web-1", "node-1", nil, false),
			}, nil
		},
		evictPodFunc: func(_ context.Context, _, name string) error {
			calls++
			if calls == 1 {
				return errors.NewTooManyRequests(fmt.Sprintf("evicting %q would violate the budget", name), 1)
			}
			return nil
		},
	}

	timeout := int32(120)
	h := NewNodesHandler(mc)

	resp, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{
		Name:           "node-1",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetCordoned() {
		t.Error("expected cordoned=true")
	}
	if resp.GetEvictedCount() != 1 {
		t.Errorf("expected evictedCount=1 after retry, got %d", resp.GetEvictedCount())
	}
	if calls < 2 {
		t.Errorf("expected at least 2 EvictPod calls (PDB retry), got %d", calls)
	}
}

func TestDrainNode_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("polling test, run with full -test.timeout")
	}

	mc := &mockClient{
		listPodsFunc: func(_ context.Context, _ string) ([]corev1.Pod, error) {
			return []corev1.Pod{
				makeDrainPod("default", "web-1", "node-1", nil, false),
			}, nil
		},
		evictPodFunc: func(_ context.Context, _, name string) error {
			// Always blocked by PDB.
			return errors.NewTooManyRequests(fmt.Sprintf("evicting %q would violate the budget", name), 1)
		},
	}

	timeout := int32(30)
	h := NewNodesHandler(mc)

	resp, err := h.DrainNode(context.Background(), &pb.DrainNodeRequest{
		Name:           "node-1",
		TimeoutSeconds: &timeout,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetTimedOut() {
		t.Error("expected timedOut=true")
	}
	if resp.GetEvictedCount() != 0 {
		t.Errorf("expected evictedCount=0, got %d", resp.GetEvictedCount())
	}
}

func TestCreateNodeGroupConfiguration_Success(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createNodeGroupConfigurationFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj

			return obj, nil
		},
	}

	weight := int32(50)
	h := NewNodesHandler(mc)
	resp, err := h.CreateNodeGroupConfiguration(context.Background(), &pb.CreateNodeGroupConfigurationRequest{
		Name:       "kubelet-tuning",
		Content:    "#!/bin/bash\nset -e\necho ok",
		NodeGroups: []string{"worker", "master"},
		Weight:     &weight,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetCreated() {
		t.Error("expected created=true")
	}
	if resp.GetName() != "kubelet-tuning" {
		t.Errorf("expected echoed name=kubelet-tuning, got %q", resp.GetName())
	}

	if captured == nil {
		t.Fatal("expected CreateNodeGroupConfiguration to be called")
	}
	if captured.GetKind() != "NodeGroupConfiguration" {
		t.Errorf("expected kind=NodeGroupConfiguration, got %q", captured.GetKind())
	}
	if captured.GetAPIVersion() != "deckhouse.io/v1alpha1" {
		t.Errorf("expected apiVersion=deckhouse.io/v1alpha1, got %q", captured.GetAPIVersion())
	}
	if captured.GetName() != "kubelet-tuning" {
		t.Errorf("expected metadata.name=kubelet-tuning, got %q", captured.GetName())
	}

	spec, ok := captured.Object["spec"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec object, got %#v", captured.Object["spec"])
	}
	if got := spec["content"]; got != "#!/bin/bash\nset -e\necho ok" {
		t.Errorf("expected spec.content to echo request, got %q", got)
	}
	if got := spec["weight"]; got != int64(50) {
		t.Errorf("expected spec.weight=50, got %v (%T)", got, got)
	}
	ngs, ok := spec["nodeGroups"].([]any)
	if !ok {
		t.Fatalf("expected spec.nodeGroups []any, got %#v", spec["nodeGroups"])
	}
	if len(ngs) != 2 || ngs[0] != "worker" || ngs[1] != "master" {
		t.Errorf("expected spec.nodeGroups=[worker, master], got %v", ngs)
	}
}

func TestCreateNodeGroupConfiguration_DefaultWeight(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createNodeGroupConfigurationFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj

			return obj, nil
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateNodeGroupConfiguration(context.Background(), &pb.CreateNodeGroupConfigurationRequest{
		Name:       "default-weight",
		Content:    "#!/bin/bash\necho hi",
		NodeGroups: []string{"worker"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spec, ok := captured.Object["spec"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec object, got %#v", captured.Object["spec"])
	}
	if got := spec["weight"]; got != int64(100) {
		t.Errorf("expected default spec.weight=100, got %v (%T)", got, got)
	}
}

func TestCreateNodeGroupConfiguration_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		createNodeGroupConfigurationFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, errors.NewAlreadyExists(
				schema.GroupResource{Group: "deckhouse.io", Resource: "nodegroupconfigurations"},
				"duplicate",
			)
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateNodeGroupConfiguration(context.Background(), &pb.CreateNodeGroupConfigurationRequest{
		Name:       "duplicate",
		Content:    "#!/bin/bash",
		NodeGroups: []string{"worker"},
	})
	if err == nil {
		t.Fatal("expected already-exists error, got nil")
	}
	if !errors.IsAlreadyExists(err) {
		t.Errorf("expected error to preserve IsAlreadyExists semantics, got %v", err)
	}
}

func TestCreateNodeGroupConfiguration_EmptyContent(t *testing.T) {
	called := false
	mc := &mockClient{
		createNodeGroupConfigurationFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			called = true

			return nil, nil
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateNodeGroupConfiguration(context.Background(), &pb.CreateNodeGroupConfigurationRequest{
		Name:       "empty-content",
		Content:    "",
		NodeGroups: []string{"worker"},
	})
	if err == nil {
		t.Fatal("expected validation error for empty content, got nil")
	}
	if called {
		t.Error("expected k8s.Client to NOT be called when content is empty")
	}
}

func TestCreateNodeGroupConfiguration_EmptyNodeGroups(t *testing.T) {
	called := false
	mc := &mockClient{
		createNodeGroupConfigurationFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			called = true

			return nil, nil
		},
	}

	h := NewNodesHandler(mc)
	_, err := h.CreateNodeGroupConfiguration(context.Background(), &pb.CreateNodeGroupConfigurationRequest{
		Name:       "empty-ngs",
		Content:    "#!/bin/bash\necho",
		NodeGroups: nil,
	})
	if err == nil {
		t.Fatal("expected validation error for empty node_groups, got nil")
	}
	if called {
		t.Error("expected k8s.Client to NOT be called when node_groups is empty")
	}
}
