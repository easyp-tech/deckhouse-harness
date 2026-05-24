package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

func TestListModuleConfigs_All(t *testing.T) {
	mc := &mockClient{
		listModuleConfigsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleConfig("mod-a", true, ""),
				makeModuleConfig("mod-b", false, ""),
			}, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.ListModuleConfigs(context.Background(), &pb.ListModuleConfigsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Modules) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Modules))
	}
}

func TestListModuleConfigs_Enabled(t *testing.T) {
	mc := &mockClient{
		listModuleConfigsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleConfig("mod-a", true, ""),
				makeModuleConfig("mod-b", false, ""),
				makeModuleConfig("mod-c", true, ""),
			}, nil
		},
	}

	h := NewModulesHandler(mc)
	enabled := true
	resp, err := h.ListModuleConfigs(context.Background(), &pb.ListModuleConfigsRequest{Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Modules) != 2 {
		t.Errorf("expected 2, got %d", len(resp.Modules))
	}
	for _, m := range resp.Modules {
		if !m.Enabled {
			t.Errorf("expected enabled=true for %s", m.Name)
		}
	}
}

func TestListModuleConfigs_Disabled(t *testing.T) {
	mc := &mockClient{
		listModuleConfigsFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModuleConfig("mod-a", true, ""),
				makeModuleConfig("mod-b", false, ""),
			}, nil
		},
	}

	h := NewModulesHandler(mc)
	enabled := false
	resp, err := h.ListModuleConfigs(context.Background(), &pb.ListModuleConfigsRequest{Enabled: &enabled})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Modules) != 1 {
		t.Errorf("expected 1, got %d", len(resp.Modules))
	}
	if resp.Modules[0].Name != "mod-b" {
		t.Errorf("expected mod-b, got %s", resp.Modules[0].Name)
	}
}

