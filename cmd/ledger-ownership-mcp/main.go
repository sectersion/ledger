// Command ledger-ownership-mcp runs the ownership MCP server on stdio, for
// use as a worker's --mcp-config entry.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sectersion/ledger/ownership"
	"github.com/sectersion/ledger/registry"
)

func main() {
	registryPath := flag.String("registry", "", "path to the lock registry JSON file")
	owner := flag.String("owner", "", "this worker's agent id")
	flag.Parse()

	if *registryPath == "" || *owner == "" {
		log.Fatal("both -registry and -owner are required")
	}

	reg, err := registry.Load(*registryPath)
	if err != nil {
		log.Fatalf("load registry: %v", err)
	}

	server := ownership.NewServer(reg, *owner)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server: %v", err)
	}
}
