// Command deckhouse-harness runs the Deckhouse MCP server.
//
// Transport: stdio only — newline-delimited JSON-RPC on stdin/stdout. Used by
// local MCP clients such as Claude Desktop, Cursor, Claude Code, etc. The server
// runtime (github.com/easyp-tech/protoc-gen-mcp/mcpruntime) ships stdio only; any
// HTTP/SSE fronting must be built externally on top of Server.HandleRaw.
//
// Kubernetes auth: in-cluster config (inside Pod) or ~/.kube/config / KUBECONFIG.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/easyp-tech/protoc-gen-mcp/mcpruntime"
	"github.com/urfave/cli/v3"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/easyp-tech/deckhouse-harness/internal/handler"
	"github.com/easyp-tech/deckhouse-harness/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-harness/proto/deckhouse/v1"
)

const (
	serverImplName    = "deckhouse-harness"
	serverImplVersion = "0.4.0"
)

func main() {
	cmd := &cli.Command{
		Name:    "deckhouse-harness",
		Version: serverImplVersion,
		Usage:   "Deckhouse Kubernetes Platform MCP server (stdio transport)",
		Action: func(ctx context.Context, _ *cli.Command) error {
			return run(ctx)
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("deckhouse-harness: %v", err)
	}
}

func run(ctx context.Context) error {
	logger := configureLogger()

	cfg, err := loadKubeConfig()
	if err != nil {
		return fmt.Errorf("loading kube config: %w", err)
	}

	client, err := k8s.New(cfg)
	if err != nil {
		return fmt.Errorf("creating k8s client: %w", err)
	}

	server, err := newServer(ctx, client)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting MCP server", "transport", "stdio", "version", serverImplVersion)
	return mcpruntime.ServeStdio(ctx, server)
}

// loadKubeConfig resolves a Kubernetes rest.Config for authenticating with the
// cluster. It tries in-cluster config first (when running inside a Pod), then
// falls back to the kubeconfig at KUBECONFIG or ~/.kube/config for local
// execution.
func loadKubeConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, configOverrides,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("loading kubeconfig: %w", err)
	}

	return cfg, nil
}

// configureLogger builds a *slog.Logger from LOG_LEVEL, LOG_OUTPUT, and LOG_FILE
// environment variables. Logs never go to stdout (that channel is reserved for
// the MCP protocol). Default output is stderr.
func configureLogger() *slog.Logger {
	var level slog.Level
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var w io.Writer = os.Stderr
	switch os.Getenv("LOG_OUTPUT") {
	case "file":
		path := os.Getenv("LOG_FILE")
		if path == "" {
			log.Println("LOG_OUTPUT=file but LOG_FILE is empty, falling back to stderr")
		} else {
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				log.Printf("failed to open log file %q: %v, falling back to stderr", path, err)
			} else {
				w = f
			}
		}
	case "discard":
		w = io.Discard
	default:
		w = os.Stderr
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	log.SetOutput(w)

	return logger
}

// newServer constructs an MCP server with all generated tool handlers, resources,
// and prompts registered against the provided Kubernetes client.
func newServer(ctx context.Context, client k8s.Client) (*mcpruntime.Server, error) {
	server := mcpruntime.NewServer(serverImplName, serverImplVersion)

	if err := pb.RegisterDiagnosticsAPITools(server, handler.NewDiagnosticsHandler(client)); err != nil {
		return nil, fmt.Errorf("registering diagnostics tools: %w", err)
	}

	if err := pb.RegisterModulesAPITools(server, handler.NewModulesHandler(client)); err != nil {
		return nil, fmt.Errorf("registering modules tools: %w", err)
	}

	if err := pb.RegisterNodesAPITools(server, handler.NewNodesHandler(client)); err != nil {
		return nil, fmt.Errorf("registering nodes tools: %w", err)
	}

	if err := pb.RegisterReleasesAPITools(server, handler.NewReleasesHandler(client)); err != nil {
		return nil, fmt.Errorf("registering releases tools: %w", err)
	}

	if err := pb.RegisterConfigAPITools(server, handler.NewConfigHandler(client)); err != nil {
		return nil, fmt.Errorf("registering config tools: %w", err)
	}

	if err := pb.RegisterSourcesAPITools(server, handler.NewSourcesHandler(client)); err != nil {
		return nil, fmt.Errorf("registering sources tools: %w", err)
	}

	// MCP resources (read-only cluster context by URI).
	if err := pb.RegisterFile_proto_deckhouse_v1_resources_protoResources(
		ctx, server, handler.NewResourcesHandler(client),
	); err != nil {
		return nil, fmt.Errorf("registering resources: %w", err)
	}

	// MCP prompts (parameterized diagnostic/operational playbooks).
	if err := pb.RegisterFile_proto_deckhouse_v1_prompts_protoPrompts(
		server, handler.NewPromptsHandler(),
	); err != nil {
		return nil, fmt.Errorf("registering prompts: %w", err)
	}

	return server, nil
}
