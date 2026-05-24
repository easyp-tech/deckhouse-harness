// Command deckhouse-mcp runs the Deckhouse MCP server over HTTP/SSE.
//
// The server is intended to run as a Pod in the d8-system namespace with
// in-cluster authentication. It exposes MCP tools generated from
// proto/deckhouse/v1/*.proto and backed by the typed/dynamic Kubernetes
// client in internal/k8s.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/client-go/rest"

	"github.com/easyp-tech/deckhouse-mcp/internal/handler"
	"github.com/easyp-tech/deckhouse-mcp/internal/k8s"
	pb "github.com/easyp-tech/deckhouse-mcp/proto/deckhouse/v1"
)

const (
	defaultListenAddr     = ":8080"
	shutdownTimeout       = 10 * time.Second
	readHeaderTimeout     = 5 * time.Second
	serverImplName        = "deckhouse-mcp"
	serverImplVersion     = "0.2.0"
	serverImplDescription = "Deckhouse Kubernetes Platform MCP server"
)

func main() {
	err := run()
	if err != nil {
		log.Fatalf("deckhouse-mcp: %v", err)
	}
}

func run() error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("loading in-cluster config: %w", err)
	}

	client, err := k8s.New(cfg)
	if err != nil {
		return fmt.Errorf("creating k8s client: %w", err)
	}

	server, err := newServer(client)
	if err != nil {
		return err
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = defaultListenAddr
	}

	httpServer := &http.Server{
		Addr: addr,
		Handler: mcp.NewSSEHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serveUntilShutdown(ctx, httpServer)
}

// serveUntilShutdown runs the HTTP server in a goroutine and blocks until the
// context is cancelled or the server fails. It then performs a graceful
// shutdown bounded by shutdownTimeout.
func serveUntilShutdown(ctx context.Context, srv *http.Server) error {
	serverErr := make(chan error, 1)

	go func() {
		log.Printf("deckhouse-mcp: listening on %q", srv.Addr)

		listenErr := srv.ListenAndServe()
		if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("http server: %w", listenErr)
		}

		close(serverErr)
	}()

	select {
	case listenErr := <-serverErr:
		return listenErr
	case <-ctx.Done():
		log.Printf("deckhouse-mcp: shutdown signal received")
	}

	// Use a fresh background context for shutdown: the inherited ctx is already
	// cancelled by SIGTERM/SIGINT, and graceful shutdown needs its own deadline.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownErr := srv.Shutdown(shutdownCtx) //nolint:contextcheck // see comment above
	if shutdownErr != nil {
		return fmt.Errorf("http shutdown: %w", shutdownErr)
	}

	return nil
}

// newServer constructs an MCP server with all generated tool handlers
// registered against the provided Kubernetes client.
func newServer(client k8s.Client) (*mcp.Server, error) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverImplName,
		Title:   serverImplDescription,
		Version: serverImplVersion,
	}, nil)

	err := pb.RegisterDiagnosticsAPITools(server, handler.NewDiagnosticsHandler(client))
	if err != nil {
		return nil, fmt.Errorf("registering diagnostics tools: %w", err)
	}

	err = pb.RegisterModulesAPITools(server, handler.NewModulesHandler(client))
	if err != nil {
		return nil, fmt.Errorf("registering modules tools: %w", err)
	}

	err = pb.RegisterNodesAPITools(server, handler.NewNodesHandler(client))
	if err != nil {
		return nil, fmt.Errorf("registering nodes tools: %w", err)
	}

	err = pb.RegisterReleasesAPITools(server, handler.NewReleasesHandler(client))
	if err != nil {
		return nil, fmt.Errorf("registering releases tools: %w", err)
	}

	err = pb.RegisterConfigAPITools(server, handler.NewConfigHandler(client))
	if err != nil {
		return nil, fmt.Errorf("registering config tools: %w", err)
	}

	err = pb.RegisterSourcesAPITools(server, handler.NewSourcesHandler(client))
	if err != nil {
		return nil, fmt.Errorf("registering sources tools: %w", err)
	}

	return server, nil
}
