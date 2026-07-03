package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	yaml "go.yaml.in/yaml/v3"
)

// PatchRuleOverrides merges add ("gate/rule_id" → severity) into the
// rule_overrides block of the config file at root, preserving the rest of the
// file including comments (a yaml.Node round-trip, not a rewrite). It creates
// the rule_overrides block if absent. Existing entries with the same key are
// updated; others are left alone. Returns an error if the file doesn't exist —
// audit --apply tunes an existing config, it does not create one.
func PatchRuleOverrides(root string, add map[string]string) error {
	if len(add) == 0 {
		return nil
	}
	path := filepath.Join(root, Filename)
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w — run `dwarpal init` first", Filename, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("%s: %w", Filename, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%s: expected a YAML mapping at the top level", Filename)
	}
	top := doc.Content[0]

	ov := valueFor(top, "rule_overrides")
	if ov == nil {
		ov = &yaml.Node{Kind: yaml.MappingNode}
		top.Content = append(top.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "rule_overrides"}, ov)
	}
	keys := make([]string, 0, len(add))
	for k := range add {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic output
	for _, k := range keys {
		setScalar(ov, k, add[k])
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) // match the starter config's 2-space style
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	enc.Close()
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// valueFor returns the value node for key in a mapping node, or nil.
func valueFor(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// setScalar sets key→val in a mapping node, updating in place or appending.
// The key is force-quoted because "gate/rule_id" contains a slash.
func setScalar(m *yaml.Node, key, val string) {
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: val, Style: yaml.DoubleQuotedStyle}
	if existing := valueFor(m, key); existing != nil {
		existing.Value = val
		existing.Style = yaml.DoubleQuotedStyle
		existing.Tag = ""
		return
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key, Style: yaml.DoubleQuotedStyle}, valNode)
}
