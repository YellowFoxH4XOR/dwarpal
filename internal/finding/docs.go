package finding

import "strings"

// docsBase is where rule documentation lives. GitHub-rendered markdown in the
// repo: real pages with zero hosting infrastructure. When a dwarpal.dev docs
// site exists, only this constant changes.
const docsBase = "https://github.com/YellowFoxH4XOR/dwarpal/blob/main/docs/rules/"

// DocsURL returns the documentation page for a finding's gate + rule. The
// engine fills this into any finding whose gate left DocsURL empty, so every
// finding ships a working link without each gate repeating the mapping.
func DocsURL(gate, ruleID string) string {
	slug := docsSlug(gate, ruleID)
	if slug == "" {
		return ""
	}
	return docsBase + slug + ".md"
}

// docsSlug maps a gate/rule identifier to its page slug (one page per rule).
func docsSlug(gate, ruleID string) string {
	if gate == "" || ruleID == "" {
		return ""
	}
	return strings.ReplaceAll(gate, "_", "-") + "-" + strings.ReplaceAll(ruleID, "/", "-")
}
