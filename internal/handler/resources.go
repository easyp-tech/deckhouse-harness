package handler

import (
	"context"
	"fmt"

	"github.com/easyp-tech/protoc-gen-mcp/mcpruntime"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	"github.com/easyp-tech/deckhouse-harness/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

// ResourcesHandler implements the generated MCP resource handler interface
// (pb.File_proto_deckhouse_v1_resources_protoResourceHandler). It exposes a
// curated subset of read-only data as URI-addressable resources, reusing the
// existing tool handlers as backing logic — no data-access code is duplicated.
type ResourcesHandler struct {
	diag *DiagnosticsHandler
	cfg  *ConfigHandler
	mod  *ModulesHandler
	rel  *ReleasesHandler
}

// NewResourcesHandler creates a ResourcesHandler backed by the given k8s client.
func NewResourcesHandler(client k8s.Client) *ResourcesHandler {
	return &ResourcesHandler{
		diag: NewDiagnosticsHandler(client),
		cfg:  NewConfigHandler(client),
		mod:  NewModulesHandler(client),
		rel:  NewReleasesHandler(client),
	}
}

// ---- Static resources ----

// ReadClusterStatus backs deckhouse://cluster/status.
func (h *ResourcesHandler) ReadClusterStatus(ctx context.Context) (*pb.ClusterStatus, error) {
	status, err := h.diag.GetClusterStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return &pb.ClusterStatus{Status: status}, nil
}

// ReadClusterConfiguration backs deckhouse://cluster/configuration.
func (h *ResourcesHandler) ReadClusterConfiguration(ctx context.Context) (*pb.ClusterConfiguration, error) {
	cfg, err := h.cfg.GetClusterConfiguration(ctx, &pb.GetClusterConfigurationRequest{})
	if err != nil {
		return nil, err
	}

	return &pb.ClusterConfiguration{Configuration: cfg}, nil
}

// ReadNodeList backs deckhouse://nodes.
func (h *ResourcesHandler) ReadNodeList(ctx context.Context) (*pb.NodeList, error) {
	resp, err := h.diag.ListNodes(ctx, &pb.ListNodesRequest{})
	if err != nil {
		return nil, err
	}

	return &pb.NodeList{Nodes: resp.GetNodes()}, nil
}

// ReadModuleList backs deckhouse://modules.
func (h *ResourcesHandler) ReadModuleList(ctx context.Context) (*pb.ModuleList, error) {
	resp, err := h.mod.ListModules(ctx, &pb.ListModulesRequest{})
	if err != nil {
		return nil, err
	}

	return &pb.ModuleList{Modules: resp.GetModules()}, nil
}

// ReadDeckhouseReleaseList backs deckhouse://releases.
func (h *ResourcesHandler) ReadDeckhouseReleaseList(ctx context.Context) (*pb.DeckhouseReleaseList, error) {
	resp, err := h.rel.ListDeckhouseReleases(ctx, &pb.ListDeckhouseReleasesRequest{})
	if err != nil {
		return nil, err
	}

	return &pb.DeckhouseReleaseList{Releases: resp.GetReleases()}, nil
}

// ---- Templated resources ----

// ListNodes enumerates node instances for deckhouse://nodes/{name}. This runs
// once at registration time (a startup snapshot); live reads always go through
// ReadNode. Enumeration failures degrade to no instances rather than blocking
// registration — the template still allows on-demand reads.
func (h *ResourcesHandler) ListNodes(ctx context.Context) ([]mcpruntime.Resource, error) {
	resp, err := h.diag.ListNodes(ctx, &pb.ListNodesRequest{})
	if err != nil {
		return nil, nil
	}

	resources := make([]mcpruntime.Resource, 0, len(resp.GetNodes()))
	for _, node := range resp.GetNodes() {
		name := node.GetName()
		resources = append(resources, mcpruntime.Resource{
			URI:         fmt.Sprintf("deckhouse://nodes/%s", name),
			Name:        name,
			Description: fmt.Sprintf("Node %s (%s)", name, node.GetStatus()),
			MIMEType:    "application/json",
		})
	}

	return resources, nil
}

// ReadNode backs deckhouse://nodes/{name}.
func (h *ResourcesHandler) ReadNode(ctx context.Context, name string) (*pb.Node, error) {
	node, err := h.diag.GetNode(ctx, &pb.GetNodeRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return &pb.Node{Node: node}, nil
}

// ListModules intentionally enumerates no concrete instances. A Deckhouse
// cluster has 100+ modules, so listing each as a separate resource would swamp
// resources/list. Module names are already discoverable via the aggregate
// deckhouse://modules resource, and any module is readable on demand through the
// deckhouse://modules/{name} template.
func (h *ResourcesHandler) ListModules(_ context.Context) ([]mcpruntime.Resource, error) {
	return nil, nil
}

// ReadModule backs deckhouse://modules/{name}.
func (h *ResourcesHandler) ReadModule(ctx context.Context, name string) (*pb.Module, error) {
	cfg, err := h.mod.GetModuleConfig(ctx, &pb.GetModuleConfigRequest{Name: name})
	if err != nil {
		return nil, err
	}

	return &pb.Module{Config: cfg}, nil
}
