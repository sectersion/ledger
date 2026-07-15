# Show HN: ledger – a fixed research/plan/implement/validate/review/ship pipeline for headless Claude Code agents

`ledger` is a CLI that points a team of headless Claude Code agents at a repo
and a task description, then runs them through a fixed six-stage pipeline —
research, plan, implement, validate, review, ship — instead of one long
freeform agent loop. Each stage's agents run in their own git worktree so
parallel work can't collide, a shared ownership registry stops two agents
from writing the same file at once, and you get a terminal UI to watch it
happen and approve or reject at three gates (after research, after
planning, before shipping). The idea is to make large multi-file changes —
migrations, audits, cross-cutting refactors — reviewable in stages instead
of one big diff you have to trust all at once.

The two design decisions I'd want feedback on: the fixed pipeline, and the
worktree/ownership isolation. The pipeline is fixed on purpose — the model
never gets to decide "skip validation this run" or silently re-plan
mid-implementation, which is the failure mode I was actually trying to
avoid (confidently wrong across 20 files, not caught until review). The
cost is you lose the flexibility a freeform loop has for small or
open-ended tasks, where the fixed structure is just overhead. Isolation is
the other bet: every role gets its own git worktree, and a small in-process
ownership registry (exposed to workers as an MCP server) makes file
ownership mutually exclusive across agents, not just checked in isolation —
so two workers can't stomp on the same file even if they're both editing
concurrently.

What's missing, honestly: no asciinema/GIF demo yet (working on it — see
DEMO.md for the exact repro steps if you want to record one before I do);
pause/resume is UI-visible only, not a real process suspend, since there's
no portable SIGSTOP equivalent via `os/exec` on Windows; and the model
router's role-decomposition step (deciding which worker roles a plan needs)
is itself one `claude` call with a JSON-array fallback, so a malformed
response falls back to a fixed Backend/Frontend/Test split rather than
failing outright. Next up: a real recorded demo, and looking at whether the
validate→re-implement scoped retry should also feed back into planning
when the same package keeps failing repeatedly instead of just retrying
the same plan.
