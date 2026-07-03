package astengine

import "testing"

func TestSupports(t *testing.T) {
	for path, want := range map[string]bool{
		"a.ts": true, "b.tsx": true, "c.js": true, "d.jsx": true, "e.py": true,
		"f.go": false, // Go stays on the stdlib go/parser path
		"g.rb": false, "h.txt": false,
	} {
		if got := Supports(path); got != want {
			t.Errorf("Supports(%s) = %v, want %v", path, got, want)
		}
	}
}

func TestParseAndQuery_TS(t *testing.T) {
	src := []byte(`function sum(xs: number[]): number {
  let t = 0;
  for (const x of xs) t += x;
  return t;
}`)
	tree, err := Parse("a.ts", src)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := tree.Query(`(function_declaration name: (identifier) @fn)`)
	if err != nil {
		t.Fatal(err)
	}
	if len(caps) != 1 || caps[0].Text != "sum" || caps[0].StartLine != 1 {
		t.Fatalf("want sum@1, got %+v", caps)
	}
}

func TestParseAndQuery_Python(t *testing.T) {
	src := []byte("def add(a, b):\n    return a + b\n")
	tree, err := Parse("b.py", src)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := tree.Query(`(function_definition name: (identifier) @fn)`)
	if err != nil {
		t.Fatal(err)
	}
	if len(caps) != 1 || caps[0].Text != "add" {
		t.Fatalf("want add, got %+v", caps)
	}
}

// A file whose language is outside the registry must error so callers fall
// through to heuristic behavior — the ast-engine spec's degradation contract.
func TestParse_UnsupportedLanguage(t *testing.T) {
	if _, err := Parse("x.rb", []byte("def x; end")); err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

// Broken source parses tolerantly (design D7): the tree is marked Partial so
// callers supplement with heuristics; queries over it must not invent matches.
func TestParse_BrokenSourceIsPartial(t *testing.T) {
	tree, err := Parse("bad.py", []byte("def broken(:::\n  ????"))
	if err != nil {
		return // full degradation is also acceptable
	}
	if !tree.Partial {
		t.Fatal("broken source must be marked Partial")
	}
	caps, qerr := tree.Query(`(function_definition name: (identifier) @fn)`)
	if qerr != nil {
		t.Fatal(qerr)
	}
	for _, c := range caps {
		if c.Text == "" {
			t.Fatalf("phantom capture from broken source: %+v", caps)
		}
	}
}
