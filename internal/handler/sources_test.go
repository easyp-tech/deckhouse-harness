package handler

import (
	"context"
	stderrors "errors"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

// makeModuleSource builds a synthetic ModuleSource fixture.
func makeModuleSource(name, registry, statusMessage string) unstructured.Unstructured {
	obj := unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "deckhouse.io/v1alpha1",
		"kind":       "ModuleSource",
		"metadata":   map[string]any{"name": name},
		"spec": map[string]any{
			"registry": map[string]any{"repo": registry},
		},
		"status": map[string]any{"message": statusMessage},
	}}

	return obj
}

// makeModuleUpdatePolicy builds a synthetic ModuleUpdatePolicy fixture.
func makeModuleUpdatePolicy(name, mode string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "deckhouse.io/v1alpha1",
		"kind":       "ModuleUpdatePolicy",
		"metadata":   map[string]any{"name": name},
		"spec": map[string]any{
			"update": map[string]any{"mode": mode},
		},
	}}
}

func TestListModuleSources_Empty(t *testing.T) {
	mc := &mockClient{
		listModuleSourcesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleSources(context.Background(), &pb.ListModuleSourcesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSources() == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(resp.GetSources()) != 0 {
		t.Errorf("expected empty list, got %d sources", len(resp.GetSources()))
	}
}

func TestListModuleSources_Happy(t *testing.T) {
	mc := &mockClient{
		listModuleSourcesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleSource("deckhouse", "registry.deckhouse.io/deckhouse/ce/modules", "Synced"),
				makeModuleSource("custom", "registry.example.com/modules", ""),
			}, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleSources(context.Background(), &pb.ListModuleSourcesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetSources()) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(resp.GetSources()))
	}

	first := resp.GetSources()[0]
	if first.GetName() != "deckhouse" {
		t.Errorf("expected name=deckhouse, got %q", first.GetName())
	}
	if first.GetRegistry() != "registry.deckhouse.io/deckhouse/ce/modules" {
		t.Errorf("expected registry from spec.registry.repo, got %q", first.GetRegistry())
	}
	if first.GetStatus() != "Synced" {
		t.Errorf("expected status=Synced, got %q", first.GetStatus())
	}
}

func TestCreateModuleSource_Happy(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createModuleSourceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj

			return obj, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.CreateModuleSource(context.Background(), &pb.CreateModuleSourceRequest{
		Name:     "custom-modules",
		Registry: "registry.example.com/modules",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetCreated() {
		t.Error("expected created=true")
	}
	if resp.GetName() != "custom-modules" {
		t.Errorf("expected echoed name=custom-modules, got %q", resp.GetName())
	}

	if captured == nil {
		t.Fatal("expected CreateModuleSource to be called")
	}
	if captured.GetKind() != "ModuleSource" {
		t.Errorf("expected kind=ModuleSource, got %q", captured.GetKind())
	}
	if captured.GetAPIVersion() != "deckhouse.io/v1alpha1" {
		t.Errorf("expected apiVersion=deckhouse.io/v1alpha1, got %q", captured.GetAPIVersion())
	}
	if captured.GetName() != "custom-modules" {
		t.Errorf("expected metadata.name=custom-modules, got %q", captured.GetName())
	}

	spec, ok := captured.Object["spec"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec object, got %#v", captured.Object["spec"])
	}
	registry, ok := spec["registry"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec.registry object, got %#v", spec["registry"])
	}
	if got := registry["repo"]; got != "registry.example.com/modules" {
		t.Errorf("expected spec.registry.repo=registry.example.com/modules, got %v", got)
	}
}

func TestCreateModuleSource_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		createModuleSourceFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, errors.NewAlreadyExists(
				schema.GroupResource{Group: "deckhouse.io", Resource: "modulesources"},
				obj.GetName(),
			)
		},
	}

	h := NewSourcesHandler(mc)
	_, err := h.CreateModuleSource(context.Background(), &pb.CreateModuleSourceRequest{
		Name:     "duplicate",
		Registry: "registry.example.com/modules",
	})
	if err == nil {
		t.Fatal("expected already-exists error, got nil")
	}
}

