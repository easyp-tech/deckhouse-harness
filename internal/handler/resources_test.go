package handler

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestResources_ReadNodeList(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{makeNode("node-1", true), makeNode("node-2", false)}, nil
		},
	}
	h := NewResourcesHandler(mc)

	res, err := h.ReadNodeList(context.Background())
	if err != nil {
		t.Fatalf("ReadNodeList: unexpected error: %v", err)
	}
	if got := len(res.GetNodes()); got != 2 {
		t.Fatalf("nodes: got %d, want 2", got)
	}
	if res.GetNodes()[0].GetName() != "node-1" {
		t.Errorf("node[0] name: got %q, want node-1", res.GetNodes()[0].GetName())
	}
}

func TestResources_ReadModuleList(t *testing.T) {
	mc := &mockClient{
		listModulesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{makeModule("cni-cilium", 30, "embedded", "Ready")}, nil
		},
	}
	h := NewResourcesHandler(mc)

	res, err := h.ReadModuleList(context.Background())
	if err != nil {
		t.Fatalf("ReadModuleList: unexpected error: %v", err)
	}
	if got := len(res.GetModules()); got != 1 {
		t.Fatalf("modules: got %d, want 1", got)
	}
	if res.GetModules()[0].GetName() != "cni-cilium" {
		t.Errorf("module name: got %q, want cni-cilium", res.GetModules()[0].GetName())
	}
}

func TestResources_ReadDeckhouseReleaseList(t *testing.T) {
	mc := &mockClient{
		listDeckhouseReleasesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{makeRelease("v1.74.0", "Deployed", "v1.74.0")}, nil
		},
	}
	h := NewResourcesHandler(mc)

	res, err := h.ReadDeckhouseReleaseList(context.Background())
	if err != nil {
		t.Fatalf("ReadDeckhouseReleaseList: unexpected error: %v", err)
	}
	if got := len(res.GetReleases()); got != 1 {
		t.Fatalf("releases: got %d, want 1", got)
	}
}

func TestResources_ReadClusterConfiguration(t *testing.T) {
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			if namespace != "kube-system" || name != "d8-cluster-configuration" {
				t.Errorf("unexpected secret ref: %s/%s", namespace, name)
			}
			return makeClusterConfigSecret("kubernetesVersion: \"1.29\"\n"), nil
		},
	}
	h := NewResourcesHandler(mc)

	res, err := h.ReadClusterConfiguration(context.Background())
	if err != nil {
		t.Fatalf("ReadClusterConfiguration: unexpected error: %v", err)
	}
	if res.GetConfiguration().GetConfiguration() == "" {
		t.Error("configuration YAML is empty")
	}
}

func TestResources_ReadClusterStatus_Empty(t *testing.T) {
	// An all-zero mock must still yield a non-nil status (graceful, no error).
	h := NewResourcesHandler(&mockClient{})

	res, err := h.ReadClusterStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadClusterStatus: unexpected error: %v", err)
	}
	if res.GetStatus() == nil {
		t.Fatal("status is nil")
	}
}

func TestResources_ReadNode(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, name string) (*corev1.Node, error) {
			node := makeNode(name, true)
			return &node, nil
		},
	}
	h := NewResourcesHandler(mc)

	res, err := h.ReadNode(context.Background(), "node-1")
	if err != nil {
		t.Fatalf("ReadNode: unexpected error: %v", err)
	}
	if res.GetNode().GetNode().GetName() != "node-1" {
		t.Errorf("node name: got %q, want node-1", res.GetNode().GetNode().GetName())
	}
}

func TestResources_ReadNode_Error(t *testing.T) {
	mc := &mockClient{
		getNodeFunc: func(_ context.Context, _ string) (*corev1.Node, error) {
			return nil, errors.New("not found")
		},
	}
	h := NewResourcesHandler(mc)

	if _, err := h.ReadNode(context.Background(), "missing"); err == nil {
		t.Fatal("ReadNode: expected error, got nil")
	}
}

func TestResources_ReadModule(t *testing.T) {
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			mc := makeModuleConfig(name, true, "")
			return &mc, nil
		},
	}
	h := NewResourcesHandler(mc)

	res, err := h.ReadModule(context.Background(), "cni-cilium")
	if err != nil {
		t.Fatalf("ReadModule: unexpected error: %v", err)
	}
	if res.GetConfig().GetName() != "cni-cilium" {
		t.Errorf("module config name: got %q, want cni-cilium", res.GetConfig().GetName())
	}
}

func TestResources_ListNodes_Enumeration(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return []corev1.Node{makeNode("node-1", true), makeNode("node-2", true)}, nil
		},
	}
	h := NewResourcesHandler(mc)

	instances, err := h.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes: unexpected error: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("instances: got %d, want 2", len(instances))
	}
	if instances[0].URI != "deckhouse://nodes/node-1" {
		t.Errorf("instance[0] URI: got %q, want deckhouse://nodes/node-1", instances[0].URI)
	}
}

func TestResources_ListNodes_DegradesOnError(t *testing.T) {
	// Enumeration failures must not block registration — return no instances.
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return nil, errors.New("api down")
		},
	}
	h := NewResourcesHandler(mc)

	instances, err := h.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes: expected graceful degradation, got error: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("instances: got %d, want 0", len(instances))
	}
}

func TestResources_ListModules_NoEnumeration(t *testing.T) {
	// Modules are deliberately not enumerated per-instance (100+ modules would
	// swamp resources/list); the template still allows on-demand reads.
	h := NewResourcesHandler(&mockClient{})

	instances, err := h.ListModules(context.Background())
	if err != nil {
		t.Fatalf("ListModules: unexpected error: %v", err)
	}
	if len(instances) != 0 {
		t.Fatalf("instances: got %d, want 0", len(instances))
	}
}
