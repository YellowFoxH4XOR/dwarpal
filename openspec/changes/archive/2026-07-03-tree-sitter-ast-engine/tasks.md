## 1. Feasibility spike (decision gate)

- [x] 1.1 Vet CGO-free candidates (gotreesitter 526★ pure-Go vs malivvan 3★ wazero) — gotreesitter chosen
- [x] 1.2 Prove parse + `.scm` query for TS/Python/Go under `CGO_ENABLED=0` (7–28 ms first-parse)
- [x] 1.3 Measure binary impact: 31 MB with all 206 grammars — under the 40 MB §5.5 cap

## 2. ast-engine package

- [x] 2.1 `internal/astengine`: `Supports(path)`, `Parse(path, src)`, `Query(tree, src)` seam over gotreesitter; language registry Go/TS/JS/Python
- [x] 2.2 Unit tests: parse+query each language, unsupported-language fallthrough, parse-failure degradation

## 3. repo-index upgrade

- [x] 3.1 TS/JS/Python extractors via astengine function queries (accurate line ranges); heuristics demoted to fallback on parse failure
- [x] 3.2 Import-style fingerprint dimension (Go via go/ast, TS/JS/Python via queries)
- [x] 3.3 Tests: TS class method extraction, Python nested def, fallback-on-parse-failure, import distributions

## 4. AST-precise rules

- [x] 4.1 `no-broad-catch` AST tier (TS/JS catch_clause, Python except_clause; empty-or-no-call handler = finding); suppress regex heuristic for AST-handled files
- [x] 4.2 `no-sql-concat` AST tier (binary `+` with SQL-keyword string operand; template literals / f-strings with interpolation); suppress regex heuristic likewise
- [x] 4.3 Tests incl. non-flagging cases (logged catch, parameterized query, non-SQL template literal)

## 5. drift import-style

- [x] 5.1 Score added import nodes against the fingerprint's dominant form (≥80% majority rule), info severity
- [x] 5.2 Tests: require-in-ESM repo flagged; matching style not flagged; weak-majority repos not flagged

## 6. Verification & ship

- [x] 6.1 Re-run the #68-style benchmark including a TS/Python corpus; assert index build stays within the 2 s budget
- [x] 6.2 Full suite + `-race`; binary size recorded; live multi-language demo (TS duplicate at real lines, Python broad-catch, TS template-literal SQL)
- [x] 6.3 Update V1-CHECKLIST (#24, #25, #28, #29, #37) + CHANGELOG; archive this change; commit through the gate
