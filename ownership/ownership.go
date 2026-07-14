// Package ownership is an MCP server exposing request_ownership/
// release_ownership tools, backed by the lock registry (registry package).
// It's passed to each worker via --mcp-config so a worker can only touch
// paths the orchestrator has granted it.
package ownership

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sectersion/ledger/registry"
)

type ownershipArgs struct {
	Path string `json:"path" jsonschema:"the file path to request or release ownership of"`
}

// NewServer builds an MCP server whose tools are checked against reg,
// scoping every grant/release to owner (the calling agent's id).
func NewServer(reg *registry.Registry, owner string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "ledger-ownership", Version: "v0.0.1"}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "request_ownership",
		Description: "Request exclusive ownership of a file path before editing it",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ownershipArgs) (*mcp.CallToolResult, any, error) {
		granted, err := reg.Acquire(args.Path, owner)
		if err != nil {
			return nil, nil, err
		}
		text := "granted"
		if !granted {
			text = "denied: already owned by another agent"
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "release_ownership",
		Description: "Release ownership of a file path previously requested",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ownershipArgs) (*mcp.CallToolResult, any, error) {
		if err := reg.Release(args.Path, owner); err != nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}, IsError: true}, nil, nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "released"}}}, nil, nil
	})

	return s
}
