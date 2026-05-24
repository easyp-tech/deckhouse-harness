package handler

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

func TestListDeckhouseReleases_All(t *testing.T) {
	mc := &mockClient{
		listDeckhouseReleasesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeRelease("v1.60.0", "Deployed", "1.60.0"),
				makeRelease("v1.61.0", "Pending", "1.61.0"),
			}, nil
		},
	}

	h := NewReleasesHandler(mc)
	resp, err := h.ListDeckhouseReleases(context.Background(), &pb.ListDeckhouseReleasesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Releases) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Releases))
	}
}

func TestListDeckhouseReleases_ByPhase(t *testing.T) {
	mc := &mockClient{
		listDeckhouseReleasesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeRelease("v1.60.0", "Deployed", "1.60.0"),
				makeRelease("v1.61.0", "Pending", "1.61.0"),
				makeRelease("v1.59.0", "Superseded", "1.59.0"),
			}, nil
		},
	}

	h := NewReleasesHandler(mc)
	phase := pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_PENDING
	resp, err := h.ListDeckhouseReleases(context.Background(), &pb.ListDeckhouseReleasesRequest{Phase: &phase})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Releases) != 1 {
		t.Errorf("expected 1, got %d", len(resp.Releases))
	}
	if resp.Releases[0].Name != "v1.61.0" {
		t.Errorf("expected v1.61.0, got %s", resp.Releases[0].Name)
	}
}

func TestGetDeckhouseRelease_Found(t *testing.T) {
	mc := &mockClient{
		getDeckhouseReleaseFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			obj := makeRelease(name, "Pending", "1.74.0")
			return &obj, nil
		},
	}

	h := NewReleasesHandler(mc)
	resp, err := h.GetDeckhouseRelease(context.Background(), &pb.GetDeckhouseReleaseRequest{Version: "v1.74.0"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "v1.74.0" {
		t.Errorf("expected v1.74.0, got %s", resp.Name)
	}
	if resp.Phase != "Pending" {
		t.Errorf("expected Pending phase, got %s", resp.Phase)
	}
}

func TestGetDeckhouseRelease_NotFound(t *testing.T) {
	mc := &mockClient{
		getDeckhouseReleaseFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("release %q not found", name)
		},
	}

	h := NewReleasesHandler(mc)
	_, err := h.GetDeckhouseRelease(context.Background(), &pb.GetDeckhouseReleaseRequest{Version: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestApproveRelease_Success(t *testing.T) {
	patched := false
	existing := makeRelease("v1.74.0", "Pending", "1.74.0")
	mc := &mockClient{
		getDeckhouseReleaseFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &existing, nil
		},
		patchDeckhouseReleaseFunc: func(_ context.Context, _ string, _ []byte) (*unstructured.Unstructured, error) {
			patched = true
			return &existing, nil
		},
	}

	h := NewReleasesHandler(mc)
	resp, err := h.ApproveRelease(context.Background(), &pb.ApproveReleaseRequest{Version: "v1.74.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if !patched {
		t.Error("expected patch to be called")
	}
}

func TestApproveRelease_AlreadyApproved(t *testing.T) {
	existing := makeRelease("v1.74.0", "Pending", "1.74.0")
	// Set the approved annotation.
	existing.SetAnnotations(map[string]string{
		"release.deckhouse.io/approved": "true",
	})
	patchCalled := false
	mc := &mockClient{
		getDeckhouseReleaseFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &existing, nil
		},
		patchDeckhouseReleaseFunc: func(_ context.Context, _ string, _ []byte) (*unstructured.Unstructured, error) {
			patchCalled = true
			return &existing, nil
		},
	}

	h := NewReleasesHandler(mc)
	resp, err := h.ApproveRelease(context.Background(), &pb.ApproveReleaseRequest{Version: "v1.74.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success=true even if already approved")
	}
	// Whether or not patch is called is an implementation detail; both idempotent patterns are valid.
	_ = patchCalled
}

func TestApproveRelease_NotFound(t *testing.T) {
	mc := &mockClient{
		getDeckhouseReleaseFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("release %q not found", name)
		},
	}

	h := NewReleasesHandler(mc)
	_, err := h.ApproveRelease(context.Background(), &pb.ApproveReleaseRequest{Version: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
