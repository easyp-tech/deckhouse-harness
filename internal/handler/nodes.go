package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/easyp-tech/deckhouse-mcp/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

const (
	defaultSSHPort                      = 22
	defaultTimeoutSeconds               = 900
	defaultDrainTimeout                 = 300
	pollInterval                        = 30 * time.Second
	mirrorPodAnnotation                 = "kubernetes.io/config.mirror"
	defaultNodeGroupConfigurationWeight = 100
)

var (
	errPrivateKeyRequired    = errors.New("privateKey is required")
	errScriptContentRequired = errors.New("content is required")
	errNodeGroupsRequired    = errors.New("node_groups must contain at least one entry")
)

// NodesHandler implements pb.NodesAPIToolHandler.
type NodesHandler struct {
	client k8s.Client
}

// NewNodesHandler creates a new NodesHandler.
func NewNodesHandler(client k8s.Client) *NodesHandler {
	return &NodesHandler{client: client}
}

func (h *NodesHandler) CreateSSHCredentials(
	ctx context.Context,
	req *pb.CreateSSHCredentialsRequest,
) (*pb.CreateSSHCredentialsResponse, error) {
	if req.GetPrivateKey() == "" {
		return nil, errPrivateKeyRequired
	}

	port := int64(defaultSSHPort)
	if req.Port != nil {
		port = int64(req.GetPort())
	}

	encodedKey := base64.StdEncoding.EncodeToString([]byte(req.GetPrivateKey()))

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "deckhouse.io/v1alpha2",
			"kind":       "SSHCredentials",
			"metadata": map[string]any{
				"name": req.GetName(),
			},
			"spec": map[string]any{
				"user":          req.GetUser(),
				"privateSSHKey": encodedKey,
				"sshPort":       port,
			},
		},
	}

	if req.SshExtraArgs != nil {
		spec, ok := obj.Object["spec"].(map[string]any)
		if !ok {
			spec = make(map[string]any)
			obj.Object["spec"] = spec
		}

		spec["sshExtraArgs"] = req.GetSshExtraArgs()
	}

	if req.SudoPassword != nil {
		spec, ok := obj.Object["spec"].(map[string]any)
		if !ok {
			spec = make(map[string]any)
			obj.Object["spec"] = spec
		}

		spec["sudoPasswordEncoded"] = base64.StdEncoding.EncodeToString([]byte(req.GetSudoPassword()))
	}

	_, err := h.client.CreateSSHCredentials(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("creating SSHCredentials: %w", err)
	}

	return &pb.CreateSSHCredentialsResponse{Name: req.GetName()}, nil
}

func (h *NodesHandler) CreateStaticInstance(
	ctx context.Context,
	req *pb.CreateStaticInstanceRequest,
) (*pb.CreateStaticInstanceResponse, error) {
	labels := make(map[string]any)
	for k, v := range req.GetLabels() {
		labels[k] = v
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "deckhouse.io/v1alpha2",
			"kind":       "StaticInstance",
			"metadata": map[string]any{
				"name":   req.GetName(),
				"labels": labels,
			},
			"spec": map[string]any{
				"address": req.GetAddress(),
				"credentialsRef": map[string]any{
					"kind": "SSHCredentials",
					"name": req.GetCredentialsRef(),
				},
			},
		},
	}

	created, err := h.client.CreateStaticInstance(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("creating StaticInstance: %w", err)
	}

	var resultLabels map[string]string

	createdLabels := created.GetLabels()
	if createdLabels != nil {
		resultLabels = createdLabels
	} else {
		resultLabels = req.GetLabels()
	}

	return &pb.CreateStaticInstanceResponse{
		Name:           req.GetName(),
		Address:        req.GetAddress(),
		CredentialsRef: req.GetCredentialsRef(),
		Labels:         resultLabels,
	}, nil
}