func TestListModuleUpdatePolicies_Empty(t *testing.T) {
	mc := &mockClient{
		listModuleUpdatePoliciesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleUpdatePolicies(context.Background(), &pb.ListModuleUpdatePoliciesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetPolicies() == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(resp.GetPolicies()) != 0 {
		t.Errorf("expected empty list, got %d policies", len(resp.GetPolicies()))
	}
}

func TestListModuleUpdatePolicies_Happy(t *testing.T) {
	mc := &mockClient{
		listModuleUpdatePoliciesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleUpdatePolicy("auto", "Auto"),
				makeModuleUpdatePolicy("manual", "Manual"),
			}, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleUpdatePolicies(context.Background(), &pb.ListModuleUpdatePoliciesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetPolicies()) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(resp.GetPolicies()))
	}

	first := resp.GetPolicies()[0]
	if first.GetName() != "auto" {
		t.Errorf("expected name=auto, got %q", first.GetName())
	}
	if first.GetUpdateMode() != "Auto" {
		t.Errorf("expected updateMode=Auto, got %q", first.GetUpdateMode())
	}
}

func TestCreateModuleUpdatePolicy_Happy(t *testing.T) {
	var captured *unstructured.Unstructured
	mc := &mockClient{
		createModuleUpdatePolicyFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj

			return obj, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.CreateModuleUpdatePolicy(context.Background(), &pb.CreateModuleUpdatePolicyRequest{
		Name:        "auto",
		UpdateMode:  "Auto",
		MatchLabels: map[string]string{"module": "cert-manager"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetCreated() {
		t.Error("expected created=true")
	}
	if resp.GetName() != "auto" {
		t.Errorf("expected echoed name=auto, got %q", resp.GetName())
	}

	if captured == nil {
		t.Fatal("expected CreateModuleUpdatePolicy to be called")
	}
	if captured.GetKind() != "ModuleUpdatePolicy" {
		t.Errorf("expected kind=ModuleUpdatePolicy, got %q", captured.GetKind())
	}
	if captured.GetAPIVersion() != "deckhouse.io/v1alpha1" {
		t.Errorf("expected apiVersion=deckhouse.io/v1alpha1, got %q", captured.GetAPIVersion())
	}
	if captured.GetName() != "auto" {
		t.Errorf("expected metadata.name=auto, got %q", captured.GetName())
	}

	spec, ok := captured.Object["spec"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec object, got %#v", captured.Object["spec"])
	}
	update, ok := spec["update"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec.update object, got %#v", spec["update"])
	}
	if got := update["mode"]; got != "Auto" {
		t.Errorf("expected spec.update.mode=Auto, got %v", got)
	}

	selector, ok := spec["moduleReleaseSelector"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec.moduleReleaseSelector object, got %#v", spec["moduleReleaseSelector"])
	}
	labelSelector, ok := selector["labelSelector"].(map[string]any)
	if !ok {
		t.Fatalf("expected spec.moduleReleaseSelector.labelSelector object, got %#v", selector["labelSelector"])
	}
	matchLabels, ok := labelSelector["matchLabels"].(map[string]any)
	if !ok {
		t.Fatalf("expected matchLabels object, got %#v", labelSelector["matchLabels"])
	}
	if got := matchLabels["module"]; got != "cert-manager" {
		t.Errorf("expected matchLabels[module]=cert-manager, got %v", got)
	}
}

func TestCreateModuleUpdatePolicy_MissingMatchLabels(t *testing.T) {
	var called bool
	mc := &mockClient{
		createModuleUpdatePolicyFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			called = true

			return nil, nil
		},
	}

	h := NewSourcesHandler(mc)
	_, err := h.CreateModuleUpdatePolicy(context.Background(), &pb.CreateModuleUpdatePolicyRequest{
		Name:       "no-selector",
		UpdateMode: "Auto",
	})
	if err == nil {
		t.Fatal("expected error for missing match_labels, got nil")
	}
	if !stderrors.Is(err, errMatchLabelsRequired) {
		t.Errorf("expected errMatchLabelsRequired, got %v", err)
	}
	if called {
		t.Error("expected k8s client not to be called when validation fails")
	}
}

func TestCreateModuleUpdatePolicy_AlreadyExists(t *testing.T) {
	mc := &mockClient{
		createModuleUpdatePolicyFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return nil, errors.NewAlreadyExists(
				schema.GroupResource{Group: "deckhouse.io", Resource: "moduleupdatepolicies"},
				obj.GetName(),
			)
		},
	}

	h := NewSourcesHandler(mc)
	_, err := h.CreateModuleUpdatePolicy(context.Background(), &pb.CreateModuleUpdatePolicyRequest{
		Name:        "duplicate",
		UpdateMode:  "Auto",
		MatchLabels: map[string]string{"module": "cert-manager"},
	})
	if err == nil {
		t.Fatal("expected already-exists error, got nil")
	}
}

// makeModuleRelease builds a synthetic ModuleRelease fixture with labels.
func makeModuleRelease(name, module, source, version, phase, approved string) unstructured.Unstructured {
	spec := map[string]any{
		"version": version,
	}
	if approved != "" {
		spec["approved"] = approved == "true"
	}

	return unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "deckhouse.io/v1alpha1",
		"kind":       "ModuleRelease",
		"metadata": map[string]any{
			"name": name,
			"labels": map[string]any{
				"module": module,
				"source": source,
			},
		},
		"spec":   spec,
		"status": map[string]any{"phase": phase},
	}}
}

