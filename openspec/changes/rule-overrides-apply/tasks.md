## 1. Mechanism
- [x] 1.1 config: rule_overrides key + namespace-allow in rejectUnknownKeys + severity validation
- [x] 1.2 engine: applySeverityOverrides before the blocking decision (both run modes)
- [x] 1.3 wire cfg.RuleOverrides into check's engine.Options

## 2. Apply + surface
- [x] 2.1 config.PatchRuleOverrides: comment-preserving yaml.Node merge
- [x] 2.2 audit --apply: demotions only; rules annotation
- [x] 2.3 tests: engine override (both modes), config patch (preserve/idempotent/no-file), txtars

## 3. Ship
- [ ] 3.1 full suite; docs (cli.md, configuration.md, CHANGELOG); PR
