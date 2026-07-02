package aipatterns

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

func fileWith(path string, lines ...gitio.Line) gitio.FileChange {
	return gitio.FileChange{Path: path, Kind: gitio.KindModified, AddedLines: lines}
}

func run(t *testing.T, g *Gate, files ...gitio.FileChange) []string {
	t.Helper()
	fs, err := g.Run(context.Background(), &gitio.Diff{Files: files}, engine.NoIndex{})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(fs))
	for i, f := range fs {
		ids[i] = f.RuleID
	}
	return ids
}

func has(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func TestSuppressionDetected(t *testing.T) {
	g := New(nil)
	ids := run(t, g, fileWith("a.ts",
		gitio.Line{Number: 5, Text: "// eslint-disable-next-line no-console"},
		gitio.Line{Number: 6, Text: "const x = 1 // @ts-ignore"},
	))
	if !has(ids, "no-new-lint-suppressions") {
		t.Fatalf("expected suppression finding, got %v", ids)
	}
}

func TestSecretsDetected(t *testing.T) {
	g := New(nil)
	ids := run(t, g,
		fileWith("k.pem", gitio.Line{Number: 1, Text: "-----BEGIN RSA PRIVATE KEY-----"}),
		fileWith("c.go", gitio.Line{Number: 2, Text: `awsKey := "AKIAIOSFODNN7EXAMPLE"`}),
		fileWith("cfg.py", gitio.Line{Number: 3, Text: `api_key = "sk_live_abcdef0123456789xyz"`}),
	)
	if !has(ids, "no-hardcoded-secrets/private-key") ||
		!has(ids, "no-hardcoded-secrets/aws-key") ||
		!has(ids, "no-hardcoded-secrets/assigned-literal") {
		t.Fatalf("expected all three secret rules to fire, got %v", ids)
	}
}

// Clean code produces no findings — guards against false positives.
func TestNoFalsePositiveOnCleanCode(t *testing.T) {
	g := New(nil)
	ids := run(t, g, fileWith("clean.go",
		gitio.Line{Number: 1, Text: "func Add(a, b int) int { return a + b }"},
		gitio.Line{Number: 2, Text: `logger.Info("processing request")`},
	))
	if len(ids) != 0 {
		t.Fatalf("clean code should yield no findings, got %v", ids)
	}
}

func TestSQLConcatHeuristic(t *testing.T) {
	g := New(nil)
	ids := run(t, g, fileWith("db.go",
		gitio.Line{Number: 1, Text: `q := "SELECT * FROM users WHERE id = " + userID`},
	))
	if !has(ids, "no-sql-concat") {
		t.Fatalf("expected no-sql-concat, got %v", ids)
	}
	// Parameterized query must NOT trip the heuristic.
	clean := run(t, g, fileWith("db.go",
		gitio.Line{Number: 1, Text: `db.Query("SELECT * FROM users WHERE id = ?", userID)`},
	))
	if has(clean, "no-sql-concat") {
		t.Fatalf("parameterized query false-positive: %v", clean)
	}
}

func TestBroadCatchHeuristic(t *testing.T) {
	g := New(nil)
	py := run(t, g, fileWith("a.py", gitio.Line{Number: 1, Text: "    except:"}))
	if !has(py, "no-broad-catch") {
		t.Fatalf("expected no-broad-catch for bare except, got %v", py)
	}
	js := run(t, g, fileWith("a.js", gitio.Line{Number: 1, Text: "try { x() } catch (e) {}"}))
	if !has(js, "no-broad-catch") {
		t.Fatalf("expected no-broad-catch for empty catch, got %v", js)
	}
}

// A disabled rule must not fire (config disable_rules).
func TestDisableRule(t *testing.T) {
	g := New([]string{"no-new-lint-suppressions"})
	ids := run(t, g, fileWith("a.go", gitio.Line{Number: 1, Text: "//nolint"}))
	if has(ids, "no-new-lint-suppressions") {
		t.Fatalf("disabled rule fired: %v", ids)
	}
}
