package intent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

// mockProvider is a deterministic, no-network Provider for tests.
type mockProvider struct {
	verdict Verdict
	err     error
	delay   time.Duration // simulates a slow/hanging provider for timeout tests
}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) Verify(ctx context.Context, prompt string) (Verdict, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return Verdict{}, ctx.Err()
		}
	}
	if m.err != nil {
		return Verdict{}, m.err
	}
	return m.verdict, nil
}

func diff(paths ...string) *gitio.Diff {
	d := &gitio.Diff{}
	for _, p := range paths {
		d.Files = append(d.Files, gitio.FileChange{
			Path: p, Kind: gitio.KindModified, Added: 1,
			AddedLines: []gitio.Line{{Number: 1, Text: "some change"}},
		})
	}
	return d
}

// A verdict flagging surprises (or scope/intent problems) produces warn
// findings — the gate is advisory, so it must never error even when it
// disagrees with the diff.
func TestIntent_VerdictWithSurprisesProducesWarnFindings(t *testing.T) {
	p := &mockProvider{verdict: Verdict{
		AccomplishesIntent: true,
		OnlyStatedIntent:   false,
		Surprises:          []string{"unrelated refactor of internal/config"},
	}}
	g := New(p, "fix login bug", time.Second)

	fs, err := g.Run(context.Background(), diff("src/auth/login.go"), engine.NoIndex{})
	if err != nil {
		t.Fatalf("advisory gate must never error, got %v", err)
	}
	if len(fs) == 0 {
		t.Fatal("expected findings for a verdict with surprises/scope issues")
	}
	for _, f := range fs {
		if f.Severity != "warn" {
			t.Errorf("expected warn severity, got %q", f.Severity)
		}
		if f.RetryHint == "" {
			t.Errorf("expected a retry hint on finding %+v", f)
		}
	}

	foundScope := false
	foundSurprise := false
	for _, f := range fs {
		if f.RuleID == "intent-scope-exceeded" {
			foundScope = true
		}
		if f.RuleID == "intent-surprise" {
			foundSurprise = true
		}
	}
	if !foundScope || !foundSurprise {
		t.Errorf("expected both scope-exceeded and surprise findings, got %+v", fs)
	}
}

// A clean verdict (accomplishes intent, only that intent, no surprises)
// produces no findings — the gate should stay silent when the diff matches
// the declared intent.
func TestIntent_CleanVerdictProducesNoFindings(t *testing.T) {
	p := &mockProvider{verdict: Verdict{
		AccomplishesIntent: true,
		OnlyStatedIntent:   true,
	}}
	g := New(p, "fix login bug", time.Second)

	fs, err := g.Run(context.Background(), diff("src/auth/login.go"), engine.NoIndex{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("clean verdict should produce no findings, got %+v", fs)
	}
}

// The single documented exception to fail-closed: a provider error is an
// infrastructure failure, not evidence the diff is unsafe, so it must not
// block the commit (no error, no findings).
func TestIntent_ProviderErrorFailsOpen(t *testing.T) {
	p := &mockProvider{err: errors.New("provider unreachable")}
	g := New(p, "fix login bug", time.Second)

	fs, err := g.Run(context.Background(), diff("src/auth/login.go"), engine.NoIndex{})
	if err != nil {
		t.Fatalf("infra failure must fail open (nil error), got %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("infra failure must fail open (no findings), got %+v", fs)
	}
}

// A provider that exceeds the configured timeout must also fail open —
// a slow/hanging third-party API is an infra failure like any other, and
// must never block a commit.
func TestIntent_TimeoutFailsOpen(t *testing.T) {
	p := &mockProvider{delay: 100 * time.Millisecond, verdict: Verdict{AccomplishesIntent: true, OnlyStatedIntent: true}}
	g := New(p, "fix login bug", 10*time.Millisecond)

	start := time.Now()
	fs, err := g.Run(context.Background(), diff("src/auth/login.go"), engine.NoIndex{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("timeout must fail open (nil error), got %v", err)
	}
	if len(fs) != 0 {
		t.Fatalf("timeout must fail open (no findings), got %+v", fs)
	}
	if elapsed >= p.delay {
		t.Errorf("expected Run to return around the gate timeout (%v), not wait for the full provider delay (%v); took %v", g.timeout, p.delay, elapsed)
	}
}

// A nil provider (gate disabled/unconfigured) must be a safe no-op, not a
// panic — mirrors how scope.Gate no-ops with no manifest.
func TestIntent_NilProviderNoop(t *testing.T) {
	g := New(nil, "fix login bug", time.Second)
	fs, err := g.Run(context.Background(), diff("src/auth/login.go"), engine.NoIndex{})
	if err != nil || len(fs) != 0 {
		t.Fatalf("nil provider should be a no-op, got fs=%+v err=%v", fs, err)
	}
}

func TestIntent_ID(t *testing.T) {
	g := New(&mockProvider{}, "", time.Second)
	if g.ID() != "intent" {
		t.Fatalf("expected gate id 'intent', got %q", g.ID())
	}
}
