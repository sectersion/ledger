// Package modelrouting is the orchestrator's own one-off "best model for
// this job" call, filtered by settings.json's model allow-list.
package modelrouting

import (
	"context"
	"fmt"
	"strings"

	"github.com/sectersion/ledger/settings"
	"github.com/sectersion/ledger/worker"
)

const choosePromptTmpl = `Pick the single best model to run this job, from exactly this list: %s.
Reply with only the model name from that list, nothing else.

Job:
%s`

// Choose asks the orchestrator's own claude call to pick the best
// allow-listed model for job, run in dir. If the allow-list has only one
// entry, or the response doesn't match an allowed model, it falls back to
// the first allow-listed model without spending a call/guessing wrong.
func Choose(ctx context.Context, dir, job string) (string, error) {
	allow := settings.LoadDefault().ModelAllowList
	if len(allow) == 0 {
		return "", fmt.Errorf("modelrouting: empty model allow-list")
	}
	if len(allow) == 1 {
		return allow[0], nil
	}

	out, err := worker.Run(ctx, dir, fmt.Sprintf(choosePromptTmpl, strings.Join(allow, ", "), job))
	if err != nil {
		return allow[0], nil
	}

	choice := strings.TrimSpace(out)
	for _, model := range allow {
		if choice == model {
			return model, nil
		}
	}
	return allow[0], nil
}

// Args returns the --model flag/value pair for a worker.Run/SpawnWorker
// extraArgs list.
func Args(model string) []string {
	return []string{"--model", model}
}