// strPtr is a tiny helper for taking address of a string literal in tests.
func strPtr(s string) *string { return &s }

func TestSourcesHandler_ListModuleReleases_Success(t *testing.T) {
	var capturedModule string
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, moduleName string) ([]unstructured.Unstructured, error) {
			capturedModule = moduleName

			return []unstructured.Unstructured{
				makeModuleRelease("deckhouse-1.70.0", "deckhouse", "deckhouse", "1.70.0", "Deployed", "true"),
				makeModuleRelease("deckhouse-1.71.0", "deckhouse", "deckhouse", "1.71.0", "Pending", ""),
			}, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleReleases(context.Background(), &pb.ListModuleReleasesRequest{
		ModuleName: "deckhouse",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedModule != "deckhouse" {
		t.Errorf("expected client called with module_name=deckhouse, got %q", capturedModule)
	}
	if got := len(resp.GetReleases()); got != 2 {
		t.Fatalf("expected 2 releases, got %d", got)
	}

	first := resp.GetReleases()[0]
	if first.GetName() != "deckhouse-1.70.0" {
		t.Errorf("expected name=deckhouse-1.70.0, got %q", first.GetName())
	}
	if first.GetModule() != "deckhouse" {
		t.Errorf("expected module=deckhouse, got %q", first.GetModule())
	}
	if first.GetVersion() != "1.70.0" {
		t.Errorf("expected version=1.70.0, got %q", first.GetVersion())
	}
	if first.GetSource() != "deckhouse" {
		t.Errorf("expected source=deckhouse, got %q", first.GetSource())
	}
	if first.GetPhase() != "Deployed" {
		t.Errorf("expected phase=Deployed, got %q", first.GetPhase())
	}
	if first.GetApproved() != "true" {
		t.Errorf("expected approved=true, got %q", first.GetApproved())
	}
}

func TestSourcesHandler_ListModuleReleases_PhaseFilter(t *testing.T) {
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleRelease("deckhouse-1.70.0", "deckhouse", "deckhouse", "1.70.0", "Deployed", "true"),
				makeModuleRelease("deckhouse-1.71.0", "deckhouse", "deckhouse", "1.71.0", "Pending", ""),
				makeModuleRelease("deckhouse-1.69.0", "deckhouse", "deckhouse", "1.69.0", "Superseded", "true"),
			}, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleReleases(context.Background(), &pb.ListModuleReleasesRequest{
		ModuleName: "deckhouse",
		Phase:      strPtr("Pending"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.GetReleases()); got != 1 {
		t.Fatalf("expected 1 release after phase filter, got %d", got)
	}
	if got := resp.GetReleases()[0].GetPhase(); got != "Pending" {
		t.Errorf("expected phase=Pending, got %q", got)
	}
	if got := resp.GetReleases()[0].GetName(); got != "deckhouse-1.71.0" {
		t.Errorf("expected deckhouse-1.71.0 (only Pending one), got %q", got)
	}
}

func TestSourcesHandler_ListModuleReleases_Empty(t *testing.T) {
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
			return nil, nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.ListModuleReleases(context.Background(), &pb.ListModuleReleasesRequest{
		ModuleName: "absent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetReleases() == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if got := len(resp.GetReleases()); got != 0 {
		t.Errorf("expected empty list, got %d", got)
	}
}

func TestSourcesHandler_ListModuleReleases_EmptyModuleName(t *testing.T) {
	called := false
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
			called = true

			return nil, nil
		},
	}

	h := NewSourcesHandler(mc)
	_, err := h.ListModuleReleases(context.Background(), &pb.ListModuleReleasesRequest{
		ModuleName: "",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if called {
		t.Error("expected k8s.Client to NOT be called when module_name is empty")
	}
	if err.Error() != "module_name is required" {
		t.Errorf("expected error %q, got %q", "module_name is required", err.Error())
	}
}

// boolPtr is a tiny helper for taking address of a bool literal in tests.
func boolPtr(b bool) *bool { return &b }

func TestSourcesHandler_DeleteModuleSource_NoActiveReleases(t *testing.T) {
	listCalls := 0
	deleteCalled := false
	deletedName := ""
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, moduleName string) ([]unstructured.Unstructured, error) {
			listCalls++
			if moduleName != "" {
				t.Errorf("expected empty moduleName for source-based pre-check, got %q", moduleName)
			}

			return nil, nil
		},
		deleteModuleSourceFunc: func(_ context.Context, name string) error {
			deleteCalled = true
			deletedName = name

			return nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.DeleteModuleSource(context.Background(), &pb.DeleteModuleSourceRequest{
		Name: "custom-modules",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCalls != 1 {
		t.Errorf("expected pre-check ListModuleReleases call count 1, got %d", listCalls)
	}
	if !deleteCalled {
		t.Error("expected DeleteModuleSource to be invoked")
	}
	if deletedName != "custom-modules" {
		t.Errorf("expected delete name=custom-modules, got %q", deletedName)
	}
	if !resp.GetDeleted() {
		t.Error("expected deleted=true")
	}
}

func TestSourcesHandler_DeleteModuleSource_BlockedByActiveReleases(t *testing.T) {
	deleteCalled := false
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleRelease("custom-1.0.0", "custom", "custom-modules", "1.0.0", "Deployed", "true"),
				makeModuleRelease("other-1.0.0", "other", "other-source", "1.0.0", "Deployed", "true"),
			}, nil
		},
		deleteModuleSourceFunc: func(_ context.Context, _ string) error {
			deleteCalled = true

			return nil
		},
	}

	h := NewSourcesHandler(mc)
	_, err := h.DeleteModuleSource(context.Background(), &pb.DeleteModuleSourceRequest{
		Name: "custom-modules",
	})
	if err == nil {
		t.Fatal("expected error blocking deletion, got nil")
	}
	if deleteCalled {
		t.Error("expected DeleteModuleSource NOT to be invoked when active releases exist")
	}
}

func TestSourcesHandler_DeleteModuleSource_ForceSkipsPreCheck(t *testing.T) {
	listCalls := 0
	deleteCalled := false
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
			listCalls++

			return []unstructured.Unstructured{
				makeModuleRelease("custom-1.0.0", "custom", "custom-modules", "1.0.0", "Deployed", "true"),
			}, nil
		},
		deleteModuleSourceFunc: func(_ context.Context, name string) error {
			deleteCalled = true
			if name != "custom-modules" {
				t.Errorf("expected name=custom-modules, got %q", name)
			}

			return nil
		},
	}

	h := NewSourcesHandler(mc)
	resp, err := h.DeleteModuleSource(context.Background(), &pb.DeleteModuleSourceRequest{
		Name:  "custom-modules",
		Force: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("unexpected error with force=true: %v", err)
	}
	if listCalls != 0 {
		t.Errorf("expected NO ListModuleReleases call when force=true, got %d calls", listCalls)
	}
	if !deleteCalled {
		t.Error("expected DeleteModuleSource to be invoked when force=true")
	}
	if !resp.GetDeleted() {
		t.Error("expected deleted=true")
	}
}

func TestSourcesHandler_DeleteModuleSource_NotFound(t *testing.T) {
	mc := &mockClient{
		listModuleReleasesFunc: func(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
			return nil, nil
		},
		deleteModuleSourceFunc: func(_ context.Context, name string) error {
			return errors.NewNotFound(
				schema.GroupResource{Group: "deckhouse.io", Resource: "modulesources"},
				name,
			)
		},
	}

	h := NewSourcesHandler(mc)
	_, err := h.DeleteModuleSource(context.Background(), &pb.DeleteModuleSourceRequest{
		Name: "missing",
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}
