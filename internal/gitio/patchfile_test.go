package gitio

import (
	"os"
	"path/filepath"
	"testing"
)

// writePatch writes content to a temp file and returns its path.
func writePatch(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "change.patch")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	return p
}

func TestFromPatchFile_AddedAndModified(t *testing.T) {
	patch := "" +
		"diff --git a/new.go b/new.go\n" +
		"new file mode 100644\n" +
		"--- /dev/null\n" +
		"+++ b/new.go\n" +
		"@@ -0,0 +1,2 @@\n" +
		"+package new\n" +
		"+// hello\n" +
		"diff --git a/main.go b/main.go\n" +
		"--- a/main.go\n" +
		"+++ b/main.go\n" +
		"@@ -3 +3 @@\n" +
		"-old line\n" +
		"+new line\n"

	d, err := FromPatchFile(writePatch(t, patch))
	if err != nil {
		t.Fatalf("FromPatchFile: %v", err)
	}
	if len(d.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(d.Files), d.Files)
	}

	added := d.Files[0]
	if added.Path != "new.go" || added.Kind != KindAdded {
		t.Errorf("added file mismatched: %+v", added)
	}
	if added.Added != 2 || len(added.AddedLines) != 2 {
		t.Errorf("expected 2 added lines, got %+v", added)
	}
	if added.AddedLines[0].Text != "package new" {
		t.Errorf("expected first added line text, got %q", added.AddedLines[0].Text)
	}

	modified := d.Files[1]
	if modified.Path != "main.go" || modified.Kind != KindModified {
		t.Errorf("modified file mismatched: %+v", modified)
	}
	if modified.Added != 1 || modified.AddedLines[0].Text != "new line" {
		t.Errorf("expected 1 added line 'new line', got %+v", modified)
	}
}

func TestFromPatchFile_Deletion(t *testing.T) {
	patch := "" +
		"diff --git a/gone.go b/gone.go\n" +
		"deleted file mode 100644\n" +
		"--- a/gone.go\n" +
		"+++ /dev/null\n" +
		"@@ -1,2 +0,0 @@\n" +
		"-line one\n" +
		"-line two\n"

	d, err := FromPatchFile(writePatch(t, patch))
	if err != nil {
		t.Fatalf("FromPatchFile: %v", err)
	}
	if len(d.Files) != 1 {
		t.Fatalf("expected 1 file, got %d: %+v", len(d.Files), d.Files)
	}
	f := d.Files[0]
	if f.Path != "gone.go" || f.Kind != KindDeleted {
		t.Errorf("deletion mismatched: %+v", f)
	}
	if f.Added != 0 || len(f.AddedLines) != 0 {
		t.Errorf("deletion should have no added lines, got %+v", f)
	}
}

func TestFromPatchFile_BinaryMarker(t *testing.T) {
	patch := "" +
		"diff --git a/logo.png b/logo.png\n" +
		"index 1111111..2222222 100644\n" +
		"Binary files a/logo.png and b/logo.png differ\n"

	d, err := FromPatchFile(writePatch(t, patch))
	if err != nil {
		t.Fatalf("FromPatchFile: %v", err)
	}
	if len(d.Files) != 1 {
		t.Fatalf("expected 1 file, got %d: %+v", len(d.Files), d.Files)
	}
	f := d.Files[0]
	if !f.Binary {
		t.Errorf("expected binary detection, got %+v", f)
	}
	if f.Path != "logo.png" || f.Kind != KindModified {
		t.Errorf("binary modified file mismatched: %+v", f)
	}
}

func TestFromPatchFile_BinaryAdded(t *testing.T) {
	patch := "" +
		"diff --git a/new.png b/new.png\n" +
		"new file mode 100644\n" +
		"Binary files /dev/null and b/new.png differ\n"

	d, err := FromPatchFile(writePatch(t, patch))
	if err != nil {
		t.Fatalf("FromPatchFile: %v", err)
	}
	if len(d.Files) != 1 {
		t.Fatalf("expected 1 file, got %d: %+v", len(d.Files), d.Files)
	}
	f := d.Files[0]
	if !f.Binary || f.Path != "new.png" || f.Kind != KindAdded {
		t.Errorf("binary added file mismatched: %+v", f)
	}
}

func TestFromPatchFile_MissingFile(t *testing.T) {
	if _, err := FromPatchFile(filepath.Join(t.TempDir(), "does-not-exist.patch")); err == nil {
		t.Fatalf("expected an error for a missing patch file")
	}
}
