// Package handler — SourcesAPI handlers for Deckhouse ModuleSource and
// ModuleUpdatePolicy CRDs.
package handler

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/easyp-tech/deckhouse-harness/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

// errModuleNameRequired is returned when ListModuleReleases is called with an empty module_name.
var errModuleNameRequired = errors.New("module_name is required")

// errActiveReleasesBlockDeletion is returned when DeleteModuleSource is invoked
// without force while at least one ModuleRelease still references the source.
var errActiveReleasesBlockDeletion = errors.New("active module releases reference this source — pass force=true to override")

// errMatchLabelsRequired is returned when CreateModuleUpdatePolicy is called
// without match_labels. The Deckhouse webhook rejects ModuleUpdatePolicy
// resources whose spec.moduleReleaseSelector is missing or empty.
var errMatchLabelsRequired = errors.New("match_labels is required: spec.moduleReleaseSelector.labelSelector.matchLabels must contain at least one entry")

const deckhouseAPIVersionV1Alpha1 = "deckhouse.io/v1alpha1"

// SourcesHandler implements pb.SourcesAPIToolHandler.
type SourcesHandler struct {
	client k8s.Client
}

// NewSourcesHandler creates a new SourcesHandler.
func NewSourcesHandler(client k8s.Client) *SourcesHandler {
	return &SourcesHandler{client: client}
}

// ListModuleSources returns all ModuleSource resources, projecting each into
// pb.ModuleSourceInfo (name + spec.registry.repo + status.message).
func (h *SourcesHandler) ListModuleSources(
	ctx context.Context,
	_ *pb.ListModuleSourcesRequest,
) (*pb.ListModuleSourcesResponse, error) {
	items, err := h.client.ListModuleSources(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing module sources: %w", err)
	}

	sources := make([]*pb.ModuleSourceInfo, 0, len(items))
	for i := range items {
		obj := &items[i]
		sources = append(sources, &pb.ModuleSourceInfo{
			Name:     obj.GetName(),
			Registry: extractModuleSourceRegistry(obj),
			Status:   extractModuleSourceStatus(obj),
		})
	}

	return &pb.ListModuleSourcesResponse{Sources: sources}, nil
}

// CreateModuleSource creates a new ModuleSource resource. The request name and
// registry are mapped to metadata.name and spec.registry.repo respectively.
func (h *SourcesHandler) CreateModuleSource(
	ctx context.Context,
	req *pb.CreateModuleSourceRequest,
) (*pb.CreateModuleSourceResponse, error) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": deckhouseAPIVersionV1Alpha1,
		"kind":       "ModuleSource",
		"metadata": map[string]any{
			"name": req.GetName(),
		},
		"spec": map[string]any{
			"registry": map[string]any{
				"repo": req.GetRegistry(),
			},
		},
	}}

	_, err := h.client.CreateModuleSource(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("creating module source %s: %w", req.GetName(), err)
	}

	return &pb.CreateModuleSourceResponse{
		Created: true,
		Name:    req.GetName(),
	}, nil
}

// DeleteModuleSource deletes a ModuleSource resource. Unless force=true, it
// first lists active ModuleRelease resources owned by the same source (label
// selector "source=<name>") and refuses deletion when any exist.
//
// REQ-2.2/2.3/2.4: safety pre-check, force flag, cascade is delegated to
// Deckhouse owner references.
func (h *SourcesHandler) DeleteModuleSource(
	ctx context.Context,
	req *pb.DeleteModuleSourceRequest,
) (*pb.DeleteModuleSourceResponse, error) {
	name := req.GetName()

	force := false
	if req.Force != nil {
		force = *req.Force
	}

	if !force {
		releases, err := h.client.ListModuleReleases(ctx, "")
		if err == nil {
			active := countSourceReferences(releases, name)
			if active > 0 {
				return nil, fmt.Errorf(
					"%w (%d active releases for source %q)",
					errActiveReleasesBlockDeletion, active, name,
				)
			}
		}
	}

	err := h.client.DeleteModuleSource(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("deleting module source %s: %w", name, err)
	}

	msg := fmt.Sprintf("ModuleSource %q deleted", name)
	if force {
		msg += " (force=true)"
	}

	return &pb.DeleteModuleSourceResponse{
		Deleted: true,
		Message: msg,
	}, nil
}

// countSourceReferences counts ModuleRelease items whose labels["source"]
// matches sourceName.
func countSourceReferences(items []unstructured.Unstructured, sourceName string) int {
	count := 0

	for i := range items {
		if extractLabel(&items[i], "source") == sourceName {
			count++
		}
	}

	return count
}

// ListModuleUpdatePolicies returns all ModuleUpdatePolicy resources, projecting
// each into pb.ModuleUpdatePolicyInfo (name + spec.update.mode).
func (h *SourcesHandler) ListModuleUpdatePolicies(
	ctx context.Context,
	_ *pb.ListModuleUpdatePoliciesRequest,
) (*pb.ListModuleUpdatePoliciesResponse, error) {
	items, err := h.client.ListModuleUpdatePolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing module update policies: %w", err)
	}

	policies := make([]*pb.ModuleUpdatePolicyInfo, 0, len(items))
	for i := range items {
		obj := &items[i]
		policies = append(policies, &pb.ModuleUpdatePolicyInfo{
			Name:       obj.GetName(),
			UpdateMode: extractUpdatePolicyMode(obj),
		})
	}

	return &pb.ListModuleUpdatePoliciesResponse{Policies: policies}, nil
}

