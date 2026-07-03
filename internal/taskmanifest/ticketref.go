// Package taskmanifest extracts task-identifying metadata (currently: ticket
// references) from branch names, so gates and reports can link findings back
// to the work item that motivated them without requiring extra config.
package taskmanifest

import (
	"regexp"
	"strings"
)

// ticketPattern matches an alpha(2-10)-digit(1-6) ticket token anchored at the
// start of a branch's final path segment (e.g. "AUTH-42" in "agent/AUTH-42-fix").
// The trailing (?:[^0-9a-z]|$) requires the digit run to end at a
// non-alphanumeric boundary or end-of-string, so a trailing letter glued onto
// the digits (as in ".../here-123x") is correctly rejected rather than
// truncated into a false match.
var ticketPattern = regexp.MustCompile(`(?i)^([a-z]{2,10})-([0-9]{1,6})(?:[^0-9a-z]|$)`)

// TicketFromBranch extracts a ticket ID such as AUTH-42 from a branch name
// like agent/AUTH-42-password-reset. The ticket token must sit at the start of
// the last '/'-separated segment; if it doesn't (or none is found), "" is
// returned rather than guessing at an ambiguous match. Input is matched
// case-insensitively; the result is always upper-cased.
func TicketFromBranch(branch string) string {
	seg := branch
	if i := strings.LastIndex(branch, "/"); i != -1 {
		seg = branch[i+1:]
	}
	m := ticketPattern.FindStringSubmatch(seg)
	if m == nil {
		return ""
	}
	return strings.ToUpper(m[1]) + "-" + m[2]
}
