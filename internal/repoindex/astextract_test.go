package repoindex

import "testing"

// TS class method extracted with accurate lines via the tree-sitter path.
func TestASTExtract_TSClassMethod(t *testing.T) {
	src := []byte(`class Svc {
  handle(req: Request): void {
    run(req);
  }
}
const twice = (n: number) => n * 2;
`)
	funcs, ok := functionsViaAST("svc.ts", src)
	if !ok {
		t.Fatal("AST extraction should succeed on valid TS")
	}
	byName := map[string]FuncInfo{}
	for _, f := range funcs {
		byName[f.Name] = f
	}
	m, present := byName["handle"]
	if !present || m.StartLine != 2 || m.EndLine != 4 {
		t.Fatalf("handle method lines wrong: %+v (present=%v)", m, present)
	}
	// NOTE: the typed arrow (`const twice = (n: number) => ...`) is the
	// documented gotreesitter TS grammar gap (design D7) — the AST tier
	// misses it, so no assertion on `twice` here.
}

// Broken TS degrades: tolerant parsing yields a Partial tree whose queries
// capture nothing, and the heuristic supplement finds nothing either — the
// result is an empty inventory, never a crash (ast-engine degradation
// contract, tolerant+supplement form).
func TestASTExtract_BrokenDegrades(t *testing.T) {
	funcs, _ := functionsViaAST("bad.ts", []byte("function {{{ ???"))
	for _, f := range funcs {
		if f.Name == "" {
			t.Fatalf("broken source must not yield nameless phantom functions: %+v", funcs)
		}
	}
}

// Cross-tier comparability: a duplicate must still score high when one side
// was extracted via AST and the other via the heuristic tier.
func TestASTExtract_ShinglesComparableAcrossTiers(t *testing.T) {
	src := []byte(`function sum(xs) {
  let t = 0;
  for (const x of xs) { t += x; }
  return t;
}
`)
	astFuncs, ok := functionsViaAST("a.js", src)
	if !ok || len(astFuncs) == 0 {
		t.Fatal("AST extraction failed")
	}
	heurFuncs := FunctionsInTSSource("b.js", src)
	if len(heurFuncs) == 0 {
		t.Fatal("heuristic extraction failed")
	}
	// Tiers differ slightly at the body boundary (AST range includes the
	// declaration header; the heuristic starts at the brace), so identical
	// code scores ~0.76 rather than 1.0 across tiers. 0.7 asserts they remain
	// comparable without pretending boundary parity.
	if sim := Jaccard(astFuncs[0].Shingles, heurFuncs[0].Shingles); sim < 0.7 {
		t.Fatalf("cross-tier similarity too low: %.2f", sim)
	}
}
