package phases

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/sectersion/ledger/journal"
	"github.com/sectersion/ledger/registry"
	"github.com/sectersion/ledger/worker"
)

// ValidateResult is the outcome of running tests/linters/plan-compliance
// against dir.
type ValidateResult struct {
	Passed         bool
	VetOutput      string
	TestOutput     string
	Compliance     string
	FailedPackages []string
}

// Validate runs `go vet` and `go test` in dir, then spawns a worker to
// check the result against plan (plan-compliance, not just green tests).
func Validate(ctx context.Context, dir, plan, journalPath string) (ValidateResult, error) {
	var result ValidateResult

	vetCmd := exec.CommandContext(ctx, "go", "vet", "./...")
	vetCmd.Dir = dir
	vetOut, vetErr := vetCmd.CombinedOutput()
	result.VetOutput = string(vetOut)

	testCmd := exec.CommandContext(ctx, "go", "test", "-json", "./...")
	testCmd.Dir = dir
	testOut, testErr := testCmd.CombinedOutput()
	result.TestOutput = string(testOut)
	result.FailedPackages = failedPackages(testOut)

	compliance, complianceErr := worker.Run(worker.WithAgentID(ctx, "validate"), dir, fmt.Sprintf(
		"Given this implementation plan, check whether the current working tree actually implements it. Reply with exactly \"PASS\" if it does, or \"FAIL: <reason>\" if it doesn't.\n\nPlan:\n%s", plan), worker.ReadOnlyArgs()...)
	result.Compliance = compliance
	if complianceErr != nil {
		result.Compliance = "FAIL: " + complianceErr.Error()
	}

	result.Passed = vetErr == nil && testErr == nil && result.Compliance == "PASS"

	journal.Append(journalPath, "phase", map[string]any{
		"phase":           "validate",
		"passed":          result.Passed,
		"failed_packages": result.FailedPackages,
		"compliance":      result.Compliance,
	})
	return result, nil
}

// FailingOwners maps each failed package/path to its current registry
// owner, deduplicated, for a scoped RPI re-run of just the team(s) that
// owned the failing slice (no dependency graph needed, per PLAN.md).
func FailingOwners(reg *registry.Registry, failedPaths []string) []string {
	seen := map[string]bool{}
	var owners []string
	for _, path := range failedPaths {
		owner, held := reg.Owner(path)
		if !held || seen[owner] {
			continue
		}
		seen[owner] = true
		owners = append(owners, owner)
	}
	return owners
}

// failedPackages parses `go test -json` output for package-level failures.
func failedPackages(testJSON []byte) []string {
	var failed []string
	scanner := bufio.NewScanner(bytes.NewReader(testJSON))
	for scanner.Scan() {
		var event struct {
			Action  string `json:"Action"`
			Package string `json:"Package"`
			Test    string `json:"Test"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.Action == "fail" && event.Test == "" && event.Package != "" {
			failed = append(failed, event.Package)
		}
	}
	return failed
}
