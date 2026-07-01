package handler

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

func TestK8sError_NotFound(t *testing.T) {
	mc := &mockClient{
		listModuleConfigsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, errors.NewNotFound(schema.GroupResource{Group: "deckhouse.io", Resource: "moduleconfigs"}, "test")
		},
	}

	h := NewModulesHandler(mc)
	_, err := h.ListModuleConfigs(context.Background(), &pb.ListModuleConfigsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestK8sError_Forbidden(t *testing.T) {
	mc := &mockClient{
		listDeckhouseReleasesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, errors.NewForbidden(schema.GroupResource{Group: "deckhouse.io", Resource: "deckhouserelease"}, "test", fmt.Errorf("forbidden"))
		},
	}

	h := NewReleasesHandler(mc)
	_, err := h.ListDeckhouseReleases(context.Background(), &pb.ListDeckhouseReleasesRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestK8sError_Unavailable(t *testing.T) {
	mc := &mockClient{
		listNodesFunc: func(_ context.Context) ([]corev1.Node, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	h := NewDiagnosticsHandler(mc)
	_, err := h.GetClusterStatus(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