func (h *NodesHandler) AddWorkerNode(
	ctx context.Context,
	req *pb.AddWorkerNodeRequest,
) (*pb.AddWorkerNodeResponse, error) {
	var nodeName string

	if req.NodeName != nil {
		nodeName = req.GetNodeName()
	} else {
		nodeName = strings.ReplaceAll(req.GetAddress(), ".", "-")
	}

	credsName := nodeName + "-creds"

	sshPort := int32(defaultSSHPort)
	if req.SshPort != nil {
		sshPort = req.GetSshPort()
	}

	err := h.provisionNodeResources(ctx, req, nodeName, credsName, sshPort)
	if err != nil {
		return nil, err
	}

	waitReady := true
	if req.WaitReady != nil {
		waitReady = req.GetWaitReady()
	}

	timeoutSec := int32(defaultTimeoutSeconds)
	if req.TimeoutSeconds != nil {
		timeoutSec = req.GetTimeoutSeconds()
	}

	phase, elapsed, timedOut, err := h.waitForNode(ctx, waitReady, nodeName, timeoutSec)
	if err != nil {
		return nil, err
	}

	return &pb.AddWorkerNodeResponse{
		NodeName:           nodeName,
		SshCredentialsName: credsName,
		StaticInstanceName: nodeName,
		Phase:              phase,
		Elapsed:            elapsed,
		TimedOut:           timedOut,
	}, nil
}

// DeleteStaticInstance deletes a StaticInstance resource by name.
func (h *NodesHandler) DeleteStaticInstance(
	ctx context.Context,
	req *pb.DeleteStaticInstanceRequest,
) (*pb.DeleteStaticInstanceResponse, error) {
	err := h.client.DeleteStaticInstance(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("deleting StaticInstance %s: %w", req.GetName(), err)
	}

	return &pb.DeleteStaticInstanceResponse{Success: true}, nil
}

// RemoveNode cordons the node, deletes non-DaemonSet pods, then removes its StaticInstance.
func (h *NodesHandler) RemoveNode(
	ctx context.Context,
	req *pb.RemoveNodeRequest,
) (*pb.RemoveNodeResponse, error) {
	// Verify StaticInstance exists (static nodes only).
	_, err := h.client.GetStaticInstance(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("static instance for node %q not found: %w", req.GetName(), err)
	}

	drain := true
	if req.Drain != nil {
		drain = req.GetDrain()
	}

	drained := false

	if drain {
		err = h.drainNode(ctx, req.GetName())
		if err != nil {
			return nil, err
		}

		drained = true
	}

	// Delete the StaticInstance — Deckhouse will clean up.
	err = h.client.DeleteStaticInstance(ctx, req.GetName())
	if err != nil {
		return &pb.RemoveNodeResponse{Drained: drained, Deleted: false},
			fmt.Errorf("deleting StaticInstance %s: %w", req.GetName(), err)
	}

	return &pb.RemoveNodeResponse{
		Drained: drained,
		Deleted: true,
	}, nil
}

// isDaemonSetPod reports whether the pod is owned by a DaemonSet.
func isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "DaemonSet" {
			return true
		}
	}

	return false
}

// CreateNodeGroup creates a new NodeGroup resource.
func (h *NodesHandler) CreateNodeGroup(
	ctx context.Context,
	req *pb.CreateNodeGroupRequest,
) (*pb.CreateNodeGroupResponse, error) {
	spec := map[string]any{
		"nodeType": req.GetNodeType(),
	}

	if req.Count != nil {
		spec["staticInstances"] = map[string]any{
			"count": int64(req.GetCount()),
		}
	}

	if req.Disruptions != nil {
		spec["disruptions"] = map[string]any{
			"approvalMode": req.GetDisruptions(),
		}
	}

	if req.MaxPodsPerNode != nil {
		spec["kubelet"] = map[string]any{
			"maxPods": int64(req.GetMaxPodsPerNode()),
		}
	}

	labels := make(map[string]any)
	for k, v := range req.GetLabels() {
		labels[k] = v
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "deckhouse.io/v1",
			"kind":       "NodeGroup",
			"metadata": map[string]any{
				"name":   req.GetName(),
				"labels": labels,
			},
			"spec": spec,
		},
	}

	created, err := h.client.CreateNodeGroup(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("creating NodeGroup %s: %w", req.GetName(), err)
	}

	resp := &pb.CreateNodeGroupResponse{
		Name:     req.GetName(),
		NodeType: req.GetNodeType(),
	}
	if req.Count != nil {
		count := req.Count
		resp.Count = count
	}

	_ = created

	return resp, nil
}

