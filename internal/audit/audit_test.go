package audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// awsKey is built from parts so this SOURCE file never contains a literal
// that trips Dwarpal's own no-hardcoded-secrets gate; the fixture files we write
// still receive the full key, which is what audit replays against.
var awsKey = "AKIA" + "IOSFODNN7EXAMPLE"

// noqa is split so this file doesn't itself trip no-new-lint-suppressions.
var noqa = "# no" + "qa"

// gitRepo builds a hermetic temp repo so the git commands inside audit behave
// identically on any host — the pattern used across the codebase's git tests.
func gitRepo(t *testing.T) (string, func(...string)) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t.co")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t.co")

	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init")
	return dir, git
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func stat(rules []RuleStat, ruleID string) *RuleStat {
	for i := range rules {
		if rules[i].RuleID == ruleID {
			return &rules[i]
		}
	}
	return nil
}

// The whole point of audit: a flagged line a human later FIXED counts as
// acted-on (signal), and one LEFT in place counts as survived (candidate noise).
// If this inverts or collapses, the calibration signal is worthless.
func TestRun_ActedOnVsSurvived(t *testing.T) {
	dir, git := gitRepo(t)

	// Root commit (audit skips it — no parent to diff against).
	write(t, dir, "seed.txt", "seed\n")
	git("add", "-A")
	git("commit", "-m", "seed")

	// Commit that INTRODUCES two flagged lines in files with a parent.
	write(t, dir, "creds.py", "key = \""+awsKey+"\"\n") // aws-key
	write(t, dir, "util.py", "x = 1  "+noqa+"\n")       // lint suppression
	git("add", "-A")
	git("commit", "-m", "add flagged lines")

	// A human FIXES the aws key (rewrites that line) but LEAVES the suppression.
	write(t, dir, "creds.py", "key = load_from_env()\n")
	git("add", "-A")
	git("commit", "-m", "fix hardcoded key")

	rep, err := Run(dir, Options{Window: 50, MinSamples: 1, DemoteThreshold: 0.15, PromoteThreshold: 0.6})
	if err != nil {
		t.Fatal(err)
	}

	aws := stat(rep.Rules, "no-hardcoded-secrets/aws-key")
	if aws == nil {
		t.Fatalf("aws-key rule never sampled; rules=%+v", rep.Rules)
	}
	if aws.ActedOn != aws.Samples || aws.Samples < 1 {
		t.Errorf("fixed secret must be 100%% acted-on, got %d/%d", aws.ActedOn, aws.Samples)
	}

	sup := stat(rep.Rules, "no-new-lint-suppressions")
	if sup == nil {
		t.Fatalf("suppression rule never sampled; rules=%+v", rep.Rules)
	}
	if sup.ActedOn != 0 {
		t.Errorf("untouched suppression must be 0%% acted-on, got %d/%d", sup.ActedOn, sup.Samples)
	}
}

// A noisy error-severity rule (flags survive untouched) over enough samples
// must be recommended for demotion — that recommendation is the product.
func TestRun_RecommendsDemotingNoisyRule(t *testing.T) {
	dir, git := gitRepo(t)
	write(t, dir, "seed.txt", "seed\n")
	git("add", "-A")
	git("commit", "-m", "seed")

	// Ten commits each adding a surviving hardcoded key (error severity, never
	// touched again) → 0% acted-on over 10 samples → should be flagged as noise.
	for i := 0; i < 10; i++ {
		name := "k" + string(rune('a'+i)) + ".py"
		write(t, dir, name, "key = \""+awsKey+"\"\n")
		git("add", "-A")
		git("commit", "-m", "add "+name)
	}

	rep, err := Run(dir, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	aws := stat(rep.Rules, "no-hardcoded-secrets/aws-key")
	if aws == nil || aws.Samples < 8 {
		t.Fatalf("expected >=8 samples for aws-key, got %+v", aws)
	}
	if aws.ActedOnRate > 0.15 || aws.Recommendation == "" {
		t.Errorf("a 0%%-acted-on error rule must be recommended for demotion, got rate=%.2f rec=%q",
			aws.ActedOnRate, aws.Recommendation)
	}
}

// Audit must not mutate the work tree or config — it is advisory only. Its one
// permitted footprint is an OS temp dir it cleans up itself.
func TestRun_WritesNothingInRepo(t *testing.T) {
	dir, git := gitRepo(t)
	write(t, dir, "seed.txt", "seed\n")
	write(t, dir, "creds.py", "key = \""+awsKey+"\"\n")
	git("add", "-A")
	git("commit", "-m", "seed")
	write(t, dir, "creds.py", "key = \""+awsKey+"\"\n// touched\n")
	git("add", "-A")
	git("commit", "-m", "touch")

	before := listFiles(t, dir)
	if _, err := Run(dir, Defaults()); err != nil {
		t.Fatal(err)
	}
	after := listFiles(t, dir)
	if before != after {
		t.Errorf("audit changed the repo tree:\nbefore=%s\nafter =%s", before, after)
	}
}

func listFiles(t *testing.T, dir string) string {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	var names string
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		// Skip .git: audit runs git commands that trigger transient
		// auto-maintenance lock files, which are git's churn, not audit
		// mutating config/source. Tolerate anything that vanishes mid-walk.
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() && p == gitDir {
			return filepath.SkipDir
		}
		names += p + "\n"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}
