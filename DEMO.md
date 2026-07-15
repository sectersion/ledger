# Demo

## Recording

_Not yet recorded — see below for the exact steps to reproduce one._

To capture your own:

```
mkdir demo-repo && cd demo-repo && git init
# add a small (~150-300 LOC) Go repo: a couple of handlers/store files with
# a few unhandled error paths (ignored Close()/Atoi() errors etc.) so the
# task below has real, spread-out work to do
git add -A && git commit -m init

asciinema rec demo.cast
go run github.com/sectersion/ledger/cmd/ledger run . "add structured logging to all error paths"
# in the TUI: watch the fanned-out worker roles run in parallel, approve
# the Research and Plan gates, let Implement/Validate/Review run, approve
# the final Review gate, watch Ship produce a branch/PR
# ctrl+d (or `q` after it finishes) to stop the recording
```

Then either embed the `.cast` directly (asciinema player) or convert to a
GIF with [agg](https://github.com/asciinema/agg):

```
agg demo.cast demo.gif
```

and drop `![demo](demo.gif)` at the top of the README.

## What the recording should show

1. `ledger run` kicking off — Research phase fanning out to its subagents.
2. The Research gate firing (`a`/`r`/`e` modal) and being approved.
3. Plan phase, its gate, and Implement fanning out multiple roles
   (Backend/Frontend/Test or similar) into parallel worktrees — the agent
   tree showing more than one agent `running` at once.
4. A Validate failure triggering a scoped re-implement of just the failing
   owner (if the demo task reproduces one) — otherwise Validate passing on
   the first try is a fine outcome to show too.
5. The Review gate and its output.
6. Ship producing a final branch/PR, and the TUI's "pipeline done" toast.
