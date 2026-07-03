package repoindex

import (
	"regexp"
	"strings"
)

// Error-idiom fingerprint dimension (checklist #37, PRD Gate 6) — Go only,
// since the idioms below are Go-shaped. Counted per line during Go file
// indexing; the drift gate compares added error-handling lines against the
// dominant idiom.

// Error idiom labels.
const (
	IdiomWrap  = "wrap"  // return fmt.Errorf("...: %w", err)
	IdiomBare  = "bare"  // return err (unwrapped)
	IdiomPanic = "panic" // panic(err) / panic("...")
)

// idiomLang keys the Conventions.Imports-style map; error idioms are their own
// namespace so they never collide with import forms.
const idiomLang = "go-error-idiom"

var (
	wrapPattern  = regexp.MustCompile(`fmt\.Errorf\([^)]*%w`)
	barePattern  = regexp.MustCompile(`^\s*return\s+(\w+,\s*)*err\s*$`)
	panicPattern = regexp.MustCompile(`\bpanic\(`)
)

// ClassifyErrorIdiomLine maps one Go source line to an error idiom, or ""
// when the line is not error handling. Shared by the fingerprint counter and
// the drift gate so both speak the same labels.
func ClassifyErrorIdiomLine(line string) string {
	switch {
	case wrapPattern.MatchString(line):
		return IdiomWrap
	case panicPattern.MatchString(line):
		return IdiomPanic
	case barePattern.MatchString(line):
		return IdiomBare
	default:
		return ""
	}
}

// countErrorIdioms folds a Go file's error-handling lines into the fingerprint.
func (idx *Index) countErrorIdioms(src []byte) {
	for _, line := range strings.Split(string(src), "\n") {
		if idiom := ClassifyErrorIdiomLine(line); idiom != "" {
			idx.addImport(idiomLang, idiom) // reuses the per-lang counter map
		}
	}
}

// DominantErrorIdiom returns the repo's dominant Go error idiom and its share,
// or ("", 0) below the sample-size threshold — same contract as
// DominantImportForm.
func (c Conventions) DominantErrorIdiom() (string, float64) {
	return c.DominantImportForm(idiomLang)
}
