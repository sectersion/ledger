package ownership

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sectersion/ledger/registry"
)

func connectAgent(t *testing.T, reg *registry.Registry, owner string) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	server := NewServer(reg, owner)
	go server.Run(context.Background(), serverTransport)

	client := mcp.NewClient(&mcp.Implementation{Name: owner, Version: "v0.0.1"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("connect %s: %v", owner, err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func requestOwnership(t *testing.T, session *mcp.ClientSession, path string) string {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "request_ownership",
		Arguments: map[string]any{"path": path},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func TestOverlappingRequestsOneDenied(t *testing.T) {
	reg, err := registry.Load(filepath.Join(t.TempDir(), "registry.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	agentA := connectAgent(t, reg, "agent-a")
	agentB := connectAgent(t, reg, "agent-b")

	if got := requestOwnership(t, agentA, "src/foo.go"); got != "granted" {
		t.Fatalf("agent-a request: got %q, want granted", got)
	}
	if got := requestOwnership(t, agentB, "src/foo.go"); got != "denied: already owned by another agent" {
		t.Fatalf("agent-b request: got %q, want denied", got)
	}
}