// WaitNodeReady polls a StaticInstance until it reaches Running phase or timeout expires.
func (h *NodesHandler) WaitNodeReady(
	ctx context.Context,
	req *pb.WaitNodeReadyRequest,
) (*pb.WaitNodeReadyResponse, error) {
	timeoutSec := int32(defaultTimeoutSeconds)
	if req.TimeoutSeconds != nil {
		timeoutSec = req.GetTimeoutSeconds()
	}

	phase, elapsed, timedOut, err := h.pollStaticInstance(ctx, req.GetName(), timeoutSec)
	if err != nil {
		return nil, err
	}

	return &pb.WaitNodeReadyResponse{
		Phase:    phase,
		Elapsed:  elapsed,
		TimedOut: timedOut,
	}, nil
}

// CordonNode marks a node as unschedulable. Reads the current Spec.Unschedulable
// before issuing the cordon to return the previous state (see ADR-1).
func (h *NodesHandler) CordonNode(
	ctx context.Context,
	req *pb.CordonNodeRequest,
) (*pb.CordonNodeResponse, error) {
	node, err := h.client.GetNode(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("getting node %s: %w", req.GetName(), err)
	}

	previousState := node.Spec.Unschedulable

	err = h.client.CordonNode(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("cordoning node %s: %w", req.GetName(), err)
	}

	return &pb.CordonNodeResponse{PreviousState: previousState}, nil
}

// UncordonNode marks a node as schedulable. Reads the current Spec.Unschedulable
// before issuing the uncordon to return the previous state.
func (h *NodesHandler) UncordonNode(
	ctx context.Context,
	req *pb.UncordonNodeRequest,
) (*pb.UncordonNodeResponse, error) {
	node, err := h.client.GetNode(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("getting node %s: %w", req.GetName(), err)
	}

	previousState := node.Spec.Unschedulable

	if !previousState {
		// Already schedulable — skip the write to remain idempotent and quiet.
		return &pb.UncordonNodeResponse{PreviousState: false}, nil
	}

	err = h.client.UncordonNode(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("uncordoning node %s: %w", req.GetName(), err)
	}

	return &pb.UncordonNodeResponse{PreviousState: previousState}, nil
}

