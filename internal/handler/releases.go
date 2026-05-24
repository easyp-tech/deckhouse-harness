package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/easyp-tech/deckhouse-mcp/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

// ReleasesHandler implements pb.ReleasesAPIToolHandler.
type ReleasesHandler struct {
	client k8s.Client
}

// NewReleasesHandler creates a new ReleasesHandler.
func NewReleasesHandler(client k8s.Client) *ReleasesHandler {
	return &ReleasesHandler{client: client}
}

func (h *ReleasesHandler) ListDeckhouseReleases(
	ctx context.Context,
	req *pb.ListDeckhouseReleasesRequest,
) (*pb.ListDeckhouseReleasesResponse, error) {
	releases, err := h.client.ListDeckhouseReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing deckhouse releases: %w", err)
	}

	var result []*pb.DeckhouseReleaseInfo

	for _, release := range releases {
		name := unstructuredNestedString(release.Object, "metadata", "name")
		phase := unstructuredNestedString(release.Object, "status", "phase")
		version := unstructuredNestedString(release.Object, "spec", "version")
		transitionTime := unstructuredNestedString(release.Object, "status", "transitionTime")
		changelogLink := unstructuredNestedString(release.Object, "spec", "changelogLink")
		approvedVal, _ := nestedField(release.Object, "status", "approved")

		approved := false
		if b, ok := approvedVal.(bool); ok {
			approved = b
		}

		if req.Phase != nil && req.GetPhase() != pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_UNSPECIFIED {
			wantPhase := phaseEnumToString(req.GetPhase())
			if phase != wantPhase {
				continue
			}
		}

		result = append(result, &pb.DeckhouseReleaseInfo{
			Name:           name,
			Phase:          phase,
			Version:        version,
			TransitionTime: transitionTime,
			ChangelogLink:  changelogLink,
			Approved:       approved,
		})
	}

	return &pb.ListDeckhouseReleasesResponse{Releases: result}, nil
}

// phaseEnumToString converts a DeckhouseReleasePhase enum to the string used in the CRD status.
func phaseEnumToString(p pb.DeckhouseReleasePhase) string {
	switch p {
	case pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_PENDING:
		return phasePending
	case pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_DEPLOYED:
		return "Deployed"
	case pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_SUPERSEDED:
		return "Superseded"
	case pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_SKIPPED:
		return "Skipped"
	case pb.DeckhouseReleasePhase_DECKHOUSE_RELEASE_PHASE_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

// GetDeckhouseRelease returns full details about a specific DeckhouseRelease by version.
// In Deckhouse the resource name matches the version (e.g. "v1.74.0").
func (h *ReleasesHandler) GetDeckhouseRelease(
	ctx context.Context,
	req *pb.GetDeckhouseReleaseRequest,
) (*pb.GetDeckhouseReleaseResponse, error) {
	release, err := h.client.GetDeckhouseRelease(ctx, req.GetVersion())
	if err != nil {
		return nil, fmt.Errorf("getting deckhouse release %s: %w", req.GetVersion(), err)
	}

	name := unstructuredNestedString(release.Object, "metadata", "name")
	phase := unstructuredNestedString(release.Object, "status", "phase")
	version := unstructuredNestedString(release.Object, "spec", "version")
	transitionTime := unstructuredNestedString(release.Object, "status", "transitionTime")
	changelogLink := unstructuredNestedString(release.Object, "spec", "changelogLink")
	approvedVal, _ := nestedField(release.Object, "status", "approved")

	approved := false
	if b, ok := approvedVal.(bool); ok {
		approved = b
	}

	// Also check annotation-based approval.
	annotations := release.GetAnnotations()
	if annotations["release.deckhouse.io/approved"] == "true" {
		approved = true
	}

	// Read requirements map.
	requirements := make(map[string]string)

	reqVal, _ := nestedField(release.Object, "spec", "requirements")
	if m, ok := reqVal.(map[string]any); ok {
		for k, v := range m {
			requirements[k] = fmt.Sprintf("%v", v)
		}
	}

	return &pb.GetDeckhouseReleaseResponse{
		Name:           name,
		Phase:          phase,
		Version:        version,
		TransitionTime: transitionTime,
		Approved:       approved,
		ChangelogLink:  changelogLink,
		Requirements:   requirements,
	}, nil
}

// ApproveRelease approves a pending Deckhouse release via merge patch on the annotation.
func (h *ReleasesHandler) ApproveRelease(
	ctx context.Context,
	req *pb.ApproveReleaseRequest,
) (*pb.ApproveReleaseResponse, error) {
	// First check if it exists and read current approval state.
	release, err := h.client.GetDeckhouseRelease(ctx, req.GetVersion())
	if err != nil {
		return nil, fmt.Errorf("getting deckhouse release %s: %w", req.GetVersion(), err)
	}

	// Check if already approved.
	annotations := release.GetAnnotations()
	prevApproved := annotations["release.deckhouse.io/approved"] == "true"

	// Apply approval via merge patch.
	patch := map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				"release.deckhouse.io/approved": "true",
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshaling approval patch: %w", err)
	}

	_, err = h.client.PatchDeckhouseRelease(ctx, req.GetVersion(), patchBytes)
	if err != nil {
		return nil, fmt.Errorf("patching deckhouse release %s: %w", req.GetVersion(), err)
	}

	return &pb.ApproveReleaseResponse{
		Success:          true,
		PreviousApproved: prevApproved,
	}, nil
}
