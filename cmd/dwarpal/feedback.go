package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// feedbackLogFile collects false-positive reports locally. Dwarpal's trust
// promise is "no telemetry, ever" — so feedback NEVER phones home. The command
// records locally and prints a prefilled GitHub issue URL the user can choose
// to open (PRD §9: false-positive rate via opt-in reports).
const feedbackLogFile = "feedback.log"

type feedbackRecord struct {
	Timestamp string `json:"timestamp"`
	RuleID    string `json:"rule_id"`
	Reason    string `json:"reason"`
}

func newFeedbackCmd() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "feedback <rule-id>",
		Short: "Record a false-positive report locally (never sent anywhere)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runFeedback(args[0], reason)
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "why this finding was wrong (required)")
	return cmd
}

func runFeedback(ruleID, reason string) error {
	if reason == "" {
		return &exitError{code: 2, msg: "--reason is required"}
	}
	root, err := repoRoot()
	if err != nil {
		return &exitError{code: 2, msg: "a git repository is required"}
	}

	rec := feedbackRecord{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RuleID:    ruleID,
		Reason:    reason,
	}
	dir := filepath.Join(root, ".dwarpal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	line, _ := json.Marshal(rec)
	f, err := os.OpenFile(filepath.Join(dir, feedbackLogFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return &exitError{code: 2, msg: err.Error()}
	}

	issueURL := "https://github.com/YellowFoxH4XOR/dwarpal/issues/new?" + url.Values{
		"title": {fmt.Sprintf("[false-positive] %s", ruleID)},
		"body":  {fmt.Sprintf("Rule: `%s`\n\nWhat happened:\n%s\n\n(Reported via `dwarpal feedback` — details added by the user; nothing was sent automatically.)", ruleID, reason)},
	}.Encode()

	fmt.Printf("• recorded locally in .dwarpal/%s (nothing was sent anywhere)\n", feedbackLogFile)
	fmt.Println("• to share it with the project, open:")
	fmt.Println("  " + issueURL)
	return nil
}
