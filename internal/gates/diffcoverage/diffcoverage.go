// Package diffcoverage implements Gate 5 — diff coverage.
//
// It reads a coverage artifact produced by the team's own test command (lcov,
// Cobertura XML, or Go's cover.out) and checks that the lines *added* by this
// diff meet a minimum coverage percentage. Dwarpal never runs tests itself
// (PRD §5.2 Gate 5); it only consumes whatever artifact the caller points it
// at.
//
// Missing artifact is warn-only (the team hasn't wired coverage yet — that's
// not this diff's fault). A present-but-unparseable artifact is a hard error
// (fail-closed), matching every other deterministic gate in this repo.
package diffcoverage

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/engine"
	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
	"github.com/YellowFoxH4XOR/dwarpal/internal/gitio"
)

const gateID = "diff_coverage"

// Gate enforces a minimum coverage percentage on changed (added) lines.
type Gate struct {
	artifactPath string
	minPercent   float64
	repoRoot     string
}

// New builds the diff-coverage gate. artifactPath is resolved relative to
// repoRoot when it is not already absolute. minPercent is the required
// coverage percentage (0-100) on added lines.
func New(artifactPath string, minPercent float64, repoRoot string) *Gate {
	return &Gate{artifactPath: artifactPath, minPercent: minPercent, repoRoot: repoRoot}
}

// ID identifies the gate.
func (g *Gate) ID() string { return gateID }

// fileCoverage maps line number -> hit count for one source file, as reported
// by the coverage artifact.
type fileCoverage map[int]int

// Run computes coverage over the diff's added lines and compares it to the
// configured threshold.
func (g *Gate) Run(_ context.Context, d *gitio.Diff, _ engine.RepoIndex) ([]finding.Finding, error) {
	path := g.artifactPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(g.repoRoot, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []finding.Finding{{
				Gate:     gateID,
				RuleID:   "artifact-missing",
				Severity: finding.SeverityInfo,
				Message:  fmt.Sprintf("no coverage artifact found at %s; diff-coverage gate skipped", g.artifactPath),
			}}, nil
		}
		return nil, fmt.Errorf("diffcoverage: reading artifact %s: %w", path, err)
	}

	cov, err := parseArtifact(string(data))
	if err != nil {
		return nil, fmt.Errorf("diffcoverage: parsing artifact %s: %w", path, err)
	}

	var coverable, covered int
	for _, f := range d.Files {
		fc, ok := lookupFile(cov, f.Path)
		if !ok {
			continue
		}
		for _, line := range f.AddedLines {
			hits, tracked := fc[line.Number]
			if !tracked {
				continue
			}
			coverable++
			if hits > 0 {
				covered++
			}
		}
	}

	if coverable == 0 {
		return nil, nil
	}

	pct := 100 * float64(covered) / float64(coverable)
	if pct < g.minPercent {
		return []finding.Finding{{
			Gate:     gateID,
			RuleID:   "below-threshold",
			Severity: finding.SeverityError,
			Message: fmt.Sprintf("changed-line coverage is %.1f%%, below the required %.1f%% (%d/%d added lines covered)",
				pct, g.minPercent, covered, coverable),
			Suggestion: "add or extend tests to cover the added lines, then regenerate the coverage artifact",
			RetryHint: fmt.Sprintf("Diff coverage is %.1f%% but %.1f%% is required. Add tests covering the uncovered added lines and rerun.",
				pct, g.minPercent),
		}}, nil
	}

	return nil, nil
}

// lookupFile finds the coverage data for a diff path, tolerating the
// artifact recording paths absolute, repo-root-relative, or with a "./"
// prefix.
func lookupFile(cov map[string]fileCoverage, diffPath string) (fileCoverage, bool) {
	diffPath = normalizePath(diffPath)
	if fc, ok := cov[diffPath]; ok {
		return fc, true
	}
	for covPath, fc := range cov {
		if strings.HasSuffix(covPath, "/"+diffPath) {
			return fc, true
		}
	}
	return nil, false
}

func normalizePath(p string) string {
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	return p
}

// parseArtifact auto-detects the coverage format by content and parses it
// into per-file line coverage.
func parseArtifact(data string) (map[string]fileCoverage, error) {
	trimmed := strings.TrimSpace(data)
	switch {
	case trimmed == "":
		return nil, fmt.Errorf("empty coverage artifact")
	case strings.HasPrefix(trimmed, "mode:"):
		return parseGoCover(data)
	case strings.Contains(data, "\nSF:") || strings.HasPrefix(trimmed, "SF:") || strings.HasPrefix(trimmed, "TN:"):
		return parseLCOV(data)
	case strings.HasPrefix(trimmed, "<?xml") || strings.HasPrefix(trimmed, "<coverage"):
		return parseCobertura(data)
	default:
		return nil, fmt.Errorf("unrecognized coverage artifact format")
	}
}

