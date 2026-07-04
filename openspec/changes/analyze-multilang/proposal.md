# Richer non-Go analyze output

## Why

`dwarpal analyze` gave non-Go languages only their import form, while Go got
function count, size, and naming. With per-language function stats now available
(FuncByLang), analyze can give Python/TS/JS the same rich fingerprint — better
facts for the agent authoring .dwarpal.yml on a non-Go repo.

## What changes

- analyze reports per language: function count, average function size, and the
  learned dominant naming style (snake_case vs camelCase), for every language,
  in both the human table and `--json`.

## Notes

- Third of the language-parity sequence (after architecture_rules and drift).
