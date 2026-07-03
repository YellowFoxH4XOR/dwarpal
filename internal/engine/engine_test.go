package engine

import (
	"context"
	"testing"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// stubGate emits a fixed finding set, recording whether it ran.
type stubGate struct {
	id  string
	out []finding.Finding
	ran *bool
}

func (s stubGate) ID() string { return s.id }
func (s stubGate) Run(context.Context, *gitio.Diff, RepoIndex) ([]finding.Finding, error) {
	*s.ran = true
	return s.out, nil
}

func block(gate string) []finding.Finding {
	return []finding.Finding{{Gate: gate, RuleID: "r", Severity: finding.SeverityError, Message: "m"}}
}

// Default: report-everything — later gates still run after a block.
func TestRun_ReportEverythingDefault(t *testing.T) {
	a, b := false, false
	res := Run(context.Background(), []Gate{
		stubGate{"g1", block("g1"), &a},
		stubGate{"g2", block("g2"), &b},
	}, &gitio.Diff{}, NoIndex{})
	if !a || !b {
		t.Fatalf("both gates should run by default (a=%v b=%v)", a, b)
	}
	if len(res.Findings) != 2 {
		t.Fatalf("want both findings reported, got %d", len(res.Findings))
	}
}

// stop_on_first_block: the run ends at the first blocking gate.
func TestRunWith_StopOnFirstBlock(t *testing.T) {
	a, b := false, false
	res := RunWith(context.Background(), []Gate{
		stubGate{"g1", block("g1"), &a},
		stubGate{"g2", block("g2"), &b},
	}, &gitio.Diff{}, NoIndex{}, Options{StopOnFirstBlock: true})
	if !a {
		t.Fatal("first gate should run")
	}
	if b {
		t.Fatal("second gate must NOT run after a block with StopOnFirstBlock")
	}
	if len(res.Findings) != 1 {
		t.Fatalf("want one finding, got %d", len(res.Findings))
	}
}

// Parallel execution must preserve gate-order determinism in the output.
func TestRun_ParallelPreservesOrder(t *testing.T) {
	var gates []Gate
	flags := make([]bool, 8)
	for i := 0; i < 8; i++ {
		gates = append(gates, stubGate{
			id:  string(rune('a' + i)),
			out: block(string(rune('a' + i))),
			ran: &flags[i],
		})
	}
	for run := 0; run < 5; run++ {
		res := Run(context.Background(), gates, &gitio.Diff{}, NoIndex{})
		if len(res.Findings) != 8 {
			t.Fatalf("want 8 findings, got %d", len(res.Findings))
		}
		for i, f := range res.Findings {
			if f.Gate != string(rune('a'+i)) {
				t.Fatalf("run %d: order broken at %d: %s", run, i, f.Gate)
			}
		}
	}
}
