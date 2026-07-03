package taskmanifest

import "testing"

func TestTicketFromBranch(t *testing.T) {
	cases := []struct {
		name   string
		branch string
		want   string
	}{
		{"simple prefix", "agent/AUTH-42", "AUTH-42"},
		{"trailing description", "agent/AUTH-42-password-reset", "AUTH-42"},
		{"underscore separator", "feature/JIRA-99_fix", "JIRA-99"},
		{"short alpha prefix", "ai/PROJ-7", "PROJ-7"},
		{"lower-cased input upper-cased out", "agent/auth-42-fix", "AUTH-42"},
		{"no ticket, plain branch", "main", ""},
		{"ambiguous: digits glued to trailing letter", "agent/no-ticket-here-123x", ""},
		{"digit run too long (>6) never reaches a boundary", "agent/AUTH-1234567", ""},
		{"alpha prefix too long (>10 chars)", "agent/VERYLONGPREFIX-42", ""},
		{"alpha prefix too short (1 char)", "agent/A-42", ""},
		{"no slash at all, ticket is whole branch", "PROJ-123", "PROJ-123"},
		{"empty branch", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TicketFromBranch(c.branch); got != c.want {
				t.Errorf("TicketFromBranch(%q) = %q, want %q", c.branch, got, c.want)
			}
		})
	}
}
