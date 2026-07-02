package gitio

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The -z numstat/name-status formats are fiddly (renames use an empty path
// field followed by two NUL-separated tokens). These unit tests pin the parser
// against crafted byte streams so a regression is caught without spawning git.

func TestParseNumstat_NormalBinaryAndRename(t *testing.T) {
	// normal (5/2), binary (-/-), rename (1/0 with empty path + old + new)
	in := []byte("5\t2\tsrc/a.go\x00" + "-\t-\tlogo.png\x00" + "1\t0\t\x00old.txt\x00new.txt\x00")
	files := parseNumstat(in)
	if len(files) != 3 {
		t.Fatalf("want 3 files, got %d: %+v", len(files), files)
	}
	if files[0].Path != "src/a.go" || files[0].Added != 5 || files[0].Removed != 2 || files[0].Binary {
		t.Errorf("normal file parsed wrong: %+v", files[0])
	}
	if !files[1].Binary || files[1].Path != "logo.png" || files[1].Added != 0 {
		t.Errorf("binary file parsed wrong: %+v", files[1])
	}
	if files[2].Kind != KindRenamed || files[2].OldPath != "old.txt" || files[2].Path != "new.txt" {
		t.Errorf("rename parsed wrong: %+v", files[2])
	}
}

func TestParseNameStatus_Kinds(t *testing.T) {
	in := []byte("A\x00added.txt\x00" + "M\x00mod.txt\x00" + "R100\x00from.txt\x00to.txt\x00" + "D\x00gone.txt\x00")
	kinds := parseNameStatus(in)
	cases := map[string]ChangeKind{
		"added.txt": KindAdded,
		"mod.txt":   KindModified,
		"to.txt":    KindRenamed,
		"gone.txt":  KindDeleted,
	}
	for path, want := range cases {
		if kinds[path] != want {
			t.Errorf("%s: kind = %q, want %q", path, kinds[path], want)
		}
	}
}

// Integration: exercise the whole extractor against a real staged tree,
// including a binary file, a rename, and a path with a space.
func TestStaged_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t.co",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t.co",
			"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	git("init")
	// Commit a file we will later rename.
	write("orig.txt", "hello\n")
	git("add", "orig.txt")
	git("commit", "-m", "base")

	// Stage: a new text file with a space in its name, a binary file, a rename.
	write("with space.txt", "a\nb\nc\n")
	write("logo.bin", "\x00\x01\x02\x03\x00\xff")
	git("mv", "orig.txt", "renamed.txt")
	git("add", "-A")

	diff, err := NewExtractor(dir).Staged()
	if err != nil {
		t.Fatal(err)
	}

	byPath := map[string]FileChange{}
	for _, f := range diff.Files {
		byPath[f.Path] = f
	}
	if f, ok := byPath["with space.txt"]; !ok || f.Kind != KindAdded || f.Added != 3 {
		t.Errorf("space-path file wrong: %+v (present=%v)", f, ok)
	}
	if f, ok := byPath["logo.bin"]; !ok || !f.Binary {
		t.Errorf("binary file wrong: %+v (present=%v)", f, ok)
	}
	if f, ok := byPath["renamed.txt"]; !ok || f.Kind != KindRenamed {
		t.Errorf("rename wrong: %+v (present=%v)", f, ok)
	}
	if diff.NewFiles() < 1 {
		t.Errorf("expected at least one new file, got %d", diff.NewFiles())
	}
}
