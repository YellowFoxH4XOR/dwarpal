//go:build race

package repoindex

// raceEnabled lets benchmarks skip wall-clock assertions under -race, whose
// instrumentation slows execution 5-10x — the numbers are not the product's.
const raceEnabled = true
