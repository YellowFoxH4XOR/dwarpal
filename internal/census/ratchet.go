package census

import (
	"fmt"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// updateHint is the standard escape valve appended to every ratchet finding:
// either remove the new decay, or explicitly accept it as the new baseline.
const updateHint = "Remove the newly-added items above, or run `dwarpal census --update-baseline` to accept them as the new baseline."

// Diff is the ratchet: it compares a fresh report against the committed
// baseline and emits a Finding for each detector whose count INCREASED. Existing
// debt is grandfathered (only increases block); the number can only ratchet
// down. This is the whole point — it makes each PR accountable for its MARGINAL
// contribution to decay, something a pure diff gate can never measure.
//
// When item identities are available, one finding names each genuinely-new item
// (set difference against the baseline). When a detector reports only a count
// (degraded identity), a single finding reports the delta. A detector missing
// from the baseline is treated as a zero baseline, so its first run blocks and
// the fix is to accept it via --update-baseline.
func Diff(base *Baseline, rep *Report) []finding.Finding {
	var findings []finding.Finding
	for _, d := range rep.Detectors {
		if d.Skipped {
			continue // handled separately as a fail-loud condition by --check
		}
		var prev BaselineEntry
		if base != nil {
			prev = base.Detectors[d.Name] // zero value if absent → first run blocks
		}
		if d.Count <= prev.Count {
			continue // flat or improved: the ratchet only blocks increases
		}

		gate := "census/" + d.Name
		newItems := subtract(d.Items, prev.Items)
		if len(newItems) == 0 {
			// Count rose but we can't name the culprits (degraded identity):
			// still block, reporting the delta.
			findings = append(findings, finding.Finding{
				Gate:      gate,
				RuleID:    "increase",
				Severity:  finding.SeverityError,
				Message:   fmt.Sprintf("%s rose from %d to %d (%d new)", d.Name, prev.Count, d.Count, d.Count-prev.Count),
				RetryHint: updateHint,
			})
			continue
		}
		for _, item := range newItems {
			findings = append(findings, finding.Finding{
				Gate:      gate,
				RuleID:    "increase",
				Severity:  finding.SeverityError,
				Message:   "new: " + item,
				RetryHint: updateHint,
			})
		}
	}
	return findings
}

// subtract returns the items in cur that are not in base, preserving cur's
// order. This is what lets the ratchet name only the NEW decay while
// grandfathering everything already in the baseline.
func subtract(cur, base []string) []string {
	seen := make(map[string]bool, len(base))
	for _, b := range base {
		seen[b] = true
	}
	var out []string
	for _, c := range cur {
		if !seen[c] {
			out = append(out, c)
		}
	}
	return out
}