// DrainNode performs a composite drain: cordon the node, then iteratively evict
// every non-DaemonSet, non-mirror pod via the Eviction API, respecting PDBs.
// Returns when all evictable pods are gone or the timeout elapses.
//
// DaemonSet-managed pods and mirror pods are skipped silently — both are
// expected to remain on the node by Kubernetes design.
//
// PDB-blocked pods (TooManyRequests) stay in the work-set for the next poll
// cycle. Other eviction errors are recorded in failed_pods and the pod is
// dropped from the work-set to avoid spinning on a permanently broken pod.
func (h *NodesHandler) DrainNode(
	ctx context.Context,
	req *pb.DrainNodeRequest,
) (*pb.DrainNodeResponse, error) {
	start := time.Now()
	timeoutSec := req.GetTimeoutSeconds()
	if timeoutSec <= 0 {
		timeoutSec = defaultDrainTimeout
	}
	deadline := start.Add(time.Duration(timeoutSec) * time.Second)

	// Step 1 — cordon. Failure aborts the drain.
	err := h.client.CordonNode(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("cordoning node %s: %w", req.GetName(), err)
	}

	// Step 2 — list drainable pods on the node.
	pods, err := h.client.ListPods(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("listing pods for drain of %s: %w", req.GetName(), err)
	}

	type podID struct {
		namespace, name string
	}

	pending := make(map[podID]struct{})
	for _, pod := range pods {
		if pod.Spec.NodeName != req.GetName() {
			continue
		}
		pod := pod
		if isDaemonSetPod(&pod) || isMirrorPod(&pod) {
			continue
		}
		pending[podID{pod.Namespace, pod.Name}] = struct{}{}
	}

	var (
		evicted    int32
		failedPods []string
	)

	// Step 3 — eviction loop.
	for len(pending) > 0 {
		for id := range pending {
			evictErr := h.client.EvictPod(ctx, id.namespace, id.name)
			switch {
			case evictErr == nil:
				evicted++
				delete(pending, id)
			case kerrors.IsNotFound(evictErr):
				// Pod already gone — count as evicted.
				evicted++
				delete(pending, id)
			case kerrors.IsTooManyRequests(evictErr):
				// PDB-blocked — retry on next poll cycle, leave in pending.
			default:
				failedPods = append(failedPods, id.namespace+"/"+id.name)
				delete(pending, id)
			}
		}

		if len(pending) == 0 {
			break
		}

		// Wait before re-polling. Honour deadline & context cancellation.
		if time.Now().After(deadline) {
			break
		}

		remaining := time.Until(deadline)
		wait := pollInterval
		if remaining < wait {
			wait = remaining
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("draining node %s: %w", req.GetName(), ctx.Err())
		case <-time.After(wait):
		}
	}

	timedOut := len(pending) > 0 && time.Now().After(deadline)
	elapsed := time.Since(start).Truncate(time.Second).String()

	return &pb.DrainNodeResponse{
		Cordoned:     true,
		EvictedCount: evicted,
		FailedPods:   failedPods,
		TimedOut:     timedOut,
		Elapsed:      elapsed,
	}, nil
}

// isMirrorPod returns true if the pod is a static (mirror) pod managed by kubelet.
func isMirrorPod(pod *corev1.Pod) bool {
	_, ok := pod.Annotations[mirrorPodAnnotation]

	return ok
}

// DeleteSSHCredentials deletes an SSHCredentials resource by name.
func (h *NodesHandler) DeleteSSHCredentials(
	ctx context.Context,
	req *pb.DeleteSSHCredentialsRequest,
) (*pb.DeleteSSHCredentialsResponse, error) {
	err := h.client.DeleteSSHCredentials(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("deleting SSHCredentials %s: %w", req.GetName(), err)
	}

	return &pb.DeleteSSHCredentialsResponse{Deleted: true}, nil
}

// CreateNodeGroupConfiguration creates a NodeGroupConfiguration resource — a
// bash script bound to one or more NodeGroups. REQ-3.4: content and
// node_groups are validated locally before any K8s API call.
func (h *NodesHandler) CreateNodeGroupConfiguration(
	ctx context.Context,
	req *pb.CreateNodeGroupConfigurationRequest,
) (*pb.CreateNodeGroupConfigurationResponse, error) {
	if req.GetContent() == "" {
		return nil, errScriptContentRequired
	}

	nodeGroups := req.GetNodeGroups()
	if len(nodeGroups) == 0 {
		return nil, errNodeGroupsRequired
	}

	weight := int64(defaultNodeGroupConfigurationWeight)
	if req.Weight != nil {
		weight = int64(req.GetWeight())
	}

	// Proto map repeated string → []any for unstructured.
	nodeGroupsAny := make([]any, 0, len(nodeGroups))
	for _, ng := range nodeGroups {
		nodeGroupsAny = append(nodeGroupsAny, ng)
	}

	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": deckhouseAPIVersionV1Alpha1,
		"kind":       "NodeGroupConfiguration",
		"metadata": map[string]any{
			"name": req.GetName(),
		},
		"spec": map[string]any{
			"content":    req.GetContent(),
			"nodeGroups": nodeGroupsAny,
			"weight":     weight,
		},
	}}

	_, err := h.client.CreateNodeGroupConfiguration(ctx, obj)
	if err != nil {
		return nil, fmt.Errorf("creating node group configuration %s: %w", req.GetName(), err)
	}

	return &pb.CreateNodeGroupConfigurationResponse{
		Created: true,
		Name:    req.GetName(),
	}, nil
}