// parseLCOV parses lcov.info: "SF:<path>" starts a section, "DA:<line>,<hits>"
// records a coverable line, "end_of_record" closes the section.
func parseLCOV(data string) (map[string]fileCoverage, error) {
	result := map[string]fileCoverage{}
	var curPath string
	var cur fileCoverage

	sc := bufio.NewScanner(strings.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "SF:"):
			curPath = normalizePath(strings.TrimPrefix(line, "SF:"))
			cur = fileCoverage{}
		case strings.HasPrefix(line, "DA:"):
			if cur == nil {
				return nil, fmt.Errorf("lcov: DA record before SF record")
			}
			fields := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
			if len(fields) < 2 {
				return nil, fmt.Errorf("lcov: malformed DA record %q", line)
			}
			ln, err := strconv.Atoi(fields[0])
			if err != nil {
				return nil, fmt.Errorf("lcov: malformed DA line number %q: %w", line, err)
			}
			hits, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("lcov: malformed DA hit count %q: %w", line, err)
			}
			cur[ln] = hits
		case line == "end_of_record":
			if curPath != "" {
				result[curPath] = cur
			}
			curPath = ""
			cur = nil
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("lcov: no SF/end_of_record sections found")
	}
	return result, nil
}

// parseGoCover parses Go's `go test -coverprofile` output:
//
//	mode: set
//	file:startLine.startCol,endLine.endCol numStmt count
//
// Each block covers a line range; we mark every line in the range as
// coverable, with the block's hit count.
func parseGoCover(data string) (map[string]fileCoverage, error) {
	result := map[string]fileCoverage{}

	sc := bufio.NewScanner(strings.NewReader(data))
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			if !strings.HasPrefix(line, "mode:") {
				return nil, fmt.Errorf("go cover: missing mode header")
			}
			continue
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			return nil, fmt.Errorf("go cover: malformed block %q", line)
		}
		path := normalizePath(line[:colon])
		rest := line[colon+1:]
		fields := strings.Fields(rest)
		if len(fields) != 3 {
			return nil, fmt.Errorf("go cover: malformed block %q", line)
		}
		posFields := strings.Split(fields[0], ",")
		if len(posFields) != 2 {
			return nil, fmt.Errorf("go cover: malformed position %q", fields[0])
		}
		startLine, err := goCoverLineOf(posFields[0])
		if err != nil {
			return nil, err
		}
		endLine, err := goCoverLineOf(posFields[1])
		if err != nil {
			return nil, err
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("go cover: malformed count %q: %w", fields[2], err)
		}

		fc := result[path]
		if fc == nil {
			fc = fileCoverage{}
			result[path] = fc
		}
		for ln := startLine; ln <= endLine; ln++ {
			// A block can appear more than once (rare); keep the max hit count.
			if existing, ok := fc[ln]; !ok || count > existing {
				fc[ln] = count
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("go cover: no coverage blocks found")
	}
	return result, nil
}

// goCoverLineOf parses the line number out of a go cover "line.col" token.
func goCoverLineOf(posColPair string) (int, error) {
	dot := strings.Index(posColPair, ".")
	if dot < 0 {
		return 0, fmt.Errorf("go cover: malformed line.col %q", posColPair)
	}
	return strconv.Atoi(posColPair[:dot])
}

// Cobertura XML shape (only the fields we need).
type coberturaReport struct {
	Packages []struct {
		Classes []struct {
			Filename string `xml:"filename,attr"`
			Lines    struct {
				Line []struct {
					Number int `xml:"number,attr"`
					Hits   int `xml:"hits,attr"`
				} `xml:"line"`
			} `xml:"lines"`
		} `xml:"classes>class"`
	} `xml:"packages>package"`
}

func parseCobertura(data string) (map[string]fileCoverage, error) {
	var report coberturaReport
	if err := xml.Unmarshal([]byte(data), &report); err != nil {
		return nil, fmt.Errorf("cobertura: %w", err)
	}

	result := map[string]fileCoverage{}
	for _, pkg := range report.Packages {
		for _, class := range pkg.Classes {
			if class.Filename == "" {
				continue
			}
			path := normalizePath(class.Filename)
			fc := result[path]
			if fc == nil {
				fc = fileCoverage{}
				result[path] = fc
			}
			for _, l := range class.Lines.Line {
				fc[l.Number] = l.Hits
			}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("cobertura: no <class filename=...> entries found")
	}
	return result, nil
}
