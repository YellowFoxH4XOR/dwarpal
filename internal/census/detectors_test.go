package census

import (
	"reflect"
	"testing"
)

// The deadcode identity must be package-qualified (Path.Name), NOT file:line —
// otherwise a dead symbol's key would churn every time an unrelated line shifts
// above it, producing phantom ratchet failures on innocent diffs.
func TestParseDeadcodeJSON_identityIsPositionIndependent(t *testing.T) {
	out := []byte(`[
	  {"Name":"main","Path":"example.com/app/cmd","Funcs":[
	    {"Name":"buildGates","Position":{"File":"cmd/gates.go","Line":73,"Col":6}}
	  ]},
	  {"Name":"engine","Path":"example.com/app/engine","Funcs":[
	    {"Name":"Run","Position":{"File":"engine/engine.go","Line":112,"Col":6}}
	  ]}
	]`)
	count, items, err := parseDeadcodeJSON(out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	want := []string{"example.com/app/cmd.buildGates", "example.com/app/engine.Run"}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %v, want %v (no file/line in identity)", items, want)
	}
}

// No dead code means deadcode emits nothing; that must read as a clean zero,
// not a parse error (which the census would escalate to fail-loud).
func TestParseDeadcodeJSON_emptyIsZeroNotError(t *testing.T) {
	count, items, err := parseDeadcodeJSON([]byte("  \n"))
	if err != nil || count != 0 || items != nil {
		t.Fatalf("empty output: got (%d, %v, %v), want (0, nil, nil)", count, items, err)
	}
}

// The grep-line parser must (a) strip line:col so identity survives code
// movement and (b) ignore banner/summary lines so they don't inflate the count.
func TestParseGrepLines_stripsPositionAndIgnoresNoise(t *testing.T) {
	out := []byte(
		"internal/x.go:10:6: func foo is unused (U1000)\n" +
			"internal/y.go:3: unused function 'bar' (60% confidence)\n" +
			"Found 2 errors.\n" + // summary line: no file:line prefix → ignored
			"\n") // blank → ignored
	count, items, err := parseGrepLines(out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2 (summary/blank must not count)", count)
	}
	want := []string{
		"internal/x.go: func foo is unused (U1000)",
		"internal/y.go: unused function 'bar' (60% confidence)",
	}
	if !reflect.DeepEqual(items, want) {
		t.Fatalf("items = %v, want %v", items, want)
	}
}

func TestLookup_unknownIsReported(t *testing.T) {
	if _, ok := Lookup("no-such-detector"); ok {
		t.Fatal("Lookup returned ok for an unknown detector")
	}
	d, ok := Lookup("deadcode")
	if !ok || d.Scope != WholeRepo {
		t.Fatalf("deadcode: ok=%v scope=%v, want true whole-repo", ok, d.Scope)
	}
}
