## ADDED Requirements

### Requirement: Pure-Go tree-sitter runtime behind one seam
Dwarpal SHALL provide an `ast-engine` capability wrapping a pure-Go tree-sitter runtime (no cgo — the static-binary promise of §5.5 holds) behind a single internal package; gates and the repo index SHALL access parsing and queries only through this seam, never the third-party dependency directly.

#### Scenario: Static binary preserved
- **WHEN** the dwarpal binary is built with `CGO_ENABLED=0` for any supported platform
- **THEN** the build succeeds and tree-sitter parsing works in the produced binary

### Requirement: Language registry gates AST behavior
The engine SHALL support exactly Go, TypeScript, JavaScript, and Python in this change, and SHALL expose a single `Supports(path)` authority; files of any other language SHALL fall through to pre-existing (regex/heuristic) behavior without error.

#### Scenario: Unsupported language falls through
- **WHEN** a staged diff adds a Ruby file
- **THEN** no AST parsing is attempted for it and regex-tier rules still evaluate it

### Requirement: Query execution over parsed trees
The engine SHALL compile and execute tree-sitter `.scm` queries against parsed trees, returning captured nodes with byte offsets and line numbers usable for findings.

#### Scenario: Function query captures names and lines
- **WHEN** a TypeScript source with a function declaration is parsed and queried for function names
- **THEN** the capture yields the function's name and its 1-indexed line number

### Requirement: Parse failures degrade, never crash
A file that fails to parse SHALL degrade to the heuristic tier for that file (extraction fallback, regex rules) and SHALL never fail the pipeline run.

#### Scenario: Pathological file degrades gracefully
- **WHEN** a TS file with syntax the grammar cannot parse is in the diff
- **THEN** the run completes and that file is handled by the heuristic tier
