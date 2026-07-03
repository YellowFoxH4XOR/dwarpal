package archrules

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// The core promise of this change: a layering rule enforces on Python/TS/JS,
// not just Go. A forbidden call OUTSIDE the allowed layer must flag; the SAME
// call INSIDE it must not.
func TestArchRules_MultiLanguageEnforcement(t *testing.T) {
	cases := []struct {
		name, lang, path, src, matches, added string
		line                                  int
	}{
		{
			name:    "python db call outside repo layer",
			lang:    "python",
			path:    "web/handler.py",
			src:     "def handle():\n    rows = db.query(\"select 1\")\n    return rows\n",
			matches: `db\.query|sql\.connect`,
			added:   `    rows = db.query("select 1")`,
			line:    2,
		},
		{
			name:    "typescript db call outside repo layer",
			lang:    "typescript",
			path:    "src/web/handler.ts",
			src:     "export function handle() {\n  const rows = db.query(\"select 1\");\n  return rows;\n}\n",
			matches: `db\.query`,
			added:   `  const rows = db.query("select 1");`,
			line:    2,
		},
		{
			name:    "javascript db call outside repo layer",
			lang:    "javascript",
			path:    "web/handler.js",
			src:     "function handle() {\n  const rows = db.query(\"select 1\");\n  return rows;\n}\n",
			matches: `db\.query`,
			added:   `  const rows = db.query("select 1");`,
			line:    2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, tc.path, tc.src)
			rule := Rule{
				ID: "db-through-repo", Description: "db calls go through the repo layer",
				Language: tc.lang, Matches: tc.matches,
				ForbiddenOutside: []string{"**/repo/**"}, Severity: "error",
			}
			g := New(root, []Rule{rule})
			d := &gitio.Diff{Files: []gitio.FileChange{
				{Path: tc.path, Kind: gitio.KindAdded, AddedLines: []gitio.Line{{Number: tc.line, Text: tc.added}}},
			}}
			fs, err := g.Run(context.Background(), d, engine.NoIndex{})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(fs) != 1 {
				t.Fatalf("want 1 finding for a forbidden %s call, got %d: %+v", tc.lang, len(fs), fs)
			}
			if fs[0].File != tc.path || fs[0].Line != tc.line {
				t.Errorf("finding at %s:%d, want %s:%d", fs[0].File, fs[0].Line, tc.path, tc.line)
			}
		})
	}
}

// The same call INSIDE the allowed layer must NOT flag — the boundary is what's
// enforced, not the call itself.
func TestArchRules_MultiLanguageAllowedInsideLayer(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "repo/db.py", "def get():\n    return db.query(\"select 1\")\n")
	rule := Rule{
		ID: "db-through-repo", Description: "db calls go through the repo layer",
		Language: "python", Matches: `db\.query`,
		ForbiddenOutside: []string{"**/repo/**"}, Severity: "error",
	}
	g := New(root, []Rule{rule})
	d := &gitio.Diff{Files: []gitio.FileChange{
		{Path: "repo/db.py", Kind: gitio.KindAdded, AddedLines: []gitio.Line{{Number: 2, Text: `    return db.query("select 1")`}}},
	}}
	fs, err := g.Run(context.Background(), d, engine.NoIndex{})
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("call inside the allowed layer must not flag, got %+v", fs)
	}
}

// A rule targeting a language Dwarpal can't parse must fail LOUDLY (config
// error), not silently do nothing — the whole point of this change. A silently
// unenforced layering rule is worse than no rule.
func TestArchRules_UnsupportedLanguageErrors(t *testing.T) {
	g := New(t.TempDir(), []Rule{{ID: "r", Language: "ruby", Matches: "x"}})
	_, err := g.Run(context.Background(), &gitio.Diff{Files: []gitio.FileChange{
		{Path: "a.rb", AddedLines: []gitio.Line{{Number: 1, Text: "x"}}},
	}}, engine.NoIndex{})
	if err == nil {
		t.Fatal("a rule for an unsupported language must return an error, not skip silently")
	}
}
