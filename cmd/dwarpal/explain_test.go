package main

import "testing"

func TestRunExplain_KnownID(t *testing.T) {
	if err := runExplain("diff_budget/max-lines", false); err != nil {
		t.Fatalf("runExplain(known id) returned error: %v", err)
	}
}

func TestRunExplain_UnknownID(t *testing.T) {
	if err := runExplain("nonexistent/rule-id", false); err == nil {
		t.Fatal("runExplain(unknown id) returned nil error, want error")
	}
}
