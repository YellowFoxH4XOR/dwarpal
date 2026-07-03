package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// explainEntry is the static rationale shown for one rule (PRD §5.1). It is
// keyed by rule_id (either the bare form, e.g. "max-lines", or the
// "gate/rule_id" form, e.g. "diff_budget/max-lines").
type explainEntry struct {
	Title   string `json:"title"`
	Why     string `json:"why"`
	Fix     string `json:"fix"`
	DocsURL string `json:"docs_url"`
}

// explainTable covers every rule emitted by the built-in gates today. Keys
// are the bare rule_id; runExplain also accepts the "gate/rule_id" form and
// strips the gate prefix before lookup.
var explainTable = map[string]explainEntry{
	"max-lines": {
		Title: "diff_budget: max-lines",
		Why:   "Oversized diffs are hard to review carefully, and large AI-generated changesets are the leading cause of unreviewed bugs slipping into a repo.",
		Fix:   "Split the change into smaller, focused commits or PRs, each under the configured line budget. Adjust gates.diff_budget.max_lines in .dwarpal.yml if the budget itself is wrong for this repo.",
	},
	"max-files": {
		Title: "diff_budget: max-files",
		Why:   "Touching many files in one change makes it hard to reason about blast radius and increases the odds an AI agent strayed outside the intended task.",
		Fix:   "Narrow the change to fewer files, or split unrelated edits into separate commits/PRs. Adjust gates.diff_budget.max_files in .dwarpal.yml if needed.",
	},
	"max-new-files": {
		Title: "diff_budget: max-new-files",
		Why:   "A burst of newly created files often signals scope creep or speculative scaffolding an AI agent added without being asked.",
		Fix:   "Remove or defer files not required for this task, or split the work so new files are introduced incrementally. Adjust gates.diff_budget.max_new_files in .dwarpal.yml if needed.",
	},
	"protected-branch": {
		Title: "branch_policy: protected-branch",
		Why:   "Committing directly to a protected branch (e.g. main) bypasses code review and CI, which is exactly the safety net you want between an AI agent and production.",
		Fix:   "Create a feature branch and open a pull request instead of committing directly. Adjust gates.branch_policy.protected in .dwarpal.yml if the protected branch list is wrong.",
	},
	"out-of-scope": {
		Title: "scope: out-of-scope",
		Why:   "A change touching files outside the task manifest's declared paths suggests the agent drifted beyond what it was asked to do.",
		Fix:   "Either restrict the change to the declared paths, or update the task manifest (run `dwarpal task <id> --paths ...`) to include the new paths intentionally.",
	},
	"no-task-manifest": {
		Title: "scope: no-task-manifest",
		Why:   "Without a task manifest declaring intended scope, the scope gate has nothing to check the diff against, defeating its purpose.",
		Fix:   "Run `dwarpal task <id> --paths <glob>...` before making changes to declare what this task is allowed to touch.",
	},
	"below-threshold": {
		Title: "diff_coverage: below-threshold",
		Why:   "Changed lines with no test coverage are the lines most likely to hide a bug introduced by this change, AI-authored or not.",
		Fix:   "Add or extend tests so the changed lines are exercised, then regenerate the coverage artifact before checking again.",
	},
	"no-new-lint-suppressions": {
		Title: "ai_patterns: no-new-lint-suppressions",
		Why:   "Newly added lint-suppression comments (e.g. eslint-disable, nolint) often mean an agent silenced a real problem instead of fixing it.",
		Fix:   "Fix the underlying issue instead of suppressing the linter. If the suppression is genuinely warranted, add it with a comment explaining why, in a reviewed, deliberate change.",
	},
	"no-hardcoded-secrets/private-key": {
		Title: "ai_patterns: no-hardcoded-secrets/private-key",
		Why:   "A private key committed to the repo is compromised the moment it lands in history, even if removed later.",
		Fix:   "Remove the key from the diff, rotate it immediately, and load it from a secret manager or environment variable instead.",
	},
	"no-hardcoded-secrets/aws-key": {
		Title: "ai_patterns: no-hardcoded-secrets/aws-key",
		Why:   "An AWS access key committed to the repo can be scraped by bots within minutes of becoming public, leading to account compromise.",
		Fix:   "Remove the key from the diff, rotate it in IAM immediately, and load credentials via an environment variable, instance role, or secret manager.",
	},
	"no-hardcoded-secrets/assigned-literal": {
		Title: "ai_patterns: no-hardcoded-secrets/assigned-literal",
		Why:   "A literal string assigned to a variable named like a secret (token, password, apiKey, ...) is a common pattern for accidentally committed credentials.",
		Fix:   "Move the value out of source into an environment variable, config file excluded from version control, or secret manager, and rotate it if it was ever a real credential.",
	},
	"no-sql-concat": {
		Title: "ai_patterns: no-sql-concat",
		Why:   "Building SQL by string concatenation or interpolation is a classic SQL-injection vector, and AI agents reach for it far more often than parameterized queries.",
		Fix:   "Use parameterized queries or an ORM/query builder instead of concatenating user input (or any variable) into SQL text.",
	},
	"no-broad-catch": {
		Title: "ai_patterns: no-broad-catch",
		Why:   "Catching all errors broadly (bare except, catch (e) {}, recover() with no handling) silently swallows failures an agent didn't know how to handle, hiding bugs from tests and users alike.",
		Fix:   "Catch the specific error types you expect and handle or re-raise the rest, or at minimum log the swallowed error instead of discarding it.",
	},
}

func newExplainCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "explain <finding-id>",
		Short: "Explain why a rule exists and how to fix a finding it raised",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runExplain(args[0], jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the entry as JSON")
	return cmd
}

// gatePrefixes lists the recognized gate names, so runExplain can strip a
// leading "gate/" prefix without corrupting compound rule_ids that
// themselves contain a slash (e.g. "no-hardcoded-secrets/aws-key").
var gatePrefixes = []string{
	"diff_budget/", "branch_policy/", "scope/", "diff_coverage/", "ai_patterns/",
}

// runExplain looks up id (accepting either the bare rule_id or the
// "gate/rule_id" form) in explainTable and prints its rationale (PRD §5.1).
func runExplain(id string, jsonOut bool) error {
	key := id
	for _, prefix := range gatePrefixes {
		if strings.HasPrefix(key, prefix) {
			key = strings.TrimPrefix(key, prefix)
			break
		}
	}

	entry, ok := explainTable[key]
	if ok {
		// DocsURL is computed from the canonical gate/rule mapping so explain,
		// findings, and the docs tree can never drift apart.
		if gate, rule, found := strings.Cut(entry.Title, ": "); found {
			entry.DocsURL = finding.DocsURL(gate, rule)
		}
	}
	if !ok {
		return &exitError{code: 2, msg: fmt.Sprintf("unknown rule id %q; run `dwarpal rules` to see the active gates and rules", id)}
	}

	if jsonOut {
		b, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return &exitError{code: 2, msg: err.Error()}
		}
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("%s\n\n", entry.Title)
	fmt.Printf("Why it matters:\n  %s\n\n", entry.Why)
	fmt.Printf("How to fix:\n  %s\n\n", entry.Fix)
	fmt.Printf("Docs: %s\n", entry.DocsURL)
	return nil
}
