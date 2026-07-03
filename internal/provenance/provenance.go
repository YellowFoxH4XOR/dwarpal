// Package provenance detects whether a change was authored by an AI agent.
//
// Provenance is what lets Dwarpal apply gates to agent commits only (the
// default, PRD §5.3 apply_gates_to: agent-only) while leaving human commits
// untouched — the key to avoiding hook fatigue (risk R2).
//
// Detection order follows PRD §5.2 Gate 2, most-explicit signal first:
//  1. AGENTGATE_AGENT environment variable (set by agent wrappers)
//  2. Co-Authored-By trailers matching a known agent identity
//  3. Branch prefix (agent/, ai/)
//  4. otherwise: not an agent
//
// Note on timing: at pre-commit the commit message does not exist yet, so
// trailer detection is only meaningful when a message is available (commit-msg
// hook or --range analysis). Detect takes the message as an optional input and
// degrades to env + branch signals when it is empty.
package provenance

import (
	"os"
	"regexp"
	"strings"
)

// EnvVar is the environment variable an agent wrapper sets to declare itself.
const EnvVar = "AGENTGATE_AGENT"

// Source records which signal identified the agent.
type Source string

const (
	SourceNone      Source = "none"
	SourceEnv       Source = "env"
	SourceTrailer   Source = "trailer"
	SourceBranch    Source = "branch"
	SourceHeuristic Source = "heuristic"
)

// Provenance is the detection result. Agent is the identified agent name when
// known (e.g. from the env var value or the matched trailer).
type Provenance struct {
	IsAgent bool
	Source  Source
	Agent   string
}

// Detector holds the configured signals. Zero value is usable but matches
// nothing; construct via New with the repo's configured prefixes/trailers.
type Detector struct {
	branchPrefixes []string
	agentTrailers  []string
	heuristics     []*regexp.Regexp
}

// New builds a Detector. branchPrefixes are branch name prefixes that mark
// agent work (e.g. "agent/", "ai/"); agentTrailers are agent identity
// substrings matched against Co-Authored-By lines (e.g. "Claude", "Cursor").
func New(branchPrefixes, agentTrailers []string) *Detector {
	return &Detector{branchPrefixes: branchPrefixes, agentTrailers: agentTrailers}
}

// WithHeuristics adds the configurable fourth detection signal (PRD Gate 2):
// user-supplied regexes matched against the branch name and commit message.
// Invalid patterns are skipped — a bad heuristic must weaken detection, not
// break the pipeline (and the config layer validates them loudly anyway).
func (d *Detector) WithHeuristics(patterns []string) *Detector {
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			d.heuristics = append(d.heuristics, re)
		}
	}
	return d
}

// Detect resolves provenance from the environment, the current branch, and an
// optional commit message. Pass an empty commitMsg when none is available yet.
func (d *Detector) Detect(branch, commitMsg string) Provenance {
	// 1. Explicit env var wins.
	if v := strings.TrimSpace(os.Getenv(EnvVar)); v != "" {
		return Provenance{IsAgent: true, Source: SourceEnv, Agent: v}
	}
	// 2. Co-Authored-By trailer matching a known agent identity.
	if agent := d.matchTrailer(commitMsg); agent != "" {
		return Provenance{IsAgent: true, Source: SourceTrailer, Agent: agent}
	}
	// 3. Branch prefix.
	for _, p := range d.branchPrefixes {
		if p != "" && strings.HasPrefix(branch, p) {
			return Provenance{IsAgent: true, Source: SourceBranch}
		}
	}
	// 4. Configurable heuristics — the weakest signal, checked last.
	for _, re := range d.heuristics {
		if re.MatchString(branch) || (commitMsg != "" && re.MatchString(commitMsg)) {
			return Provenance{IsAgent: true, Source: SourceHeuristic}
		}
	}
	// 5. Not identified as an agent.
	return Provenance{IsAgent: false, Source: SourceNone}
}

// matchTrailer returns the agent identity from the first Co-Authored-By trailer
// that contains a configured agent substring (case-insensitive), or "".
func (d *Detector) matchTrailer(commitMsg string) string {
	if commitMsg == "" || len(d.agentTrailers) == 0 {
		return ""
	}
	for _, line := range strings.Split(commitMsg, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(trimmed), "co-authored-by:") {
			continue
		}
		for _, t := range d.agentTrailers {
			if t != "" && strings.Contains(strings.ToLower(trimmed), strings.ToLower(t)) {
				return t
			}
		}
	}
	return ""
}
