package provenance

import "testing"

func TestDetect_EnvVarWins(t *testing.T) {
	t.Setenv(EnvVar, "Claude Code")
	d := New([]string{"agent/"}, []string{"Claude"})
	// Even on a non-agent branch with no trailer, the env var identifies it.
	got := d.Detect("main", "")
	if !got.IsAgent || got.Source != SourceEnv || got.Agent != "Claude Code" {
		t.Fatalf("env detection wrong: %+v", got)
	}
}

func TestDetect_Trailer(t *testing.T) {
	t.Setenv(EnvVar, "")
	d := New(nil, []string{"Claude", "Cursor"})
	msg := "feat: thing\n\nCo-Authored-By: Cursor <agent@cursor.sh>\n"
	got := d.Detect("main", msg)
	if !got.IsAgent || got.Source != SourceTrailer || got.Agent != "Cursor" {
		t.Fatalf("trailer detection wrong: %+v", got)
	}
}

func TestDetect_BranchPrefix(t *testing.T) {
	t.Setenv(EnvVar, "")
	d := New([]string{"agent/", "ai/"}, []string{"Claude"})
	got := d.Detect("agent/AUTH-42", "")
	if !got.IsAgent || got.Source != SourceBranch {
		t.Fatalf("branch detection wrong: %+v", got)
	}
}

// A human commit on a normal branch with no agent signals must NOT be flagged —
// this is what keeps human commits untouched (risk R2).
func TestDetect_HumanNotFlagged(t *testing.T) {
	t.Setenv(EnvVar, "")
	d := New([]string{"agent/"}, []string{"Claude"})
	got := d.Detect("main", "fix: typo\n\nCo-Authored-By: Alice <alice@corp.com>\n")
	if got.IsAgent || got.Source != SourceNone {
		t.Fatalf("human commit misflagged: %+v", got)
	}
}

// Detection order: env var must take precedence over branch/trailer.
func TestDetect_OrderEnvBeatsBranch(t *testing.T) {
	t.Setenv(EnvVar, "Devin")
	d := New([]string{"agent/"}, []string{"Claude"})
	got := d.Detect("agent/x", "Co-Authored-By: Claude <c@a.com>")
	if got.Source != SourceEnv {
		t.Fatalf("expected env to win, got %+v", got)
	}
}
