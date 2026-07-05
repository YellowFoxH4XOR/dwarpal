package branchpolicy

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

func run(t *testing.T, g *Gate) int {
	t.Helper()
	fs, err := g.Run(context.Background(), &gitio.Diff{})
	if err != nil {
		t.Fatal(err)
	}
	return len(fs)
}

func TestBranchPolicy_AgentOnProtectedBlocked(t *testing.T) {
	if n := run(t, New([]string{"main", "release/*"}, "main", true)); n != 1 {
		t.Fatalf("agent on main should block, got %d findings", n)
	}
	if n := run(t, New([]string{"main", "release/*"}, "release/1.2", true)); n != 1 {
		t.Fatalf("agent on release/* should block, got %d findings", n)
	}
}

func TestBranchPolicy_AgentOnFeatureBranchAllowed(t *testing.T) {
	if n := run(t, New([]string{"main", "release/*"}, "agent/AUTH-42", true)); n != 0 {
		t.Fatalf("agent on feature branch should pass, got %d findings", n)
	}
}

// Human commits are never blocked by branch policy, even on protected branches.
func TestBranchPolicy_HumanNeverBlocked(t *testing.T) {
	if n := run(t, New([]string{"main"}, "main", false)); n != 0 {
		t.Fatalf("human on main must not be blocked by this gate, got %d findings", n)
	}
}
