## ADDED Requirements

### Requirement: Off by default, explicit opt-in
Gate 7 (intent verification) SHALL NOT run unless `gates.intent_check.enabled: true` is set in `.dwarpal.yml`. With no config file, or with `intent_check` absent/`enabled: false`, the gate SHALL be skipped entirely and make no network connection.

#### Scenario: Default config skips the gate
- **WHEN** `dwarpal check` runs with no `.dwarpal.yml` or `intent_check.enabled` unset
- **THEN** no intent-verification network call is made and no `intent_check` finding appears in the report

#### Scenario: Explicit opt-in enables the gate
- **WHEN** `.dwarpal.yml` sets `gates.intent_check.enabled: true` with a valid `provider` and a task manifest is present
- **THEN** the gate runs and its verdict is included in the report

### Requirement: Diff-only payload to the provider
When enabled, Gate 7 SHALL send only the unified diff and the task manifest text to the configured provider. It SHALL NOT read or transmit repository files outside the diff, and SHALL NOT use `RepoIndex`.

#### Scenario: Payload scoped to the diff
- **WHEN** Gate 7 runs against a staged diff with a task manifest present
- **THEN** the outbound request body contains only the diff text and the manifest text, with no other repository file contents

### Requirement: Provider abstraction supports Anthropic, OpenAI, and OpenAI-compatible endpoints
Gate 7 SHALL support `provider: anthropic`, `provider: openai`, and `provider: openai-compatible` (the latter accepting a user-supplied `endpoint`, usable for local models such as Ollama). Provider selection and credentials SHALL come from configuration and/or environment variables, never hardcoded or embedded in the binary.

#### Scenario: OpenAI-compatible local endpoint
- **WHEN** `.dwarpal.yml` sets `provider: openai-compatible` and `endpoint: http://localhost:11434/v1`
- **THEN** Gate 7 sends its verification request to the configured local endpoint instead of any hosted provider

#### Scenario: Missing credentials
- **WHEN** `intent_check.enabled: true` with `provider: anthropic` but no API key is available via env or config
- **THEN** the gate reports an infra error per the fail-open requirement below rather than crashing the process

### Requirement: Infra failures fail open; verdicts fail normally
A Gate 7 infrastructure failure (timeout, network error, auth error, rate limit, malformed provider response) SHALL produce a `warn`-severity finding and SHALL NOT by itself cause `dwarpal check` to exit 1. A successful provider response carrying a "does not match intent" verdict SHALL produce a normal blocking (`error`-severity) finding like any other gate, in `enforce` mode.

#### Scenario: Provider outage does not block
- **WHEN** the configured provider times out or is unreachable during `dwarpal check`
- **THEN** the process does not exit 1 solely because of the intent gate, and the report includes a `warn`-severity finding naming the infra failure

#### Scenario: Negative verdict blocks normally
- **WHEN** the provider successfully responds with a verdict that the diff does not accomplish the stated intent
- **THEN** the gate emits an `error`-severity finding and, in `enforce` mode, `dwarpal check` exits 1

### Requirement: Hard timeout and token cap
Gate 7 SHALL enforce a configurable hard timeout (default 30s) on the provider call and a token cap on the payload sent. Exceeding either SHALL be treated as an infra failure under the fail-open rule.

#### Scenario: Timeout enforced
- **WHEN** the provider does not respond within the configured timeout (default 30s)
- **THEN** the gate aborts the call, treats it as an infra failure, and does not block the commit on that basis alone
