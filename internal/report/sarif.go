package report

import (
	"encoding/json"
	"io"

	"github.com/YellowFoxH4XOR/dwarpal/internal/finding"
)

// SARIF (Static Analysis Results Interchange Format) 2.1.0 output. Emitting
// SARIF is high-leverage/low-cost (PRD §6 #5): GitHub renders it as inline PR
// annotations for free in CI mode, with no bespoke integration.
//
// We emit the minimal valid subset: one run, one tool driver (dwarpal), the
// distinct rules seen, and one result per finding with a file:line location.

const sarifSchema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           *sarifRegion          `json:"region,omitempty"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

// sarifLevel maps a Dwarpal severity to a SARIF level.
func sarifLevel(s finding.Severity) string {
	switch s {
	case finding.SeverityError:
		return "error"
	case finding.SeverityWarn:
		return "warning"
	default:
		return "note"
	}
}

// SARIF writes the findings as a SARIF 2.1.0 log to w.
func SARIF(w io.Writer, in Input) error {
	seenRules := map[string]bool{}
	// Non-nil so a zero-finding run marshals "rules": [] — GitHub's SARIF
	// validator rejects null here (runs[0].tool.driver.rules must be an array).
	rules := []sarifRule{}
	results := make([]sarifResult, 0, len(in.Findings))

	for _, f := range in.Findings {
		ruleID := f.Gate + "/" + f.RuleID
		if !seenRules[ruleID] {
			seenRules[ruleID] = true
			rules = append(rules, sarifRule{ID: ruleID, Name: f.RuleID})
		}
		res := sarifResult{
			RuleID:  ruleID,
			Level:   sarifLevel(f.Severity),
			Message: sarifMessage{Text: f.Message},
		}
		if f.File != "" {
			loc := sarifLocation{PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: f.File},
			}}
			if f.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{StartLine: f.Line}
			}
			res.Locations = []sarifLocation{loc}
		}
		results = append(results, res)
	}

	log := sarifLog{
		Schema:  sarifSchema,
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "dwarpal",
				InformationURI: "https://github.com/YellowFoxH4XOR/dwarpal",
				Rules:          rules,
			}},
			Results: results,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}
