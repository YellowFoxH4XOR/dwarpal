package analyze

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo builds a hermetic temp repo (global/system config pinned to
// /dev/null, deterministic identity) so the git commands inside analyze behave
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

// commitN writes a file of n lines and commits it, producing a commit whose
// shortstat insertion count is n — the unit suggestBudget fits against.
func commitN(t *testing.T, dir string, git func(...string), name string, n int) {
	t.Helper()
	body := make([]byte, 0, n*2)
	for i := 0; i < n; i++ {
		body = append(body, 'x', '\n')
	}
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", name)
	git("commit", "-m", "add "+name)
}

// The budget must track the repo's own commit sizes, not a fixed default: a
// repo that commits in small chunks should get a small budget. This is the
// core promise of analyze — if it ever returns the 500 default here, the
// history-fitting logic has silently broken.
func TestSuggestBudget_FitsRepoHistory(t *testing.T) {
	dir, git := gitRepo(t)
	for i := 0; i < 20; i++ {
		commitN(t, dir, git, "f"+string(rune('a'+i))+".txt", 40)
	}

	b := suggestBudget(dir)
	if b.SampleCount < 15 {
		t.Fatalf("sampled %d commits, want >=15 — shortstat parsing likely broke", b.SampleCount)
	}
	// All commits are ~40 lines, so p75*2 lands near 100 and must be far below
	// the 500-line default. A default here means history-fitting was skipped.
	if b.MaxLines >= 500 {
		t.Errorf("MaxLines = %d; want a repo-fitted value well below the 500 default", b.MaxLines)
	}
	if b.MedianLines == 0 || b.P75Lines == 0 {
		t.Errorf("distribution not populated: median=%d p75=%d", b.MedianLines, b.P75Lines)
	}
}

// A single giant commit (bootstrap import, generated code) must NOT drag the
// budget up to its size — that is the outlier-robustness the p75 basis exists
// to provide. If this fails, one code-dump commit would set everyday policy.
func TestSuggestBudget_IgnoresOutlierCommits(t *testing.T) {
	dir, git := gitRepo(t)
	for i := 0; i < 15; i++ {
		commitN(t, dir, git, "f"+string(rune('a'+i))+".txt", 30)
	}
	commitN(t, dir, git, "generated.txt", 5000) // the outlier

	b := suggestBudget(dir)
	if b.MaxSeen < 5000 {
		t.Fatalf("MaxSeen = %d; the 5000-line commit should be visible in the distribution", b.MaxSeen)
	}
	// The budget must stay near the typical 30-line commits, not the outlier.
	if b.MaxLines > 1000 {
		t.Errorf("MaxLines = %d; a single 5000-line commit skewed the budget — p75 basis failed", b.MaxLines)
	}
}

// Too little history is the honest-fallback case: rather than fit a budget to
// 3 commits, analyze must fall back to the documented default and SAY so, so
// the agent knows the number is a guess, not a measurement.
func TestSuggestBudget_FallsBackWhenHistoryThin(t *testing.T) {
	dir, git := gitRepo(t)
	commitN(t, dir, git, "a.txt", 10)
	commitN(t, dir, git, "b.txt", 10)

	b := suggestBudget(dir)
	if b.MaxLines != 500 {
		t.Errorf("MaxLines = %d; want the 500 default with <10 commits", b.MaxLines)
	}
}

// parseShortstat is the fragile seam (git's wording varies: singular
// "insertion", deletions-only lines). Lock the contract with representative
// lines so a git output change is caught here, not as a silently wrong budget.
func TestParseShortstat(t *testing.T) {
	cases := map[string]int{
		" 3 files changed, 42 insertions(+), 7 deletions(-)": 49,
		" 1 file changed, 1 insertion(+)":                    1,
		" 1 file changed, 5 deletions(-)":                    5,
		" 2 files changed, 10 insertions(+)":                 10,
	}
	for line, want := range cases {
		if got := parseShortstat(line); got != want {
			t.Errorf("parseShortstat(%q) = %d, want %d", line, got, want)
		}
	}
}

// analyze must not touch the user's config or source. The ONE permitted side
// effect is warming the gitignored convention cache under .dwarpal/cache/ (the
// same accelerator the gates already write) — anything outside that path would
// break the "reads your repo, never mutates it" promise the design rests on.
func TestRun_WritesOnlyTheGitignoredCache(t *testing.T) {
	dir, git := gitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package p\nfunc Foo() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-m", "src")

	before := snapshot(t, dir)
	if _, err := Run(dir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	after := snapshot(t, dir)
	cacheDir := filepath.Join(dir, ".dwarpal")
	for p := range after {
		if before[p] {
			continue
		}
		if strings.HasPrefix(p, cacheDir) {
			continue // the allowed, gitignored accelerator
		}
		t.Errorf("Run created %s outside the cache; analyze must not mutate config or source", p)
	}
}

func snapshot(t *testing.T, dir string) map[string]bool {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	files := map[string]bool{}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		// .git is git's own churn (auto-maintenance creates and removes lock
		// files mid-walk); the test cares only about config/source/cache, so
		// skip it entirely — and tolerate anything that vanishes under us.
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() && p == gitDir {
			return filepath.SkipDir
		}
		files[p] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// analyze must produce a per-language convention fingerprint for non-Go
// languages too — the whole point of the language-parity work. A Python repo
// should report Python's function count and its learned naming style, not just
// import forms.
func TestRun_PerLanguageConventionsForNonGo(t *testing.T) {
	dir, git := gitRepo(t)
	for i := 0; i < 8; i++ {
		name := "h" + string(rune('a'+i)) + ".py"
		write(t, dir, name, "def handle_request_"+string(rune('a'+i))+"():\n    return 1\n")
	}
	git("add", "-A")
	git("commit", "-m", "python")

	rep, err := Run(dir)
	if err != nil {
		t.Fatal(err)
	}
	py, ok := rep.Conventions["python"]
	if !ok {
		t.Fatalf("expected python conventions; got languages=%v", rep.Languages)
	}
	if py.Funcs < 5 {
		t.Errorf("expected python function count, got %d", py.Funcs)
	}
	if py.DominantNaming != "snake_case" {
		t.Errorf("a snake_case Python repo should report snake_case naming, got %q", py.DominantNaming)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
