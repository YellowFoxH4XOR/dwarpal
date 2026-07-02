package gitio

import (
	"strconv"
	"strings"
)

// Line is a single added line in the diff: its line number in the new file and
// its text (without the leading '+'). Content gates (secrets, suppressions,
// sql-concat) match against these.
type Line struct {
	Number int
	Text   string
}

// addedContent runs `git diff <selector> -U0` and returns, per new-file path,
// the added lines with their new-file line numbers. Zero context (-U0) keeps
// the output minimal — only the changed lines appear.
func (e *Extractor) addedContent(selector []string) (map[string][]Line, error) {
	out, err := e.run(append([]string{"diff", "-U0", "--no-color", "--no-ext-diff"}, selector...))
	if err != nil {
		return nil, err
	}
	return parseUnifiedAdded(string(out)), nil
}

// parseUnifiedAdded parses unified-diff text into added lines per new path.
//
// It tracks the current file from the "+++ b/<path>" header and the current
// new-file line number from each "@@ ... +start[,count] @@" hunk header. Within
// a hunk, '+' lines are added (consuming a line number), '-' lines are removed
// (not consuming one). Deleted files (+++ /dev/null) contribute nothing.
func parseUnifiedAdded(diff string) map[string][]Line {
	result := map[string][]Line{}
	var curPath string
	var newLine int

	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			curPath = parsePlusPlusPath(line)
		case strings.HasPrefix(line, "@@"):
			newLine = parseHunkNewStart(line)
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			if curPath != "" {
				result[curPath] = append(result[curPath], Line{Number: newLine, Text: line[1:]})
				newLine++
			}
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			// removed line — does not advance the new-file counter
		}
	}
	return result
}

// parsePlusPlusPath extracts the path from a "+++ b/<path>" header, returning
// "" for /dev/null (a deletion).
func parsePlusPlusPath(line string) string {
	p := strings.TrimPrefix(line, "+++ ")
	if p == "/dev/null" {
		return ""
	}
	p = strings.TrimPrefix(p, "b/")
	return p
}

// parseHunkNewStart reads the new-file start line from "@@ -a,b +c,d @@".
func parseHunkNewStart(line string) int {
	plus := strings.Index(line, "+")
	if plus < 0 {
		return 0
	}
	rest := line[plus+1:]
	// rest is like "c,d @@ ..." or "c @@ ..."
	end := strings.IndexAny(rest, ", ")
	if end >= 0 {
		rest = rest[:end]
	}
	n, _ := strconv.Atoi(rest)
	return n
}
