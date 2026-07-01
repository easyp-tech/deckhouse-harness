package handler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/easyp-tech/deckhouse-harness/internal/k8s"
)

// Verify mockClient implements k8s.Client.
var _ k8s.Client = (*mockClient)(nil)

// mockClient is a test double for k8s.Client.
type mockClient struct {
	listNodesFunc             func(ctx context.Context) ([]corev1.Node, error)
	getNodeFunc               func(ctx context.Context, name string) (*corev1.Node, error)
	cordonNodeFunc            func(ctx context.Context, name string) error
	listPodsFunc              func(ctx context.Context, namespace string) ([]corev1.Pod, error)
	deletePodFunc             func(ctx context.Context, namespace, name string) error
	listNodeEventsFunc        func(ctx context.Context, nodeName string) ([]corev1.Event, error)
	getPodLogsFunc            func(ctx context.Context, namespace, pod, container string, tail *int64, since *string) (string, error)
	getSecretFunc             func(ctx context.Context, namespace, name string) (*corev1.Secret, error)
	listNodeGroupsFunc        func(ctx context.Context) ([]unstructured.Unstructured, error)
	getNodeGroupFunc          func(ctx context.Context, name string) (*unstructured.Unstructured, error)
	createNodeGroupFunc       func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	listStaticInstancesFunc   func(ctx context.Context) ([]unstructured.Unstructured, error)
	getStaticInstanceFunc     func(ctx context.Context, name string) (*unstructured.Unstructured, error)
	createStaticInstanceFunc  func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	deleteStaticInstanceFunc  func(ctx context.Context, name string) error
	listModuleConfigsFunc     func(ctx context.Context) ([]unstructured.Unstructured, error)
	getModuleConfigFunc       func(ctx context.Context, name string) (*unstructured.Unstructured, error)
	updateModuleConfigFunc    func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	listDeckhouseReleasesFunc func(ctx context.Context) ([]unstructured.Unstructured, error)
	getDeckhouseReleaseFunc   func(ctx context.Context, name string) (*unstructured.Unstructured, error)
	patchDeckhouseReleaseFunc func(ctx context.Context, name string, patch []byte) (*unstructured.Unstructured, error)
	createSSHCredentialsFunc  func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	deleteSSHCredentialsFunc  func(ctx context.Context, name string) error
	listModulesFunc           func(ctx context.Context) ([]unstructured.Unstructured, error)
	deleteNodeGroupFunc       func(ctx context.Context, name string) error
	uncordonNodeFunc          func(ctx context.Context, name string) error
	evictPodFunc              func(ctx context.Context, namespace, name string) error
	updateSecretFunc          func(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error)

	listModuleSourcesFunc        func(ctx context.Context) ([]unstructured.Unstructured, error)
	createModuleSourceFunc       func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	deleteModuleSourceFunc       func(ctx context.Context, name string) error
	listModuleUpdatePoliciesFunc func(ctx context.Context) ([]unstructured.Unstructured, error)
	createModuleUpdatePolicyFunc func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)

	listModuleReleasesFunc           func(ctx context.Context, moduleName string) ([]unstructured.Unstructured, error)
	createNodeGroupConfigurationFunc func(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error)
	patchModuleConfigFunc            func(ctx context.Context, name string, patch []byte) (*unstructured.Unstructured, error)
}

