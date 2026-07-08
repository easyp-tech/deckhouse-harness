package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/easyp-tech/protoc-gen-mcp/mcpruntime"

	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

// PromptsHandler implements the generated MCP prompt handler interface
// (pb.File_proto_deckhouse_v1_prompts_protoPromptHandler). Each prompt returns a
// single user message: a parameterized playbook that orchestrates the existing
// tools and resources. Prompts are pure templates — they do not touch the
// cluster themselves; the agent executes the referenced tools.
type PromptsHandler struct{}

// NewPromptsHandler creates a PromptsHandler.
func NewPromptsHandler() *PromptsHandler {
	return &PromptsHandler{}
}

// userMessage wraps playbook text as a single user prompt message. It returns an
// error slot (always nil) so callers can `return userMessage(...)` directly to
// satisfy the generated handler signature.
func userMessage(text string) ([]mcpruntime.PromptMessage, error) {
	return []mcpruntime.PromptMessage{
		{
			Role:    mcpruntime.RoleUser,
			Content: &mcpruntime.TextContent{Text: strings.TrimSpace(text)},
		},
	}, nil
}

// DiagnoseClusterHealth backs the diagnose_cluster_health prompt.
func (h *PromptsHandler) DiagnoseClusterHealth(
	_ context.Context,
	req *pb.DiagnoseClusterHealth,
) ([]mcpruntime.PromptMessage, error) {
	scope := "the whole cluster"
	if focus := req.GetFocus(); focus != "" {
		scope = fmt.Sprintf("the %q area", focus)
	}

	return userMessage(fmt.Sprintf(`
You are diagnosing the health of a Deckhouse Kubernetes cluster. Focus on %s.

Follow these steps and summarize findings with concrete next actions:
1. Read the resource deckhouse://cluster/status (or call GetClusterStatus) for the
   aggregated picture: node readiness, per-NodeGroup health, errored ModuleConfigs,
   pending DeckhouseReleases, unhealthy pod count, and Deckhouse version.
2. If any NodeGroup is degraded, call ListNodeGroups and GetNodeGroup for details.
3. If unhealthy pods > 0, call ListUnhealthyPods and inspect the worst offenders
   with GetPodLogs.
4. If modules are errored, call ListModuleConfigs / GetModuleConfig for the failing
   ones and check GetDeckhouseLogs for related errors.
5. If releases are pending, call ListDeckhouseReleases to see what awaits approval.

End with a short prioritized list: what is broken, likely cause, and the exact
tool or command to remediate.`, scope))
}

// TriageUnhealthyPods backs the triage_unhealthy_pods prompt.
func (h *PromptsHandler) TriageUnhealthyPods(
	_ context.Context,
	req *pb.TriageUnhealthyPods,
) ([]mcpruntime.PromptMessage, error) {
	scope := "across all namespaces"
	if ns := req.GetNamespace(); ns != "" {
		scope = fmt.Sprintf("in namespace %q", ns)
	}

	return userMessage(fmt.Sprintf(`
Triage unhealthy pods %s in a Deckhouse cluster.

Steps:
1. Call ListUnhealthyPods (pass the namespace filter if one was given) to get the
   failing pods, their status/reason, and restart counts.
2. For each notable pod, call GetPodLogs to read recent logs; prefer the container
   named in the failure and use the grep filter for error keywords.
3. Correlate with node health: if pods on one node fail, call GetNode and
   GetNodeEvents for that node.
4. Distinguish causes: CrashLoopBackOff (app/config), ImagePullBackOff (registry/
   credentials), Pending (scheduling/resources), OOMKilled (limits).

Report each pod with: namespace/name, root-cause hypothesis, supporting evidence
from logs/events, and the concrete fix.`, scope))
}

// InvestigateNode backs the investigate_node prompt.
func (h *PromptsHandler) InvestigateNode(
	_ context.Context,
	req *pb.InvestigateNode,
) ([]mcpruntime.PromptMessage, error) {
	name := req.GetName()

	return userMessage(fmt.Sprintf(`
Investigate the Deckhouse cluster node %q.

Steps:
1. Read deckhouse://nodes/%s (or call GetNode) for status, conditions,
   allocatable/capacity resources, StaticInstance phase, and recent events.
2. Call GetNodeEvents for the full recent event history and look for pressure,
   networking, or kubelet problems.
3. Identify the node's NodeGroup and call GetNodeGroup to see whether the group is
   healthy and up to date.
4. If the node is NotReady or degraded, check kubelet/system pods on it via
   ListUnhealthyPods and GetPodLogs.

Conclude with the node's overall state, the specific problem (if any), and the
recommended remediation (cordon/drain, restart kubelet, fix NodeGroup, etc.).`, name, name))
}

// PrepareDeckhouseUpgrade backs the prepare_deckhouse_upgrade prompt.
func (h *PromptsHandler) PrepareDeckhouseUpgrade(
	_ context.Context,
	req *pb.PrepareDeckhouseUpgrade,
) ([]mcpruntime.PromptMessage, error) {
	target := "the next pending release"
	if v := req.GetTargetVersion(); v != "" {
		target = fmt.Sprintf("version %s", v)
	}

	return userMessage(fmt.Sprintf(`
Prepare a Deckhouse cluster for an upgrade to %s. Do NOT approve anything yet —
this is a readiness review.

Steps:
1. Call ListDeckhouseReleases (or read deckhouse://releases) to see all releases
   and their phase; identify the target and any that must be applied first.
2. For the target, call GetDeckhouseRelease to read its requirements and changelog.
3. Read deckhouse://cluster/status (or GetClusterStatus): the cluster must be
   healthy before upgrading — no degraded NodeGroups, no errored modules, minimal
   unhealthy pods.
4. Review module readiness with ListModules / ListModuleConfigs for anything in a
   failing state that could block the upgrade.

Produce a go/no-go checklist: unmet requirements, health blockers, and — only if
everything is green — the exact ApproveRelease call to proceed.`, target))
}

// AddWorkerNode backs the add_worker_node prompt.
func (h *PromptsHandler) AddWorkerNode(
	_ context.Context,
	req *pb.AddWorkerNode,
) ([]mcpruntime.PromptMessage, error) {
	nodeGroup := req.GetNodeGroup()

	address := req.GetAddress()
	addressLine := "the SSH-reachable address of the target machine"
	if address != "" {
		addressLine = fmt.Sprintf("the machine at %s", address)
	}

	return userMessage(fmt.Sprintf(`
Guide the operator through adding a worker node to NodeGroup %q using a
StaticInstance (%s).

Steps:
1. Confirm the target NodeGroup exists and is a Static type: call GetNodeGroup for
   %q. If it does not exist or is not static, stop and explain what is needed.
2. Ensure SSH credentials exist for bootstrapping (SSHCredentials referenced by the
   StaticInstance's credentialsRef). Create them if missing.
3. Create the StaticInstance for the machine (address, labels selecting NodeGroup
   %q) so Deckhouse can bootstrap it.
4. Watch progress: call ListStaticInstances / GetStaticInstance and wait for the
   phase to reach Running; then confirm the node appears via ListNodes and becomes
   Ready.

At each step state the exact tool call and expected result, and how to recover if
the phase gets stuck (Pending/Bootstrapping).`, nodeGroup, addressLine, nodeGroup, nodeGroup))
}
