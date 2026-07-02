package gitio

import "testing"

func TestParseUnifiedAdded_LineNumbers(t *testing.T) {
	// Two files. main.go adds two lines starting at new-file line 10;
	// util.go adds one line at line 3 and has a removed line (which must NOT
	// advance the new-file counter).
	diff := "" +
		"diff --git a/main.go b/main.go\n" +
		"--- a/main.go\n" +
		"+++ b/main.go\n" +
		"@@ -9,0 +10,2 @@\n" +
		"+first added\n" +
		"+second added\n" +
		"diff --git a/util.go b/util.go\n" +
		"--- a/util.go\n" +
		"+++ b/util.go\n" +
		"@@ -3 +3 @@\n" +
		"-old line\n" +
		"+new line\n"

	got := parseUnifiedAdded(diff)

	main := got["main.go"]
	if len(main) != 2 || main[0].Number != 10 || main[1].Number != 11 {
		t.Fatalf("main.go added lines wrong: %+v", main)
	}
	if main[0].Text != "first added" {
		t.Errorf("text not stripped of +: %q", main[0].Text)
	}
	util := got["util.go"]
	if len(util) != 1 || util[0].Number != 3 || util[0].Text != "new line" {
		t.Fatalf("util.go added line wrong: %+v", util)
	}
}

func TestParseUnifiedAdded_DeletedFileIgnored(t *testing.T) {
	diff := "" +
		"diff --git a/gone.go b/gone.go\n" +
		"--- a/gone.go\n" +
		"+++ /dev/null\n" +
		"@@ -1,2 +0,0 @@\n" +
		"-line one\n" +
		"-line two\n"
	got := parseUnifiedAdded(diff)
	if len(got) != 0 {
		t.Fatalf("deleted file should contribute no added lines, got %+v", got)
	}
}
