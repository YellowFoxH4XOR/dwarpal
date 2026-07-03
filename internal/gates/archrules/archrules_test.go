package archrules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// writeFile creates dir/path with contents, creating parent dirs as needed.
func writeFile(t *testing.T, root, path, contents string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

const dbRepoSrc = `package repo

import "database/sql"

func Open() *sql.DB {
	db, _ := sql.Open("postgres", "")
	return db
}
`

// handlerSrc has db.Query on line 8 (added) and a second call on line 9
// (not added), used to verify only added-line calls are flagged.
const handlerSrc = `package web

func Handle(db *sql.DB) {
	_ = 1
	_ = 2
	_ = 3
	_ = 4
	rows, _ := db.Query("select 1") // line 8
	rows2, _ := db.Query("select 2") // line 9, not added
	_ = rows
	_ = rows2
}
`

func repoRule(forbiddenOutside []string) Rule {
	return Rule{
		ID:               "db-through-repo-layer",
		Description:      "database calls must go through the repo layer",
		Language:         "go",
		Matches:          "sql.Open|db.Query",
		ForbiddenOutside: forbiddenOutside,
		Severity:         "error",
	}
}

func TestArchRules_ForbiddenCallOnAddedLineFlagged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/repo/db.go", dbRepoSrc)
	writeFile(t, root, "web/handler.go", handlerSrc)

	g := New(root, []Rule{repoRule([]string{"internal/repo/**"})})

	d := &gitio.Diff{Files: []gitio.FileChange{
		{Path: "internal/repo/db.go", Kind: gitio.KindAdded, AddedLines: []gitio.Line{{Number: 6, Text: `db, _ := sql.Open("postgres", "")`}}},
		{Path: "web/handler.go", Kind: gitio.KindAdded, AddedLines: []gitio.Line{{Number: 8, Text: `rows, _ := db.Query("select 1")`}}},
	}}

	fs, err := g.Run(context.Background(), d, engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 1 {
		t.Fatalf("expected exactly one finding, got %d: %+v", len(fs), fs)
	}
	f := fs[0]
	if f.File != "web/handler.go" || f.Line != 8 {
		t.Errorf("expected finding at web/handler.go:8, got %s:%d", f.File, f.Line)
	}
	if f.Gate != gateID || f.RuleID != "db-through-repo-layer" {
		t.Errorf("unexpected gate/rule id: %+v", f)
	}
	if f.Severity.Blocking() != true {
		t.Errorf("expected error severity to block, got %s", f.Severity)
	}
}

// A matching call on a non-added line (unmodified/context line) must not be
// flagged — the gate only checks lines this change actually added.
func TestArchRules_NonAddedLineNotFlagged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "web/handler.go", handlerSrc)

	g := New(root, []Rule{repoRule([]string{"internal/repo/**"})})

	// Only line 9 (the second db.Query, NOT line 8) is reported as added.
	d := &gitio.Diff{Files: []gitio.FileChange{
		{Path: "web/handler.go", Kind: gitio.KindModified, AddedLines: []gitio.Line{{Number: 9, Text: `rows2, _ := db.Query("select 2")`}}},
	}}

	fs, err := g.Run(context.Background(), d, engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 1 || fs[0].Line != 9 {
		t.Fatalf("expected exactly one finding at line 9, got %+v", fs)
	}
}

func TestArchRules_AllowedLocationNotFlagged(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/repo/db.go", dbRepoSrc)

	g := New(root, []Rule{repoRule([]string{"internal/repo/**"})})

	d := &gitio.Diff{Files: []gitio.FileChange{
		{Path: "internal/repo/db.go", Kind: gitio.KindAdded, AddedLines: []gitio.Line{{Number: 6, Text: `db, _ := sql.Open("postgres", "")`}}},
	}}

	fs, err := g.Run(context.Background(), d, engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("expected no findings for allowed location, got %+v", fs)
	}
}

func TestArchRules_InvalidRegexpErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "web/handler.go", handlerSrc)

	g := New(root, []Rule{{
		ID:               "bad",
		Language:         "go",
		Matches:          "(unterminated",
		ForbiddenOutside: []string{"internal/repo/**"},
	}})

	d := &gitio.Diff{Files: []gitio.FileChange{
		{Path: "web/handler.go", Kind: gitio.KindModified, AddedLines: []gitio.Line{{Number: 8}}},
	}}

	_, err := g.Run(context.Background(), d, engine.NoIndex{})
	if err == nil {
		t.Fatal("expected error for invalid regexp, got nil")
	}
}

func TestArchRules_NonGoLanguageSkipped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "web/handler.go", handlerSrc)

	g := New(root, []Rule{{
		ID:               "js-rule",
		Language:         "javascript",
		Matches:          "db.Query",
		ForbiddenOutside: []string{"internal/repo/**"},
	}})

	d := &gitio.Diff{Files: []gitio.FileChange{
		{Path: "web/handler.go", Kind: gitio.KindModified, AddedLines: []gitio.Line{{Number: 8}}},
	}}

	fs, err := g.Run(context.Background(), d, engine.NoIndex{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("non-go rule should be skipped silently, got %+v", fs)
	}
}
