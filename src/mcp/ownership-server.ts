import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { pathToFileURL } from "node:url";
import { z } from "zod";
import { requestOwnership, releaseOwnership } from "../registry.ts";

export function createOwnershipServer(ledgerDir: string, agentId: string): McpServer {
  const server = new McpServer({ name: "ledger-ownership", version: "0.0.1" });

  server.registerTool(
    "request_ownership",
    {
      description: "Request exclusive ownership of a file path before editing it.",
      inputSchema: { path: z.string() },
    },
    async ({ path }) => {
      const granted = await requestOwnership(ledgerDir, path, agentId);
      return {
        content: [{ type: "text", text: granted ? "granted" : "denied: already owned by another agent" }],
      };
    },
  );

  server.registerTool(
    "release_ownership",
    {
      description: "Release ownership of a file path previously requested.",
      inputSchema: { path: z.string() },
    },
    async ({ path }) => {
      await releaseOwnership(ledgerDir, path, agentId);
      return { content: [{ type: "text", text: "released" }] };
    },
  );

  return server;
}

async function main(): Promise<void> {
  const [ledgerDir, agentId] = process.argv.slice(2);
  if (!ledgerDir || !agentId) {
    console.error("usage: ownership-server.ts <ledgerDir> <agentId>");
    process.exit(1);
  }
  const server = createOwnershipServer(ledgerDir, agentId);
  await server.connect(new StdioServerTransport());
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
