package aipatterns

import (
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

func TestEntropyFindings(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantFlag bool
	}{
		// --- should flag: real-shaped secrets ---
		{
			name:     "base64 encoded secret token",
			line:     `token := "c2VjcmV0LXRva2VuLWFiY2RlZmdoaWprbG1ub3A="`,
			wantFlag: true,
		},
		{
			name:     "32+ char random alnum api key",
			line:     `apiSecret := "aB3xQ9mK2pL7vN4sT8wZ1yR6uE5dF0gH9jC8"`,
			wantFlag: true,
		},
		{
			name:     "random alnum bare token without quotes",
			line:     `os.Setenv("STRIPE_KEY", sk9F2mQwXz7bLpN4vR8tYh3JcA6dEoU1)`,
			wantFlag: true,
		},
		{
			name:     "random secret assigned to a config field",
			line:     `cfg.SigningSecret = "kX7pQ2mZ9vR4tY6wB1nJ8cF3sL0dH5eA"`,
			wantFlag: true,
		},
		{
			name:     "high entropy secret embedded in JSON",
			line:     `{"client_secret": "9fQ2xLpM7vZkR3wYtB6nJ1cA8sD4eH0g"}`,
			wantFlag: true,
		},
		{
			name:     "random session token in header literal",
			line:     `headers["X-Session-Token"] = "hT9pL2qWxZ7vNcR4mB8kJ1sD6yA3fU0e"`,
			wantFlag: true,
		},
		// --- should NOT flag ---
		{
			name:     "normal prose comment",
			line:     `// Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod`,
			wantFlag: false,
		},
		{
			name:     "long camelCase function identifier",
			line:     `func calculateMonthlyInterestRateForLoan(principal float64) float64 {`,
			wantFlag: false,
		},
		{
			name:     "long camelCase type instantiation identifier",
			line:     `repo := userAccountRepositoryImplementation{}`,
			wantFlag: false,
		},
		{
			name:     "long snake_case constant identifier",
			line:     `// README_INSTALLATION_INSTRUCTIONS_FOR_DEVELOPERS applies here`,
			wantFlag: false,
		},
		{
			name:     "hex color code",
			line:     `color := "#1a2b3c4d5e6f7a8b9c0d1e2f"`,
			wantFlag: false,
		},
		{
			name:     "url without embedded token",
			line:     `// see http://www.example.com/docs/getting-started/installation for details`,
			wantFlag: false,
		},
		{
			name:     "url with long path but no random token",
			line:     `img.Src = "https://cdn.example.com/assets/images/logo-large.png"`,
			wantFlag: false,
		},
		{
			name:     "short base64-looking string",
			line:     `shortToken := "dGVzdA=="`,
			wantFlag: false,
		},
		{
			name:     "long descriptive snake_case config identifier",
			line:     `const configDatabaseConnectionPoolSettings = loadConfig()`,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := gitio.FileChange{
				Path:       "example.go",
				AddedLines: []gitio.Line{{Number: 1, Text: tt.line}},
			}
			got := EntropyFindings(f)
			if tt.wantFlag && len(got) == 0 {
				t.Fatalf("expected a finding for line %q, got none", tt.line)
			}
			if !tt.wantFlag && len(got) != 0 {
				t.Fatalf("expected no finding for line %q, got %+v", tt.line, got)
			}
			if tt.wantFlag {
				fnd := got[0]
				if fnd.Gate != gateID {
					t.Errorf("Gate = %q, want %q", fnd.Gate, gateID)
				}
				if fnd.RuleID != "no-hardcoded-secrets/entropy" {
					t.Errorf("RuleID = %q, want no-hardcoded-secrets/entropy", fnd.RuleID)
				}
				if fnd.File != "example.go" {
					t.Errorf("File = %q, want example.go", fnd.File)
				}
				if fnd.Line != 1 {
					t.Errorf("Line = %d, want 1", fnd.Line)
				}
				if fnd.Suggestion == "" || fnd.RetryHint == "" {
					t.Errorf("expected non-empty Suggestion and RetryHint about secret managers, got %+v", fnd)
				}
			}
		})
	}
}

// EntropyFindings must only look at AddedLines — pre-existing secrets in
// context are not this commit's concern (matches the rest of Gate 3).
func TestEntropyFindings_OnlyAddedLines(t *testing.T) {
	f := gitio.FileChange{Path: "x.go"} // no AddedLines
	if got := EntropyFindings(f); len(got) != 0 {
		t.Fatalf("expected no findings with no added lines, got %+v", got)
	}
}

// Regression: Dwarpal's own hook scripts contain a repo URL in a comment
// ("# Managed by Dwarpal (https://github.com/YellowFoxH4XOR/dwarpal)...")
// whose path run scores above the entropy threshold. URLs and path-like
// tokens must never be flagged — this false positive blocked real commits.
func TestEntropy_URLAndPathNotFlagged(t *testing.T) {
	f := gitio.FileChange{Path: "hook.sh", AddedLines: []gitio.Line{
		{Number: 2, Text: `# Managed by Dwarpal (https://github.com/YellowFoxH4XOR/dwarpal). Do not edit.`},
		{Number: 3, Text: `include "internal/gates/aipatterns/SomethingLongEnough/here.go"`},
	}}
	if fs := EntropyFindings(f); len(fs) != 0 {
		t.Fatalf("URL/path tokens must not be flagged, got %+v", fs)
	}
}

// Regression: long camelCase identifiers scored above the entropy threshold
// and BLOCKED a real commit (TestCache_WarmRebuildPreservesShingles, 4.2
// bits/char). Digit-less tokens are identifiers, not secrets.
func TestEntropy_LongIdentifiersNotFlagged(t *testing.T) {
	f := gitio.FileChange{Path: "cache_test.go", AddedLines: []gitio.Line{
		{Number: 11, Text: "func TestCache_WarmRebuildPreservesShingles(t *testing.T) {"},
		{Number: 12, Text: "ConventionsOnlyExtractsNothingButImports := true"},
	}}
	if fs := EntropyFindings(f); len(fs) != 0 {
		t.Fatalf("identifiers must not be flagged, got %+v", fs)
	}
}
