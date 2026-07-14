package ownership

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sectersion/ledger/registry"
)

// NewHTTPHandler serves one ownership MCP server per role over HTTP, all
// backed by the same in-memory reg, at "<mount>/<role>". A single shared
// Registry (mutex-protected) is what makes concurrent workers' ownership
// requests actually mutually exclusive, rather than racing independent
// per-process copies.
func NewHTTPHandler(reg *registry.Registry, mount string) *mcp.StreamableHTTPHandler {
	mount = strings.TrimSuffix(mount, "/")
	return mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		role := strings.TrimPrefix(r.URL.Path, mount+"/")
		return NewServer(reg, role)
	}, nil)
}

// WriteMCPConfig writes a --mcp-config JSON file wiring the "ownership"
// server at baseURL+"/"+role for the given worker.
func WriteMCPConfig(path, baseURL, role string) error {
	config := map[string]any{
		"mcpServers": map[string]any{
			"ownership": map[string]any{
				"type": "http",
				"url":  strings.TrimSuffix(baseURL, "/") + "/" + role,
			},
		},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