func (m *mockClient) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	if m.listNodesFunc != nil {
		return m.listNodesFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	if m.getNodeFunc != nil {
		return m.getNodeFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockClient) CordonNode(ctx context.Context, name string) error {
	if m.cordonNodeFunc != nil {
		return m.cordonNodeFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	if m.listPodsFunc != nil {
		return m.listPodsFunc(ctx, namespace)
	}
	return nil, nil
}

func (m *mockClient) DeletePod(ctx context.Context, namespace, name string) error {
	if m.deletePodFunc != nil {
		return m.deletePodFunc(ctx, namespace, name)
	}
	return nil
}

func (m *mockClient) ListNodeEvents(ctx context.Context, nodeName string) ([]corev1.Event, error) {
	if m.listNodeEventsFunc != nil {
		return m.listNodeEventsFunc(ctx, nodeName)
	}
	return nil, nil
}

func (m *mockClient) GetPodLogs(ctx context.Context, namespace, pod, container string, tail *int64, since *string) (string, error) {
	if m.getPodLogsFunc != nil {
		return m.getPodLogsFunc(ctx, namespace, pod, container, tail, since)
	}
	return "", nil
}

func (m *mockClient) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	if m.getSecretFunc != nil {
		return m.getSecretFunc(ctx, namespace, name)
	}
	return nil, nil
}

func (m *mockClient) ListNodeGroups(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listNodeGroupsFunc != nil {
		return m.listNodeGroupsFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) GetNodeGroup(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if m.getNodeGroupFunc != nil {
		return m.getNodeGroupFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockClient) CreateNodeGroup(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.createNodeGroupFunc != nil {
		return m.createNodeGroupFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) ListStaticInstances(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listStaticInstancesFunc != nil {
		return m.listStaticInstancesFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) GetStaticInstance(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if m.getStaticInstanceFunc != nil {
		return m.getStaticInstanceFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockClient) CreateStaticInstance(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.createStaticInstanceFunc != nil {
		return m.createStaticInstanceFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) DeleteStaticInstance(ctx context.Context, name string) error {
	if m.deleteStaticInstanceFunc != nil {
		return m.deleteStaticInstanceFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) ListModuleConfigs(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listModuleConfigsFunc != nil {
		return m.listModuleConfigsFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) GetModuleConfig(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if m.getModuleConfigFunc != nil {
		return m.getModuleConfigFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockClient) UpdateModuleConfig(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.updateModuleConfigFunc != nil {
		return m.updateModuleConfigFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) ListDeckhouseReleases(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listDeckhouseReleasesFunc != nil {
		return m.listDeckhouseReleasesFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) GetDeckhouseRelease(ctx context.Context, name string) (*unstructured.Unstructured, error) {
	if m.getDeckhouseReleaseFunc != nil {
		return m.getDeckhouseReleaseFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockClient) PatchDeckhouseRelease(ctx context.Context, name string, patch []byte) (*unstructured.Unstructured, error) {
	if m.patchDeckhouseReleaseFunc != nil {
		return m.patchDeckhouseReleaseFunc(ctx, name, patch)
	}
	return nil, nil
}

func (m *mockClient) CreateSSHCredentials(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.createSSHCredentialsFunc != nil {
		return m.createSSHCredentialsFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) ListModules(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listModulesFunc != nil {
		return m.listModulesFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) UncordonNode(ctx context.Context, name string) error {
	if m.uncordonNodeFunc != nil {
		return m.uncordonNodeFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) EvictPod(ctx context.Context, namespace, name string) error {
	if m.evictPodFunc != nil {
		return m.evictPodFunc(ctx, namespace, name)
	}
	return nil
}

func (m *mockClient) UpdateSecret(ctx context.Context, secret *corev1.Secret) (*corev1.Secret, error) {
	if m.updateSecretFunc != nil {
		return m.updateSecretFunc(ctx, secret)
	}
	return secret, nil
}

func (m *mockClient) DeleteSSHCredentials(ctx context.Context, name string) error {
	if m.deleteSSHCredentialsFunc != nil {
		return m.deleteSSHCredentialsFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) DeleteNodeGroup(ctx context.Context, name string) error {
	if m.deleteNodeGroupFunc != nil {
		return m.deleteNodeGroupFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) ListModuleSources(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listModuleSourcesFunc != nil {
		return m.listModuleSourcesFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) CreateModuleSource(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.createModuleSourceFunc != nil {
		return m.createModuleSourceFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) ListModuleUpdatePolicies(ctx context.Context) ([]unstructured.Unstructured, error) {
	if m.listModuleUpdatePoliciesFunc != nil {
		return m.listModuleUpdatePoliciesFunc(ctx)
	}
	return nil, nil
}

func (m *mockClient) CreateModuleUpdatePolicy(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.createModuleUpdatePolicyFunc != nil {
		return m.createModuleUpdatePolicyFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) DeleteModuleSource(ctx context.Context, name string) error {
	if m.deleteModuleSourceFunc != nil {
		return m.deleteModuleSourceFunc(ctx, name)
	}
	return nil
}

func (m *mockClient) ListModuleReleases(ctx context.Context, moduleName string) ([]unstructured.Unstructured, error) {
	if m.listModuleReleasesFunc != nil {
		return m.listModuleReleasesFunc(ctx, moduleName)
	}
	return nil, nil
}

func (m *mockClient) CreateNodeGroupConfiguration(ctx context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if m.createNodeGroupConfigurationFunc != nil {
		return m.createNodeGroupConfigurationFunc(ctx, obj)
	}
	return obj, nil
}

func (m *mockClient) PatchModuleConfig(ctx context.Context, name string, patch []byte) (*unstructured.Unstructured, error) {
	if m.patchModuleConfigFunc != nil {
		return m.patchModuleConfigFunc(ctx, name, patch)
	}
	return nil, nil
}
