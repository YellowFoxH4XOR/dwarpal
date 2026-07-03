package repoindex

import (
	"regexp"
	"strings"

	"github.com/YellowFoxH4XOR/dwarpal/internal/astengine"
)

// Import-style fingerprint dimension (tree-sitter-ast-engine change, D6).
// The index counts import forms per language; the drift gate compares added
// imports against the dominant form.

// Import form labels. Kept as strings so the fingerprint marshals trivially
// and new languages add forms without a type change.
const (
	FormESNamed     = "es-named"     // import { x } from 'y'
	FormESDefault   = "es-default"   // import x from 'y'
	FormESNamespace = "es-namespace" // import * as x from 'y'
	FormRequire     = "require"      // const x = require('y')
	FormPyImport    = "import"       // import os
	FormPyFrom      = "from-import"  // from os import path
	FormGoGrouped   = "grouped"      // import ( ... )
	FormGoSingle    = "single"       // import "x"
)

// importQueries capture import-ish statements per language.
var importQueries = map[astengine.Language]string{
	astengine.LangTS: `[(import_statement) @i (lexical_declaration) @i]`,
	astengine.LangJS: `[(import_statement) @i (lexical_declaration) @i]`,
	astengine.LangPy: `[(import_statement) @i (import_from_statement) @i]`,
}

// ClassifyImportLine maps one source line to an import form, or "" when the
// line is not an import. Shared by the fingerprint (below) and the drift
// gate's added-line scoring, so both sides speak the same labels.
func ClassifyImportLine(lang astengine.Language, line string) string {
	t := strings.TrimSpace(line)
	switch lang {
	case astengine.LangTS, astengine.LangJS:
		switch {
		case requirePattern.MatchString(t):
			return FormRequire
		case strings.HasPrefix(t, "import * as "):
			return FormESNamespace
		case strings.HasPrefix(t, "import {") || strings.HasPrefix(t, "import type {"):
			return FormESNamed
		case strings.HasPrefix(t, "import ") && strings.Contains(t, " from "):
			return FormESDefault
		}
	case astengine.LangPy:
		switch {
		case strings.HasPrefix(t, "from ") && strings.Contains(t, " import "):
			return FormPyFrom
		case strings.HasPrefix(t, "import "):
			return FormPyImport
		}
	}
	return ""
}

var requirePattern = regexp.MustCompile(`\brequire\s*\(`)

// countImportsFromTree adds a file's import forms to the fingerprint using an
// already-parsed tree (nil tree = parse failed: degrade to line scanning).
func (idx *Index) countImportsFromTree(tree *astengine.Tree, rel string, src []byte) {
	lang := astengine.LanguageFor(rel)
	if lang == "" {
		return
	}
	for _, line := range importStatementLines(tree, src, lang) {
		if form := ClassifyImportLine(lang, line); form != "" {
			idx.addImport(string(lang), form)
		}
	}
}

// importStatementLines yields the first line of each import-ish statement —
// via tree-sitter captures when the file parses, else every source line (the
// classifier ignores non-imports, so over-feeding is safe).
func importStatementLines(tree *astengine.Tree, src []byte, lang astengine.Language) []string {
	if tree != nil {
		if caps, qerr := tree.Query(importQueries[lang]); qerr == nil {
			var out []string
			for _, c := range caps {
				first := c.Text
				if i := strings.IndexByte(first, '\n'); i >= 0 {
					first = first[:i]
				}
				out = append(out, first)
			}
			return out
		}
	}
	return strings.Split(string(src), "\n")
}

// addImport records one import form occurrence.
func (idx *Index) addImport(lang, form string) {
	if idx.Conventions.Imports == nil {
		idx.Conventions.Imports = map[string]map[string]int{}
	}
	if idx.Conventions.Imports[lang] == nil {
		idx.Conventions.Imports[lang] = map[string]int{}
	}
	idx.Conventions.Imports[lang][form]++
}

// DominantImportForm returns the language's dominant form and its share, or
// ("", 0) when the language has too few imports to have a meaningful norm.
func (c Conventions) DominantImportForm(lang string) (string, float64) {
	forms := c.Imports[lang]
	total := 0
	best, bestN := "", 0
	for form, n := range forms {
		total += n
		if n > bestN {
			best, bestN = form, n
		}
	}
	if total < 5 { // no meaningful majority in a tiny sample
		return "", 0
	}
	return best, float64(bestN) / float64(total)
}
