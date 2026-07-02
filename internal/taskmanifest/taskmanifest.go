// Package taskmanifest reads and writes the per-branch task scope declaration.
//
// A task manifest (.dwarpal-task.yml) declares what a change is allowed to touch
// so Gate 4 (scope) can block files outside it. It is written by `dwarpal task`
// or by an agent wrapper, and lives on the branch alongside the work.
package taskmanifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Filename is the manifest file at the repo root.
const Filename = ".dwarpal-task.yml"

// Manifest is a declared task scope.
type Manifest struct {
	ID    string   `koanf:"id"`
	Paths []string `koanf:"paths"`
}

// Load reads the manifest at root. The bool is false (with nil error) when no
// manifest exists — the caller decides whether that is warn-only or a block.
func Load(root string) (Manifest, bool, error) {
	path := filepath.Join(root, Filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Manifest{}, false, nil
	}
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return Manifest{}, false, fmt.Errorf("reading %s: %w", Filename, err)
	}
	var m Manifest
	if err := k.Unmarshal("", &m); err != nil {
		return Manifest{}, false, fmt.Errorf("parsing %s: %w", Filename, err)
	}
	return m, true, nil
}

// Write persists a manifest at root. Kept as a hand-written template (the schema
// is two fields) to avoid pulling in a YAML marshaller.
func Write(root, id string, paths []string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "id: %q\n", id)
	b.WriteString("paths:\n")
	for _, p := range paths {
		fmt.Fprintf(&b, "  - %q\n", p)
	}
	return os.WriteFile(filepath.Join(root, Filename), []byte(b.String()), 0o644)
}
