package handler

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"

	"github.com/easyp-tech/deckhouse-harness/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

const (
	clusterConfigNamespace = "kube-system"
	clusterConfigSecret    = "d8-cluster-configuration"
	clusterConfigKey       = "cluster-configuration.yaml"
	updateConflictRetries  = 3
)

// ConfigHandler implements pb.ConfigAPIToolHandler.
type ConfigHandler struct {
	client k8s.Client
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(client k8s.Client) *ConfigHandler {
	return &ConfigHandler{client: client}
}

// GetClusterConfiguration reads the ClusterConfiguration YAML from the d8-cluster-configuration Secret.
func (h *ConfigHandler) GetClusterConfiguration(ctx context.Context, _ *pb.GetClusterConfigurationRequest) (*pb.GetClusterConfigurationResponse, error) {
	secret, err := h.client.GetSecret(ctx, "kube-system", "d8-cluster-configuration")
	if err != nil {
		return nil, fmt.Errorf("cluster configuration secret not found: %w", err)
	}

	config, ok := secret.Data["cluster-configuration.yaml"]
	if !ok {
		return nil, fmt.Errorf("key cluster-configuration.yaml not found in d8-cluster-configuration secret")
	}

	return &pb.GetClusterConfigurationResponse{
		Configuration: string(config),
	}, nil
}

// GetStaticClusterConfiguration reads the StaticClusterConfiguration YAML from the
// d8-cluster-configuration Secret (key static-cluster-configuration.yaml).
func (h *ConfigHandler) GetStaticClusterConfiguration(
	ctx context.Context,
	_ *pb.GetStaticClusterConfigurationRequest,
) (*pb.GetStaticClusterConfigurationResponse, error) {
	secret, err := h.client.GetSecret(ctx, "kube-system", "d8-cluster-configuration")
	if err != nil {
		return nil, fmt.Errorf("cluster configuration secret not found: %w", err)
	}

	config, ok := secret.Data["static-cluster-configuration.yaml"]
	if !ok {
		return nil, fmt.Errorf("key static-cluster-configuration.yaml not found in d8-cluster-configuration secret")
	}

	return &pb.GetStaticClusterConfigurationResponse{
		Configuration: string(config),
	}, nil
}

// UpdateKubernetesVersion patches the kubernetesVersion field inside the
// ClusterConfiguration YAML stored in d8-cluster-configuration Secret. The
// Secret is updated using read-modify-write with up to 3 retries on conflict.
//
// The YAML round-trip preserves all other fields verbatim (no key reordering
// or comment loss beyond what sigs.k8s.io/yaml does — which is a single-pass
// JSON-backed encoder).
func (h *ConfigHandler) UpdateKubernetesVersion(
	ctx context.Context,
	req *pb.UpdateKubernetesVersionRequest,
) (*pb.UpdateKubernetesVersionResponse, error) {
	var previousVersion string

	for attempt := 0; attempt < updateConflictRetries; attempt++ {
		secret, err := h.client.GetSecret(ctx, clusterConfigNamespace, clusterConfigSecret)
		if err != nil {
			return nil, fmt.Errorf("getting %s secret: %w", clusterConfigSecret, err)
		}

		raw, ok := secret.Data[clusterConfigKey]
		if !ok {
			return nil, fmt.Errorf("key %s not found in %s secret", clusterConfigKey, clusterConfigSecret)
		}

		var doc map[string]any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", clusterConfigKey, err)
		}

		if existing, hasField := doc["kubernetesVersion"].(string); hasField {
			previousVersion = existing
		}

		doc["kubernetesVersion"] = req.GetVersion()

		updated, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("re-marshalling %s: %w", clusterConfigKey, err)
		}

		secret.Data[clusterConfigKey] = updated

		_, err = h.client.UpdateSecret(ctx, secret)
		switch {
		case err == nil:
			return &pb.UpdateKubernetesVersionResponse{
				Updated:         true,
				PreviousVersion: previousVersion,
			}, nil
		case kerrors.IsConflict(err):
			// Optimistic concurrency failure — re-fetch and try again.
			continue
		default:
			return nil, fmt.Errorf("updating %s secret: %w", clusterConfigSecret, err)
		}
	}

	return nil, fmt.Errorf("updating %s secret: exhausted %d retries on conflict", clusterConfigSecret, updateConflictRetries)
}