// DeleteNodeGroup deletes a NodeGroup resource by name.
func (h *NodesHandler) DeleteNodeGroup(
	ctx context.Context,
	req *pb.DeleteNodeGroupRequest,
) (*pb.DeleteNodeGroupResponse, error) {
	err := h.client.DeleteNodeGroup(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("deleting NodeGroup %s: %w", req.GetName(), err)
	}

	return &pb.DeleteNodeGroupResponse{Deleted: true}, nil
}

func (h *NodesHandler) waitForNode(
	ctx context.Context,
	waitReady bool,
	nodeName string,
	timeoutSec int32,
) (string, string, bool, error) {
	start := time.Now()

	if !waitReady {
		return phasePending, "0s", false, nil
	}

	phase, elapsed, timedOut, err := h.pollStaticInstance(ctx, nodeName, timeoutSec)
	if err != nil {
		return "", time.Since(start).Truncate(time.Second).String(), false, err
	}

	return phase, elapsed, timedOut, nil
}

func (h *NodesHandler) pollStaticInstance(
	ctx context.Context,
	name string,
	timeoutSec int32,
) (string, string, bool, error) {
	start := time.Now()
	timeout := time.Duration(timeoutSec) * time.Second
	deadline := start.Add(timeout)
	phase := phasePending
	timedOut := false

	for {
		staticInstance, siErr := h.client.GetStaticInstance(ctx, name)
		if siErr != nil {
			return phase, time.Since(start).Truncate(time.Second).String(), false,
				fmt.Errorf("polling StaticInstance %s: %w", name, siErr)
		}

		currentPhase := unstructuredNestedString(staticInstance.Object, "status", "currentStatus", "phase")
		if currentPhase != "" {
			phase = currentPhase
		}

		if phase == phaseRunning {
			break
		}

		if time.Now().After(deadline) {
			timedOut = true

			break
		}

		select {
		case <-ctx.Done():
			return phase, time.Since(start).Truncate(time.Second).String(), false,
				fmt.Errorf("polling StaticInstance %s: %w", name, ctx.Err())
		case <-time.After(pollInterval):
		}
	}

	return phase, time.Since(start).Truncate(time.Second).String(), timedOut, nil
}

func (h *NodesHandler) drainNode(ctx context.Context, nodeName string) error {
	err := h.client.CordonNode(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("cordoning node %s: %w", nodeName, err)
	}

	pods, err := h.client.ListPods(ctx, "")
	if err != nil {
		return fmt.Errorf("listing pods for node %s: %w", nodeName, err)
	}

	for _, pod := range pods {
		if pod.Spec.NodeName != nodeName {
			continue
		}

		if isDaemonSetPod(&pod) {
			continue
		}

		err = h.client.DeletePod(ctx, pod.Namespace, pod.Name)
		if err != nil {
			return fmt.Errorf("deleting pod %s/%s: %w", pod.Namespace, pod.Name, err)
		}
	}

	return nil
}

func (h *NodesHandler) provisionNodeResources(
	ctx context.Context,
	req *pb.AddWorkerNodeRequest,
	nodeName, credsName string,
	sshPort int32,
) error {
	_, err := h.CreateSSHCredentials(ctx, &pb.CreateSSHCredentialsRequest{
		Name:       credsName,
		User:       req.GetSshUser(),
		PrivateKey: req.GetPrivateKey(),
		Port:       &sshPort,
	})
	if err != nil {
		return fmt.Errorf("creating SSHCredentials for node %s: %w", nodeName, err)
	}

	_, err = h.CreateStaticInstance(ctx, &pb.CreateStaticInstanceRequest{
		Name:           nodeName,
		Address:        req.GetAddress(),
		CredentialsRef: credsName,
		Labels: map[string]string{
			"node.deckhouse.io/group": req.GetNodeGroup(),
		},
	})
	if err != nil {
		return fmt.Errorf(
			"creating StaticInstance for node %s (SSHCredentials %q already created): %w",
			nodeName, credsName, err,
		)
	}

	return nil
}
