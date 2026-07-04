package census

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
)

// DetectorStat is one detector's whole-repo result. Skipped is true when the
// detector's binary is absent — recorded, not silently dropped, so `--check`
// can refuse to certify a ratchet it couldn't actually run.
type DetectorStat struct {
	Name       string   `json:"name"`
	Scope      string   `json:"scope"`
	Count      int      `json:"count"`
	Items      []string `json:"items"`
	Skipped    bool     `json:"skipped,omitempty"`
	SkipReason string   `json:"skip_reason,omitempty"`
}

// Report is the census result over all requested detectors.
type Report struct {
	Detectors []DetectorStat `json:"detectors"`
}

// Run executes each named whole-repo detector from root and aggregates counts.
//
// A detector whose binary is absent is recorded as Skipped and Run continues —
// an expected, graceful condition the caller decides how to treat. A detector
// that RAN but whose output could not be parsed is a hard error (fail loud):
// we must never let a broken parser masquerade as "zero decay".
func Run(ctx context.Context, root string, names []string) (*Report, error) {
	rep := &Report{}
	for _, name := range names {
		d, ok := Lookup(name)
		if !ok {
			return nil, fmt.Errorf("unknown detector %q", name)
		}
		if !d.Available() {
			rep.Detectors = append(rep.Detectors, DetectorStat{
				Name:       name,
				Scope:      d.Scope.String(),
				Skipped:    true,
				SkipReason: fmt.Sprintf("%q not found on PATH", d.binary()),
			})
			continue
		}

		// Exit code is not a reliable signal (deadcode exits 0 on findings,
		// ruff/vulture exit nonzero), so we key off parse success, not the code.
		cmd := exec.CommandContext(ctx, "sh", "-c", d.Command)
		cmd.Dir = root
		out, runErr := cmd.Output()

		count, items, perr := d.Parse(out)
		if perr != nil {
			return nil, fmt.Errorf("detector %q: parsing output failed: %w", name, perr)
		}
		// A run that produced no parseable findings AND errored to start (vs.
		// merely exiting nonzero) is a real failure — surface it.
		if runErr != nil && count == 0 {
			if _, isExit := runErr.(*exec.ExitError); !isExit {
				return nil, fmt.Errorf("detector %q failed to run: %w", name, runErr)
			}
		}

		sort.Strings(items) // deterministic output and stable diffs
		rep.Detectors = append(rep.Detectors, DetectorStat{
			Name:  name,
			Scope: d.Scope.String(),
			Count: count,
			Items: items,
		})
	}
	return rep, nil
}

// SkippedConfigured returns the names of detectors that were requested but
// could not run because their binary was absent. `dwarpal census --check`
// treats a non-empty result as fatal: a ratchet you couldn't run is not a pass.
func (r *Report) SkippedConfigured() []string {
	var out []string
	for _, d := range r.Detectors {
		if d.Skipped {
			out = append(out, d.Name)
		}
	}
	return out
}
