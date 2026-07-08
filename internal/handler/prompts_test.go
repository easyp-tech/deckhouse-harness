package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/easyp-tech/protoc-gen-mcp/mcpruntime"

	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

// promptText extracts the text of the single user message a prompt returns.
func promptText(t *testing.T, msgs []mcpruntime.PromptMessage) string {
	t.Helper()
	if len(msgs) != 1 {
		t.Fatalf("messages: got %d, want 1", len(msgs))
	}
	if msgs[0].Role != mcpruntime.RoleUser {
		t.Errorf("role: got %q, want user", msgs[0].Role)
	}
	tc, ok := msgs[0].Content.(*mcpruntime.TextContent)
	if !ok {
		t.Fatalf("content type: got %T, want *mcpruntime.TextContent", msgs[0].Content)
	}
	if strings.TrimSpace(tc.Text) == "" {
		t.Fatal("prompt text is empty")
	}
	return tc.Text
}

func TestPrompts_DiagnoseClusterHealth_DefaultScope(t *testing.T) {
	h := NewPromptsHandler()
	msgs, err := h.DiagnoseClusterHealth(context.Background(), &pb.DiagnoseClusterHealth{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := promptText(t, msgs)
	if !strings.Contains(text, "the whole cluster") {
		t.Errorf("expected default scope in text, got: %s", text)
	}
	if !strings.Contains(text, "deckhouse://cluster/status") {
		t.Errorf("expected cluster status resource reference in text")
	}
}

func TestPrompts_DiagnoseClusterHealth_Focus(t *testing.T) {
	h := NewPromptsHandler()
	focus := "modules"
	msgs, err := h.DiagnoseClusterHealth(context.Background(), &pb.DiagnoseClusterHealth{Focus: &focus})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(promptText(t, msgs), "modules") {
		t.Error("expected focus value interpolated into text")
	}
}

func TestPrompts_TriageUnhealthyPods_Namespace(t *testing.T) {
	h := NewPromptsHandler()
	ns := "d8-system"
	msgs, err := h.TriageUnhealthyPods(context.Background(), &pb.TriageUnhealthyPods{Namespace: &ns})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := promptText(t, msgs)
	if !strings.Contains(text, "d8-system") {
		t.Errorf("expected namespace in text, got: %s", text)
	}
	if !strings.Contains(text, "ListUnhealthyPods") {
		t.Error("expected ListUnhealthyPods tool reference")
	}
}

func TestPrompts_InvestigateNode(t *testing.T) {
	h := NewPromptsHandler()
	msgs, err := h.InvestigateNode(context.Background(), &pb.InvestigateNode{Name: "worker-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := promptText(t, msgs)
	if !strings.Contains(text, "worker-1") {
		t.Errorf("expected node name in text, got: %s", text)
	}
	if !strings.Contains(text, "deckhouse://nodes/worker-1") {
		t.Error("expected node resource URI with name interpolated")
	}
}

func TestPrompts_PrepareDeckhouseUpgrade_Target(t *testing.T) {
	h := NewPromptsHandler()
	v := "v1.74.0"
	msgs, err := h.PrepareDeckhouseUpgrade(context.Background(), &pb.PrepareDeckhouseUpgrade{TargetVersion: &v})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := promptText(t, msgs)
	if !strings.Contains(text, "v1.74.0") {
		t.Errorf("expected target version in text, got: %s", text)
	}
	if !strings.Contains(text, "ApproveRelease") {
		t.Error("expected ApproveRelease reference")
	}
}

func TestPrompts_AddWorkerNode(t *testing.T) {
	h := NewPromptsHandler()
	addr := "10.0.0.5"
	msgs, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNode{NodeGroup: "workers", Address: &addr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := promptText(t, msgs)
	if !strings.Contains(text, "workers") {
		t.Errorf("expected node group in text, got: %s", text)
	}
	if !strings.Contains(text, "10.0.0.5") {
		t.Error("expected address interpolated into text")
	}
}

func TestPrompts_AddWorkerNode_NoAddress(t *testing.T) {
	h := NewPromptsHandler()
	msgs, err := h.AddWorkerNode(context.Background(), &pb.AddWorkerNode{NodeGroup: "workers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(promptText(t, msgs), "workers") {
		t.Error("expected node group in text")
	}
}
