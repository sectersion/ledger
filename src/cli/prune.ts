import { pruneWorktree } from "../worktree.ts";

const [repoPath, worktreePath] = process.argv.slice(2);

if (!repoPath || !worktreePath) {
  console.error("usage: tsx src/cli/prune.ts <repoPath> <worktreePath>");
  process.exit(1);
}

await pruneWorktree(repoPath, worktreePath);
console.log(`pruned ${worktreePath}`);
