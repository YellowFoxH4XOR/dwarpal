package census

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// DefaultBaselinePath is where the committed baseline lives. It sits OUTSIDE
// .dwarpal/cache/ (the only .dwarpal path .gitignore excludes) precisely so it
// is committed: the ratchet baseline must travel with the repo for PRs to diff
// against it.
const DefaultBaselinePath = ".dwarpal/baseline.json"

// baselineVersion guards the on-disk format for future migrations.
const baselineVersion = 1

// Baseline is the committed record of accepted whole-repo decay counts. It is
// machine-owned, so it is plain JSON (unlike .dwarpal.yml, which is hand-edited
// and needs comment-preserving round-trips).
type Baseline struct {
	Version   int                     `json:"version"`
	Detectors map[string]BaselineEntry `json:"detectors"`
}

// BaselineEntry is one detector's accepted count and the item identities behind
// it (so the ratchet can name only the genuinely NEW items on an increase).
type BaselineEntry struct {
	Count int      `json:"count"`
	Items []string `json:"items"`
}

// LoadBaseline reads the baseline at path (joined under root). A missing file
// returns (nil, nil): the caller decides whether that is fatal (it is, for
// --check — there is nothing to ratchet against) or expected.
func LoadBaseline(root, path string) (*Baseline, error) {
	full := filepath.Join(root, path)
	raw, err := os.ReadFile(full)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var b Baseline
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &b, nil
}

// WriteBaseline persists the report's non-skipped detectors as the new accepted
// baseline. Skipped detectors are omitted — recording a count we could not
// measure would let a later run silently "improve" against a phantom number.
func WriteBaseline(root, path string, rep *Report) error {
	b := Baseline{Version: baselineVersion, Detectors: map[string]BaselineEntry{}}
	for _, d := range rep.Detectors {
		if d.Skipped {
			continue
		}
		items := append([]string(nil), d.Items...)
		sort.Strings(items)
		b.Detectors[d.Name] = BaselineEntry{Count: d.Count, Items: items}
	}

	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, raw, 0o644)
}