func TestGetModuleConfig_Found(t *testing.T) {
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			obj := makeModuleConfig(name, true, "all good")
			return &obj, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.GetModuleConfig(context.Background(), &pb.GetModuleConfigRequest{Name: "cert-manager"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "cert-manager" {
		t.Errorf("expected cert-manager, got %s", resp.Name)
	}
	if !resp.Enabled {
		t.Error("expected enabled=true")
	}
}

func TestGetModuleConfig_NotFound(t *testing.T) {
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("module config %q not found", name)
		},
	}

	h := NewModulesHandler(mc)
	_, err := h.GetModuleConfig(context.Background(), &pb.GetModuleConfigRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnableModule_WasDisabled(t *testing.T) {
	existing := makeModuleConfig("mod-a", false, "")
	updated := makeModuleConfig("mod-a", true, "")
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &existing, nil
		},
		updateModuleConfigFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return &updated, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.EnableModule(context.Background(), &pb.EnableModuleRequest{Name: "mod-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.PreviousState {
		t.Error("expected previousState=false")
	}
}

func TestEnableModule_AlreadyEnabled(t *testing.T) {
	existing := makeModuleConfig("mod-a", true, "")
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &existing, nil
		},
		updateModuleConfigFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return obj, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.EnableModule(context.Background(), &pb.EnableModuleRequest{Name: "mod-a"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if !resp.PreviousState {
		t.Error("expected previousState=true")
	}
}

func TestEnableModule_NotFound(t *testing.T) {
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("module config %q not found", name)
		},
	}

	h := NewModulesHandler(mc)
	_, err := h.EnableModule(context.Background(), &pb.EnableModuleRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDisableModule_WasEnabled(t *testing.T) {
	existing := makeModuleConfig("mod-b", true, "")
	updated := makeModuleConfig("mod-b", false, "")
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			return &existing, nil
		},
		updateModuleConfigFunc: func(_ context.Context, _ *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			return &updated, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.DisableModule(context.Background(), &pb.DisableModuleRequest{Name: "mod-b"})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if !resp.PreviousState {
		t.Error("expected previousState=true (was enabled)")
	}
}

func TestDisableModule_NotFound(t *testing.T) {
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("module config %q not found", name)
		},
	}

	h := NewModulesHandler(mc)
	_, err := h.DisableModule(context.Background(), &pb.DisableModuleRequest{Name: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListModules_Happy(t *testing.T) {
	mc := &mockClient{
		listModulesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return []unstructured.Unstructured{
				makeModule("cert-manager", 50, "deckhouse", "Enabled"),
				makeModule("custom-module", 100, "third-party", "Disabled"),
			}, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.ListModules(context.Background(), &pb.ListModulesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(resp.Modules))
	}
	if resp.Modules[0].Name != "cert-manager" {
		t.Errorf("expected first name cert-manager, got %q", resp.Modules[0].Name)
	}
	if resp.Modules[0].Weight != 50 {
		t.Errorf("expected weight=50, got %d", resp.Modules[0].Weight)
	}
	if resp.Modules[0].Source != "deckhouse" {
		t.Errorf("expected source=deckhouse, got %q", resp.Modules[0].Source)
	}
	if resp.Modules[0].State != "Enabled" {
		t.Errorf("expected state=Enabled, got %q", resp.Modules[0].State)
	}
	if resp.Modules[1].State != "Disabled" {
		t.Errorf("expected state=Disabled for second module, got %q", resp.Modules[1].State)
	}
}

func TestListModules_Empty(t *testing.T) {
	mc := &mockClient{
		listModulesFunc: func(_ context.Context) ([]unstructured.Unstructured, error) {
			return nil, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.ListModules(context.Background(), &pb.ListModulesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Modules) != 0 {
		t.Errorf("expected 0, got %d", len(resp.Modules))
	}
}

func TestUpdateModuleSettings_Happy(t *testing.T) {
	existing := makeModuleConfigWithSettings("mod-a", map[string]any{
		"a": float64(1),
		"nested": map[string]any{
			"b": float64(2),
			"c": float64(3),
		},
	})

	var captured *unstructured.Unstructured

	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			obj := existing
			return &obj, nil
		},
		updateModuleConfigFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj
			return obj, nil
		},
	}

	settings, err := structpb.NewStruct(map[string]any{
		"nested": map[string]any{"b": float64(99)},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	h := NewModulesHandler(mc)

	resp, err := h.UpdateModuleSettings(context.Background(), &pb.UpdateModuleSettingsRequest{
		Name:     "mod-a",
		Settings: settings,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.GetUpdated() {
		t.Error("expected updated=true")
	}

	if captured == nil {
		t.Fatal("expected UpdateModuleConfig to be called")
	}

	merged, _, err := unstructured.NestedMap(captured.Object, "spec", "settings")
	if err != nil {
		t.Fatalf("reading merged settings: %v", err)
	}

	if merged["a"] != float64(1) {
		t.Errorf("expected a=1 preserved, got %v", merged["a"])
	}

	nested, ok := merged["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested is not a map: %T", merged["nested"])
	}

	if nested["b"] != float64(99) {
		t.Errorf("expected nested.b=99, got %v", nested["b"])
	}

	if nested["c"] != float64(3) {
		t.Errorf("expected nested.c=3 preserved, got %v", nested["c"])
	}
}

func TestUpdateModuleSettings_NullRemoves(t *testing.T) {
	existing := makeModuleConfigWithSettings("mod-a", map[string]any{
		"a": float64(1),
		"nested": map[string]any{
			"b": float64(2),
			"c": float64(3),
		},
	})

	var captured *unstructured.Unstructured

	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			obj := existing
			return &obj, nil
		},
		updateModuleConfigFunc: func(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			captured = obj
			return obj, nil
		},
	}

	settings, err := structpb.NewStruct(map[string]any{
		"nested": map[string]any{"b": nil},
	})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	h := NewModulesHandler(mc)

	_, err = h.UpdateModuleSettings(context.Background(), &pb.UpdateModuleSettingsRequest{
		Name:     "mod-a",
		Settings: settings,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured == nil {
		t.Fatal("expected UpdateModuleConfig to be called")
	}

	nested, _, err := unstructured.NestedMap(captured.Object, "spec", "settings", "nested")
	if err != nil {
		t.Fatalf("reading nested: %v", err)
	}

	if _, hasB := nested["b"]; hasB {
		t.Errorf("expected nested.b to be removed, still present: %v", nested["b"])
	}

	if nested["c"] != float64(3) {
		t.Errorf("expected nested.c=3 preserved, got %v", nested["c"])
	}
}

func TestUpdateModuleSettings_Empty(t *testing.T) {
	getCalled := false

	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, _ string) (*unstructured.Unstructured, error) {
			getCalled = true
			obj := makeModuleConfig("mod-a", true, "")
			return &obj, nil
		},
	}

	settings, err := structpb.NewStruct(map[string]any{})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	h := NewModulesHandler(mc)

	_, err = h.UpdateModuleSettings(context.Background(), &pb.UpdateModuleSettingsRequest{
		Name:     "mod-a",
		Settings: settings,
	})
	if err == nil {
		t.Fatal("expected error for empty settings, got nil")
	}

	if getCalled {
		t.Error("GetModuleConfig must NOT be called when settings are empty")
	}
}

func TestUpdateModuleSettings_NotFound(t *testing.T) {
	mc := &mockClient{
		getModuleConfigFunc: func(_ context.Context, name string) (*unstructured.Unstructured, error) {
			return nil, fmt.Errorf("module config %q not found", name)
		},
	}

	settings, err := structpb.NewStruct(map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	h := NewModulesHandler(mc)

	_, err = h.UpdateModuleSettings(context.Background(), &pb.UpdateModuleSettingsRequest{
		Name:     "missing",
		Settings: settings,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The error must carry the underlying not-found cause for callers that wrap further.
	if errors.Is(err, errNotImplemented) {
		t.Errorf("got stub-level errNotImplemented; expected wrapped not-found: %v", err)
	}
}

// makeModuleConfigWithSettings builds a ModuleConfig with pre-populated spec.settings.
func makeModuleConfigWithSettings(name string, settings map[string]any) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "deckhouse.io/v1alpha1",
			"kind":       "ModuleConfig",
			"metadata":   map[string]any{"name": name},
			"spec": map[string]any{
				"enabled":  true,
				"settings": settings,
			},
		},
	}
}

// makeModule builds an unstructured Deckhouse Module resource for tests.
func makeModule(name string, weight int64, source, state string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "deckhouse.io/v1alpha1",
			"kind":       "Module",
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"weight": weight,
				"source": source,
			},
			"status": map[string]interface{}{
				"phase": state,
			},
		},
	}
}

// parseMaintenancePatch decodes a JSON merge patch and returns the value at
// spec.maintenance. Returns (value, ok) where ok=false when the path is absent.
func parseMaintenancePatch(t *testing.T, patch []byte) (any, bool) {
	t.Helper()

	var raw map[string]any

	err := json.Unmarshal(patch, &raw)
	if err != nil {
		t.Fatalf("invalid patch JSON: %v", err)
	}

	spec, ok := raw["spec"].(map[string]any)
	if !ok {
		return nil, false
	}

	v, ok := spec["maintenance"]

	return v, ok
}

func TestSetModuleMaintenance_EnableHappy(t *testing.T) {
	var capturedName string

	var capturedPatch []byte

	mc := &mockClient{
		patchModuleConfigFunc: func(_ context.Context, name string, patch []byte) (*unstructured.Unstructured, error) {
			capturedName = name
			capturedPatch = patch

			return &unstructured.Unstructured{}, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.SetModuleMaintenance(context.Background(), &pb.SetModuleMaintenanceRequest{
		Name:    "cert-manager",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.GetMaintenanceEnabled() {
		t.Error("expected maintenance_enabled=true in response")
	}
	if resp.GetName() != "cert-manager" {
		t.Errorf("expected echoed name=cert-manager, got %q", resp.GetName())
	}
	if capturedName != "cert-manager" {
		t.Errorf("expected client called with name=cert-manager, got %q", capturedName)
	}

	v, ok := parseMaintenancePatch(t, capturedPatch)
	if !ok {
		t.Fatal("expected patch to contain spec.maintenance")
	}
	if v != "NoResourceReconciliation" {
		t.Errorf("expected spec.maintenance=NoResourceReconciliation, got %v", v)
	}
}

func TestSetModuleMaintenance_DisableHappy(t *testing.T) {
	var capturedPatch []byte
	mc := &mockClient{
		patchModuleConfigFunc: func(_ context.Context, _ string, patch []byte) (*unstructured.Unstructured, error) {
			capturedPatch = patch

			return &unstructured.Unstructured{}, nil
		},
	}

	h := NewModulesHandler(mc)
	resp, err := h.SetModuleMaintenance(context.Background(), &pb.SetModuleMaintenanceRequest{
		Name:    "cert-manager",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetMaintenanceEnabled() {
		t.Error("expected maintenance_enabled=false in response")
	}

	v, ok := parseMaintenancePatch(t, capturedPatch)
	if !ok {
		t.Fatal("expected patch to contain spec.maintenance key (as null)")
	}
	if v != nil {
		t.Errorf("expected spec.maintenance=null in disable patch, got %v", v)
	}
}

func TestSetModuleMaintenance_PatchShape(t *testing.T) {
	var capturedPatch []byte
	mc := &mockClient{
		patchModuleConfigFunc: func(_ context.Context, _ string, patch []byte) (*unstructured.Unstructured, error) {
			capturedPatch = patch

			return &unstructured.Unstructured{}, nil
		},
	}

	h := NewModulesHandler(mc)
	_, err := h.SetModuleMaintenance(context.Background(), &pb.SetModuleMaintenanceRequest{
		Name:    "cert-manager",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"spec":{"maintenance":"NoResourceReconciliation"}}`
	if string(capturedPatch) != expected {
		t.Errorf("expected patch JSON %q, got %q", expected, string(capturedPatch))
	}
}

func TestSetModuleMaintenance_NotFound(t *testing.T) {
	mc := &mockClient{
		patchModuleConfigFunc: func(_ context.Context, name string, _ []byte) (*unstructured.Unstructured, error) {
			return nil, kerrors.NewNotFound(
				schema.GroupResource{Group: "deckhouse.io", Resource: "moduleconfigs"},
				name,
			)
		},
	}

	h := NewModulesHandler(mc)
	_, err := h.SetModuleMaintenance(context.Background(), &pb.SetModuleMaintenanceRequest{
		Name:    "missing",
		Enabled: true,
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !kerrors.IsNotFound(err) {
		t.Errorf("expected error to preserve IsNotFound semantics, got %v", err)
	}
}

func TestSetModuleMaintenance_Idempotent(t *testing.T) {
	calls := 0
	mc := &mockClient{
		patchModuleConfigFunc: func(_ context.Context, _ string, _ []byte) (*unstructured.Unstructured, error) {
			calls++

			return &unstructured.Unstructured{}, nil
		},
	}

	h := NewModulesHandler(mc)
	req := &pb.SetModuleMaintenanceRequest{Name: "cert-manager", Enabled: true}

	for i := 0; i < 3; i++ {
		_, err := h.SetModuleMaintenance(context.Background(), req)
		if err != nil {
			t.Fatalf("call %d returned unexpected error: %v", i+1, err)
		}
	}
	if calls != 3 {
		t.Errorf("expected 3 PatchModuleConfig calls (idempotent), got %d", calls)
	}
}
