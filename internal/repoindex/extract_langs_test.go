package repoindex

import "testing"

// TestFunctionsInTSSource_ExtractsDeclarationArrowAndMethod covers the three
// shapes duplicate detection needs from TS/JS: a function declaration, an
// arrow function assigned to a const, and a class method. Line ranges are
// asserted so downstream gates can report accurate locations.
func TestFunctionsInTSSource_ExtractsDeclarationArrowAndMethod(t *testing.T) {
	src := []byte(`function add(a: number, b: number): number {
  return a + b;
}

const multiply = (a: number, b: number) => {
  return a * b;
};

class Calc {
  subtract(a: number, b: number) {
    return a - b;
  }
}
`)
	got := FunctionsInTSSource("calc.ts", src)

	want := map[string][2]int{
		"add":      {1, 3},
		"multiply": {5, 7},
		"subtract": {10, 12},
	}
	if len(got) != len(want) {
		names := make([]string, len(got))
		for i, f := range got {
			names[i] = f.Name
		}
		t.Fatalf("got %d funcs %v, want %d: %v", len(got), names, len(want), want)
	}
	for _, f := range got {
		lines, ok := want[f.Name]
		if !ok {
			t.Errorf("unexpected function %q", f.Name)
			continue
		}
		if f.StartLine != lines[0] || f.EndLine != lines[1] {
			t.Errorf("%s: got lines %d-%d, want %d-%d", f.Name, f.StartLine, f.EndLine, lines[0], lines[1])
		}
	}
}

// TestFunctionsInPythonSource_NestedDef documents this package's choice for
// nested defs: both the outer and inner function are captured as separate
// FuncInfo entries (see python.go's package doc for why), so a helper
// duplicated across two outer functions is still caught by the dup gate.
func TestFunctionsInPythonSource_NestedDef(t *testing.T) {
	src := []byte(`def outer(xs):
    def inner(x):
        return x * 2
    total = 0
    for x in xs:
        total += inner(x)
    return total

def other():
    return 1
`)
	got := FunctionsInPythonSource("sample.py", src)

	byName := map[string]FuncInfo{}
	for _, f := range got {
		byName[f.Name] = f
	}
	if len(got) != 3 {
		t.Fatalf("got %d funcs, want 3 (outer, inner, other)", len(got))
	}

	outer, ok := byName["outer"]
	if !ok {
		t.Fatal("outer not captured")
	}
	if outer.StartLine != 1 || outer.EndLine != 7 {
		t.Errorf("outer: got lines %d-%d, want 1-7", outer.StartLine, outer.EndLine)
	}

	inner, ok := byName["inner"]
	if !ok {
		t.Fatal("inner not captured (documented choice: nested defs ARE captured)")
	}
	if inner.StartLine != 2 || inner.EndLine != 3 {
		t.Errorf("inner: got lines %d-%d, want 2-3", inner.StartLine, inner.EndLine)
	}

	other, ok := byName["other"]
	if !ok {
		t.Fatal("other not captured")
	}
	if other.StartLine != 9 || other.EndLine != 10 {
		t.Errorf("other: got lines %d-%d, want 9-10", other.StartLine, other.EndLine)
	}
}

// TestFunctionsInTSSource_NearDuplicateAcrossFiles verifies the tokenizer +
// shingle pipeline in tsjs.go feeds the package's Jaccard helper correctly:
// two structurally identical functions (renamed identifiers/literals) across
// two source files should score as near-duplicates.
func TestFunctionsInTSSource_NearDuplicateAcrossFiles(t *testing.T) {
	a := FunctionsInTSSource("a.ts", []byte(`function sumAll(xs) {
  let total = 0;
  for (let i = 0; i < xs.length; i++) {
    total += xs[i] * 2;
  }
  return total;
}
`))
	b := FunctionsInTSSource("b.ts", []byte(`function addUp(ys) {
  let sum = 0;
  for (let j = 0; j < ys.length; j++) {
    sum += ys[j] * 3;
  }
  return sum;
}
`))
	c := FunctionsInTSSource("c.ts", []byte(`function isEven(n) {
  return n % 2 === 0;
}
`))
	if len(a) != 1 || len(b) != 1 || len(c) != 1 {
		t.Fatalf("expected 1 func per source, got %d %d %d", len(a), len(b), len(c))
	}

	sim := Jaccard(a[0].Shingles, b[0].Shingles)
	if sim < 0.8 {
		t.Errorf("sumAll vs addUp Jaccard = %.2f, want >= 0.8", sim)
	}

	simUnrelated := Jaccard(a[0].Shingles, c[0].Shingles)
	if simUnrelated >= 0.8 {
		t.Errorf("sumAll vs isEven Jaccard = %.2f, want < 0.8 (unrelated)", simUnrelated)
	}
}
