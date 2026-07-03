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

// docsSlug maps gate/rule identifiers to their page slug. Pages are one-per-
// rule except architecture_rules and plugins, whose rule IDs are user-defined
// and share a generic page each.
func docsSlug(gate, ruleID string) string {
	switch gate {
	case "architecture_rules":
		return "architecture-rules"
	default:
		if strings.HasPrefix(gate, "plugin/") || gate == "plugin" {
			return "plugin-exit-nonzero"
		}
	}
	if gate == "" || ruleID == "" {
		return ""
	}
	return strings.ReplaceAll(gate, "_", "-") + "-" + strings.ReplaceAll(ruleID, "/", "-")
}
