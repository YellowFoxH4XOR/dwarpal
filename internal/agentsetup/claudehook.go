package agentsetup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Claude Code PreToolUse hook: before any `git commit` Bash call, run the
// gate; when it blocks, print the agent-readable JSON to stderr and exit 2 —
// Claude Code feeds a code-2 hook's stderr back to the model, so the agent
// sees retry_hints BEFORE the commit attempt instead of a failed tool call.
//
// Exit-code mapping is deliberate: dwarpal's "blocked" (1) becomes the hook's
// "deny with reason" (2); dwarpal's config error (2) also blocks — a broken
// gate must fail closed here too; pass (0) lets the commit proceed.
const claudeHookCommand = `out=$(dwarpal check --explain-for-agent 2>&1); code=$?; if [ $code -ne 0 ]; then echo "$out" >&2; exit 2; fi`

// hookMarker identifies our hook entry for idempotence checks.
const hookMarker = "dwarpal check --explain-for-agent"

// MergeClaudeSettings inserts the pre-flight hook into root/.claude/
// settings.json, creating the file if needed. Every existing key and hook is
// preserved (the file is decoded generically, never into a struct that would
// drop unknown fields). Returns the path and whether the hook was added
// (false = already present).
func MergeClaudeSettings(root string) (string, bool, error) {
	dir := filepath.Join(root, ".claude")
	path := filepath.Join(dir, "settings.json")

	settings := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &settings); err != nil {
			return path, false, fmt.Errorf("%s exists but is not valid JSON: %w — fix it or add the hook manually", path, err)
		}
	} else if !os.IsNotExist(err) {
		return path, false, err
	}

	if strings.Contains(mustJSON(settings), hookMarker) {
		return path, false, nil // already wired — idempotent no-op
	}

	entry := map[string]any{
		"matcher": "Bash",
		"hooks": []any{map[string]any{
			"type":    "command",
			"if":      "Bash(git commit*)",
			"command": claudeHookCommand,
		}},
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	pre, _ := hooks["PreToolUse"].([]any)
	hooks["PreToolUse"] = append(pre, entry)
	settings["hooks"] = hooks

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return path, false, err
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return path, false, err
	}
	return path, true, os.WriteFile(path, append(out, '\n'), 0o644)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
