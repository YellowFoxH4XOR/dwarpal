# Intent check: change a reviewer would find surprising

`intent/intent-surprise`

## What it catches

The LLM intent gate listed specific changes it judged a human reviewer would not expect given the stated task — each listed in the finding.

## Why this rule exists

"Surprising" is the reviewer's most expensive discovery to make late. Surfacing it pre-commit turns a review-round-trip into an agent retry. Advisory and fail-open.

## How to fix it

For each listed surprise: either it's intended (say so in the commit message) or it isn't (remove it).
