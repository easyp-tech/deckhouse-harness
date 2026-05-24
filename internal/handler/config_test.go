package handler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

func TestGetClusterConfiguration_Success(t *testing.T) {
	configYAML := "apiVersion: deckhouse.io/v1\nkind: ClusterConfiguration\n"
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			if namespace != "kube-system" || name != "d8-cluster-configuration" {
				return nil, fmt.Errorf("unexpected secret: %s/%s", namespace, name)
			}
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"cluster-configuration.yaml": []byte(configYAML),
				},
			}, nil
		},
	}

	h := NewConfigHandler(mc)
	resp, err := h.GetClusterConfiguration(context.Background(), &pb.GetClusterConfigurationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Configuration != configYAML {
		t.Errorf("expected configuration YAML, got %q", resp.Configuration)
	}
}

func TestGetClusterConfiguration_SecretNotFound(t *testing.T) {
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
		},
	}

	h := NewConfigHandler(mc)
	_, err := h.GetClusterConfiguration(context.Background(), &pb.GetClusterConfigurationRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetStaticClusterConfiguration_Happy(t *testing.T) {
	staticYAML := "apiVersion: deckhouse.io/v1\nkind: StaticClusterConfiguration\ninternalNetworkCIDRs:\n  - 10.0.0.0/16\n"
	otherYAML := "apiVersion: deckhouse.io/v1\nkind: ClusterConfiguration\n"
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			if namespace != "kube-system" || name != "d8-cluster-configuration" {
				return nil, fmt.Errorf("unexpected secret: %s/%s", namespace, name)
			}
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
				Data: map[string][]byte{
					"cluster-configuration.yaml":        []byte(otherYAML),
					"static-cluster-configuration.yaml": []byte(staticYAML),
				},
			}, nil
		},
	}

	h := NewConfigHandler(mc)
	resp, err := h.GetStaticClusterConfiguration(context.Background(), &pb.GetStaticClusterConfigurationRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Configuration != staticYAML {
		t.Errorf("expected static-cluster-configuration.yaml content, got %q", resp.Configuration)
	}
}

func TestGetStaticClusterConfiguration_KeyMissing(t *testing.T) {
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, _, _ string) (*corev1.Secret, error) {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "d8-cluster-configuration", Namespace: "kube-system"},
				Data: map[string][]byte{
					"cluster-configuration.yaml": []byte("apiVersion: deckhouse.io/v1\n"),
				},
			}, nil
		},
	}

	h := NewConfigHandler(mc)
	_, err := h.GetStaticClusterConfiguration(context.Background(), &pb.GetStaticClusterConfigurationRequest{})
	if err == nil {
		t.Fatal("expected error for missing static-cluster-configuration.yaml key, got nil")
	}
}

func TestGetStaticClusterConfiguration_SecretMissing(t *testing.T) {
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
		},
	}

	h := NewConfigHandler(mc)
	_, err := h.GetStaticClusterConfiguration(context.Background(), &pb.GetStaticClusterConfigurationRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

const baseClusterConfigYAML = `apiVersion: deckhouse.io/v1
kind: ClusterConfiguration
clusterDomain: cluster.local
kubernetesVersion: "1.28"
clusterType: Static
podSubnetCIDR: 10.244.0.0/16
serviceSubnetCIDR: 10.96.0.0/16
`

func makeClusterConfigSecret(yamlContent string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "d8-cluster-configuration",
			Namespace:       "kube-system",
			ResourceVersion: "1",
		},
		Data: map[string][]byte{
			"cluster-configuration.yaml": []byte(yamlContent),
		},
	}
}

func TestUpdateKubernetesVersion_Happy(t *testing.T) {
	var captured *corev1.Secret

	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			if namespace != "kube-system" || name != "d8-cluster-configuration" {
				return nil, fmt.Errorf("unexpected secret: %s/%s", namespace, name)
			}
			return makeClusterConfigSecret(baseClusterConfigYAML), nil
		},
		updateSecretFunc: func(_ context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
			captured = secret
			return secret, nil
		},
	}

	h := NewConfigHandler(mc)

	resp, err := h.UpdateKubernetesVersion(context.Background(), &pb.UpdateKubernetesVersionRequest{Version: "1.29"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetUpdated() {
		t.Error("expected updated=true")
	}
	if resp.GetPreviousVersion() != "1.28" {
		t.Errorf("expected previousVersion=1.28, got %q", resp.GetPreviousVersion())
	}

	if captured == nil {
		t.Fatal("expected UpdateSecret to be called")
	}

	updated := string(captured.Data["cluster-configuration.yaml"])
	if !strings.Contains(updated, `kubernetesVersion: "1.29"`) && !strings.Contains(updated, "kubernetesVersion: 1.29") {
		t.Errorf("expected kubernetesVersion=1.29 in updated YAML, got:\n%s", updated)
	}
	if !strings.Contains(updated, "clusterDomain: cluster.local") {
		t.Errorf("expected clusterDomain preserved in updated YAML, got:\n%s", updated)
	}
}

func TestUpdateKubernetesVersion_SecretMissing(t *testing.T) {
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, namespace, name string) (*corev1.Secret, error) {
			return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
		},
	}

	h := NewConfigHandler(mc)
	_, err := h.UpdateKubernetesVersion(context.Background(), &pb.UpdateKubernetesVersionRequest{Version: "1.29"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateKubernetesVersion_KeyMissing(t *testing.T) {
	mc := &mockClient{
		getSecretFunc: func(_ context.Context, _, _ string) (*corev1.Secret, error) {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "d8-cluster-configuration", Namespace: "kube-system"},
				Data: map[string][]byte{
					"static-cluster-configuration.yaml": []byte("apiVersion: deckhouse.io/v1\n"),
				},
			}, nil
		},
	}

	h := NewConfigHandler(mc)
	_, err := h.UpdateKubernetesVersion(context.Background(), &pb.UpdateKubernetesVersionRequest{Version: "1.29"})
	if err == nil {
		t.Fatal("expected error for missing cluster-configuration.yaml key, got nil")
	}
}

func TestUpdateKubernetesVersion_RetryOnConflict(t *testing.T) {
	var updateCalls int

	mc := &mockClient{
		getSecretFunc: func(_ context.Context, _, _ string) (*corev1.Secret, error) {
			return makeClusterConfigSecret(baseClusterConfigYAML), nil
		},
		updateSecretFunc: func(_ context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
			updateCalls++
			if updateCalls == 1 {
				return nil, errors.NewConflict(
					schema.GroupResource{Resource: "secrets"},
					secret.Name,
					fmt.Errorf("the object has been modified"),
				)
			}
			return secret, nil
		},
	}

	h := NewConfigHandler(mc)
	resp, err := h.UpdateKubernetesVersion(context.Background(), &pb.UpdateKubernetesVersionRequest{Version: "1.29"})
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if !resp.GetUpdated() {
		t.Error("expected updated=true after successful retry")
	}
	if updateCalls < 2 {
		t.Errorf("expected at least 2 UpdateSecret calls (retry on conflict), got %d", updateCalls)
	}
}
