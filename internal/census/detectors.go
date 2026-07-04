// Package census implements Dwarpal's whole-repo decay ratchet.
//
// Where `dwarpal check` is a diff-scoped, pre-commit gate — it sees only the
// staged change — the census sees the WHOLE repo. That distinction is the
// point: cumulative decay (dead code, unused exports, existing-vs-existing
// duplication) is a global, time-evolving property no diff gate can catch. A
// function is never born dead; it becomes dead in a later PR whose diff never
// textually touches it. See dwarpal_codebase_decay_strategy.md.
//
// Dwarpal owns the RATCHET (baseline + delta gate — the novel piece); it does
// NOT own the analysis. This registry wraps mature external detectors, the same
// "orchestrate a team's tools" move the exec-plugin gate already makes. Users
// install the detectors they want; a missing detector is surfaced, never
// silently treated as "zero decay".
package census

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strings"
)

// Scope says whether a detector is cheap enough for the pre-commit path.
//
//   - WholeRepo detectors (reachability, unused-export census) are O(repo) and
//     belong only in `dwarpal census`, never the 2s commit budget.
//   - DiffLocal detectors (unused imports/vars) are cheap and may additionally
//     be wired into `dwarpal check` via a plugin-gate `preset:` shorthand.
type Scope int

const (
	WholeRepo Scope = iota
	DiffLocal
)

func (s Scope) String() string {
	if s == DiffLocal {
		return "diff-local"
	}
	return "whole-repo"
}

// Detector is one wrapped external tool. Command is run via `sh -c` from the
// repo root (like plugin.Gate), and Parse turns its stdout into a count plus a
// set of stable item identities — the identities let the ratchet name WHAT
// increased, not just report a bigger number.
type Detector struct {
	Name    string
	Command string
	Scope   Scope
	// Parse extracts (count, items) from the tool's stdout. items should use a
	// position-independent identity (package-qualified symbol, message text)
	// so a dead symbol keeps the same key when unrelated lines shift above it.
	Parse func(stdout []byte) (count int, items []string, err error)
}

// registry is the built-in detector set. Adding a detector is one entry here.
//
// Verified live against this Go repo: deadcode, staticcheck-unused.
// Format-confident (grep-line), fixture-tested: vulture, ruff-unused.
// Deliberately NOT included yet: knip / jscpd — Node-ecosystem tools this repo
// can't exercise, and a guessed parser is worse than an absent one. Add them
// here once their output can be verified live.
var registry = map[string]Detector{
	"deadcode": {
		Name:    "deadcode",
		Command: "deadcode -json ./...",
		Scope:   WholeRepo,
		Parse:   parseDeadcodeJSON,
	},
	"vulture": {
		Name:    "vulture",
		Command: "vulture .",
		Scope:   WholeRepo,
		Parse:   parseGrepLines,
	},
	"staticcheck-unused": {
		Name:    "staticcheck-unused",
		Command: "staticcheck -checks U1000 ./...",
		Scope:   DiffLocal,
		Parse:   parseGrepLines,
	},
	"ruff-unused": {
		Name:    "ruff-unused",
		Command: "ruff check --select F401,F841 --output-format concise .",
		Scope:   DiffLocal,
		Parse:   parseGrepLines,
	},
}

// Lookup returns the named detector.
func Lookup(name string) (Detector, bool) {
	d, ok := registry[name]
	return d, ok
}

// Names returns the registered detector names (unsorted).
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	return out
}

// Available reports whether the detector's binary is on PATH. This is the
// pre-check that lets the census distinguish "tool absent" (skip / fail loud
// under --check) from "tool ran and found nothing" (a real zero).
func (d Detector) Available() bool {
	_, err := exec.LookPath(d.binary())
	return err == nil
}

// binary is the first shell token of Command — the executable to probe.
func (d Detector) binary() string {
	fields := strings.Fields(d.Command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// parseDeadcodeJSON parses `deadcode -json` output: an array of packages, each
// with a Funcs list. The item identity is the package-qualified function name
// (Path.Name) so it is stable under line-number churn.
func parseDeadcodeJSON(out []byte) (int, []string, error) {
	if len(strings.TrimSpace(string(out))) == 0 {
		return 0, nil, nil // no dead code: deadcode emits nothing
	}
	var pkgs []struct {
		Path  string `json:"Path"`
		Funcs []struct {
			Name string `json:"Name"`
		} `json:"Funcs"`
	}
	if err := json.Unmarshal(out, &pkgs); err != nil {
		return 0, nil, err
	}
	var items []string
	for _, p := range pkgs {
		for _, f := range p.Funcs {
			items = append(items, p.Path+"."+f.Name)
		}
	}
	return len(items), items, nil
}

// posLine matches a grep-style "file:line[:col]: message" diagnostic, the shape
// staticcheck / ruff / vulture all emit. Capturing group 1 is the file, group 2
// the message; the line/col in between are dropped so the identity is stable
// when code moves. Summary lines ("Found 3 errors.") lack the prefix and are
// ignored.
var posLine = regexp.MustCompile(`^([^:\s][^:]*):\d+(?::\d+)?:\s*(.*)$`)

// parseGrepLines counts diagnostic lines and derives a position-independent
// identity ("file: message") for each. Non-matching lines (banners, blanks)
// are skipped, so the count reflects findings only.
func parseGrepLines(out []byte) (int, []string, error) {
	var items []string
	for _, line := range strings.Split(string(out), "\n") {
		m := posLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		items = append(items, m[1]+": "+m[2])
	}
	return len(items), items, nil
}