// CreateModuleUpdatePolicy creates a new ModuleUpdatePolicy resource. The
// request name and updateMode are mapped to metadata.name and spec.update.mode;
// match_labels populates spec.moduleReleaseSelector.labelSelector.matchLabels
// (required by the Deckhouse validating webhook).
func (h *SourcesHandler) CreateModuleUpdatePolicy(
	ctx context.Context,
	req *pb.CreateModuleUpdatePolicyRequest,
) (*pb.CreateModuleUpdatePolicyResponse, error) {
	if len(req.GetMatchLabels()) == 0 {
		return nil, errMatchLabelsRequired
	}

	matchLabels := make(map[string]any, len(req.GetMatchLabels()))
	for k, v := range req.GetMatchLabels() {
		matchLabels[k] = v
	}

	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": deckhouseAPIVersionV1Alpha1,
		"kind":       "ModuleUpdatePolicy",
		"metadata": map[string]any{
			"name": req.GetName(),
		},
		"spec": map[string]any{
			"update": map[string]any{
				"mode": req.GetUpdateMode(),
			},
			"moduleReleaseSelector": map[string]any{
				"labelSelector": map[string]any{
					"matchLabels": matchLabels,
				},
			},
		},
	}}

	_, err := h.client.CreateModuleUpdatePolicy(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("creating module update policy %s: %w", req.GetName(), err)
	}

	return &pb.CreateModuleUpdatePolicyResponse{
		Created: true,
		Name:    req.GetName(),
	}, nil
}

// extractModuleSourceRegistry reads spec.registry.repo as a string. Returns
// empty string if the path is absent or not a string.
func extractModuleSourceRegistry(obj *unstructured.Unstructured) string {
	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return ""
	}

	registry, ok := spec["registry"].(map[string]any)
	if !ok {
		return ""
	}

	repo, _ := registry["repo"].(string)

	return repo
}

// extractModuleSourceStatus reads status.message as a string. Returns empty
// string if the path is absent or not a string.
func extractModuleSourceStatus(obj *unstructured.Unstructured) string {
	status, ok := obj.Object["status"].(map[string]any)
	if !ok {
		return ""
	}

	msg, _ := status["message"].(string)

	return msg
}

// extractUpdatePolicyMode reads spec.update.mode as a string. Returns empty
// string if the path is absent or not a string.
func extractUpdatePolicyMode(obj *unstructured.Unstructured) string {
	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return ""
	}

	update, ok := spec["update"].(map[string]any)
	if !ok {
		return ""
	}

	mode, _ := update["mode"].(string)

	return mode
}

// ListModuleReleases returns ModuleRelease resources for a specific module,
// optionally narrowed to a single lifecycle phase.
//
// REQ-1.4: module_name is required and validated locally before any K8s API
// call. REQ-1.1/1.2: the module_name filter is applied by the K8s API via
// label selector "module=<name>"; the optional phase filter is applied
// in-process to status.phase.
func (h *SourcesHandler) ListModuleReleases(
	ctx context.Context,
	req *pb.ListModuleReleasesRequest,
) (*pb.ListModuleReleasesResponse, error) {
	moduleName := req.GetModuleName()
	if moduleName == "" {
		return nil, errModuleNameRequired
	}

	items, err := h.client.ListModuleReleases(ctx, moduleName)
	if err != nil {
		return nil, fmt.Errorf("listing module releases for %s: %w", moduleName, err)
	}

	phaseFilter := ""
	if req.Phase != nil {
		phaseFilter = *req.Phase
	}

	releases := make([]*pb.ModuleReleaseInfo, 0, len(items))
	for i := range items {
		obj := &items[i]

		phase := extractModuleReleasePhase(obj)
		if phaseFilter != "" && phase != phaseFilter {
			continue
		}

		releases = append(releases, &pb.ModuleReleaseInfo{
			Name:     obj.GetName(),
			Module:   extractLabel(obj, "module"),
			Version:  extractModuleReleaseVersion(obj),
			Source:   extractLabel(obj, "source"),
			Phase:    phase,
			Approved: extractModuleReleaseApproved(obj),
		})
	}

	return &pb.ListModuleReleasesResponse{Releases: releases}, nil
}

// extractLabel reads metadata.labels[key] as a string. Returns empty string if
// the label is absent.
func extractLabel(obj *unstructured.Unstructured, key string) string {
	labels := obj.GetLabels()
	if labels == nil {
		return ""
	}

	return labels[key]
}

// extractModuleReleaseVersion reads spec.version as a string.
func extractModuleReleaseVersion(obj *unstructured.Unstructured) string {
	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return ""
	}

	v, _ := spec["version"].(string)

	return v
}

// extractModuleReleasePhase reads status.phase as a string.
func extractModuleReleasePhase(obj *unstructured.Unstructured) string {
	status, ok := obj.Object["status"].(map[string]any)
	if !ok {
		return ""
	}

	p, _ := status["phase"].(string)

	return p
}

// extractModuleReleaseApproved reads spec.approved. The CRD allows bool or
// missing; we normalise to "true" / "false" / "".
func extractModuleReleaseApproved(obj *unstructured.Unstructured) string {
	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return ""
	}

	switch v := spec["approved"].(type) {
	case bool:
		if v {
			return "true"
		}

		return "false"
	case string:
		return v
	default:
		return ""
	}
}
