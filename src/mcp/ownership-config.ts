export function ownershipMcpConfig(ledgerDir: string, agentId: string): object {
  return {
    mcpServers: {
      "ledger-ownership": {
        command: "tsx",
        args: ["src/mcp/ownership-server.ts", ledgerDir, agentId],
      },
    },
  };
}
