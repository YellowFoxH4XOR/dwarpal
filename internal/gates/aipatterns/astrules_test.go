package aipatterns

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// writeAndChange writes src into a temp root and returns the root plus a
// FileChange marking every line as added — the AST tier's real input shape.
func writeAndChange(t *testing.T, name, src string) (string, gitio.FileChange) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var lines []gitio.Line
	n := 1
	for _, l := range splitLines(src) {
		lines = append(lines, gitio.Line{Number: n, Text: l})
		n++
	}
	return root, gitio.FileChange{Path: name, Kind: gitio.KindAdded, AddedLines: lines}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func runOn(t *testing.T, root string, fc gitio.FileChange) []string {
	t.Helper()
	fs, err := New(root, nil).Run(context.Background(), &gitio.Diff{Files: []gitio.FileChange{fc}}, engine.NoIndex{})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, f := range fs {
		ids = append(ids, f.RuleID)
	}
	return ids
}

func TestAST_TSTemplateLiteralSQLFlagged(t *testing.T) {
	root, fc := writeAndChange(t, "db.ts", "const q = `SELECT * FROM t WHERE id = ${id}`;\nrun(q);\n")
	if !has(runOn(t, root, fc), "no-sql-concat") {
		t.Fatal("template-literal SQL interpolation should be flagged")
	}
}

func TestAST_TSPlainTemplateNotFlagged(t *testing.T) {
	// SQL keywords but NO interpolation — a constant query string is fine.
	root, fc := writeAndChange(t, "db.ts", "const q = `SELECT * FROM t WHERE id = ?`;\nrun(q, id);\n")
	if has(runOn(t, root, fc), "no-sql-concat") {
		t.Fatal("constant template literal must not be flagged")
	}
}

func TestAST_EmptyCatchFlagged(t *testing.T) {
	root, fc := writeAndChange(t, "svc.ts", "try {\n  run();\n} catch (e) {\n}\n")
	if !has(runOn(t, root, fc), "no-broad-catch") {
		t.Fatal("empty catch should be flagged")
	}
}

func TestAST_LoggedCatchNotFlagged(t *testing.T) {
	root, fc := writeAndChange(t, "svc.ts", "try {\n  run();\n} catch (e) {\n  logger.error(e);\n}\n")
	if has(runOn(t, root, fc), "no-broad-catch") {
		t.Fatal("logged catch must not be flagged")
	}
}

func TestAST_PythonBarePassFlagged(t *testing.T) {
	root, fc := writeAndChange(t, "app.py", "try:\n    run()\nexcept:\n    pass\n")
	if !has(runOn(t, root, fc), "no-broad-catch") {
		t.Fatal("bare except: pass should be flagged")
	}
}

func TestAST_PythonHandledExceptNotFlagged(t *testing.T) {
	root, fc := writeAndChange(t, "app.py", "try:\n    run()\nexcept ValueError as e:\n    log.warning(e)\n")
	if has(runOn(t, root, fc), "no-broad-catch") {
		t.Fatal("handled except must not be flagged")
	}
}

// Suppression: the regex heuristic must not double-report a file the AST tier
// handled (design D5).
func TestAST_NoDoubleReporting(t *testing.T) {
	root, fc := writeAndChange(t, "db.ts", "const q = `SELECT * FROM t WHERE id = ${id}`;\n")
	count := 0
	for _, id := range runOn(t, root, fc) {
		if id == "no-sql-concat" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("want exactly one no-sql-concat finding, got %d", count)
	}
}
