package analyze

import (
	"os"
	"path/filepath"
	"strings"
)

// detectCoverage looks for a coverage artifact the diff-coverage gate could
// consume, checking the conventional locations for the supported formats.
func detectCoverage(root string) string {
	for _, p := range []string{
		"coverage/lcov.info", "lcov.info", "coverage.xml",
		"cover.out", "coverage/cobertura.xml", "coverage.lcov",
	} {
		if fileExists(filepath.Join(root, p)) {
			return p
		}
	}
	return ""
}

// detectTools reports security tools present in the repo that make good
// plugin gates — inferred from their config files so the suggestion only
// appears when the team already uses the tool.
func detectTools(root string) []string {
	var tools []string
	markers := map[string][]string{
		"gitleaks": {".gitleaks.toml", "gitleaks.toml"},
		"semgrep":  {".semgrep.yml", ".semgrep.yaml", "semgrep.yml"},
		"trivy":    {"trivy.yaml", ".trivyignore"},
		"bandit":   {".bandit", "bandit.yaml"},
	}
	for tool, files := range markers {
		for _, f := range files {
			if fileExists(filepath.Join(root, f)) {
				tools = append(tools, tool)
				break
			}
		}
	}
	return tools
}

// detectBranchPrefixes samples recent branch names for prefixes worth feeding
// provenance/branch_prefixes (an agent wrapper's convention, a team's flow).
func detectBranchPrefixes(root string) []string {
	out, err := gitOut(root, "for-each-ref", "--format=%(refname:short)", "--count=100", "refs/heads")
	if err != nil {
		return nil
	}
	seen := map[string]int{}
	for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
		if i := strings.Index(name, "/"); i > 0 {
			seen[name[:i+1]]++
		}
	}
	var prefixes []string
	for p, n := range seen {
		if n >= 2 { // a prefix used more than once is a convention
			prefixes = append(prefixes, p)
		}
	}
	return prefixes
}

// detectLayering surfaces package/directory names that commonly denote a
// layer boundary worth an architecture_rule (e.g. a repo/data-access layer).
func detectLayering(root string) []string {
	var hints []string
	candidates := map[string]string{
		"internal/repo":  "DB/data access likely belongs here — consider forbidding sql/db calls outside it",
		"internal/store": "storage layer — consider a boundary rule",
		"repository":     "repository layer — consider a boundary rule",
		"dal":            "data-access layer — consider a boundary rule",
		"db":             "db package — consider forbidding direct DB calls elsewhere",
	}
	for dir, hint := range candidates {
		if dirExists(filepath.Join(root, dir)) {
			hints = append(hints, dir+": "+hint)
		}
	}
	return hints
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
