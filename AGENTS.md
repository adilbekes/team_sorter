# AGENTS Guide for `team_sorter`

## Big Picture
- Core logic lives in `pkg/teamsorter`; CLI wrappers in `cmd/sorter` and `cmd/demo` should stay thin.
- Main request flow: JSON input -> `ValidateSortTeamsRequest` -> placeholder padding (`withMedianPlaceholders`) -> exact search (`fillTeamsOptimally`) -> response/meta shaping (`buildSortTeamsMeta`).
- `SortTeams` returns one randomly selected optimal solution (reservoir sampling), while `ListOptimalNameSolutions` returns all optimal name-only solutions.
- Team balancing is exact (backtracking), not heuristic: changes to search/pruning can change correctness and solution counts.

## Architecture Boundaries
- `cmd/sorter/main.go`: IO + flags only (`-d`, `-f`, `-o`, `-list`), reports JSON errors to stderr and optionally output file.
- `pkg/teamsorter/validator.go`: strict business validation; keep new input rules here and expose user-facing errors from `errors.go`.
- `pkg/teamsorter/types.go`: JSON contracts and one-decimal normalization; custom marshaling preserves backward compatibility.
- `pkg/teamsorter/sorter.go`: optimization, placeholder generation, deduplication of placeholder-swapped solutions.

## Project-Specific Conventions
- Ratings are normalized to one decimal via `normalizeRating`; do not bypass it when mutating totals/metrics.
- `Participant.Ratings` is internal; external JSON uses `rating` as scalar or list (see custom marshal/unmarshal in `types.go`).
- If input has no ratings, random ratings are generated in `[1.0, 10.0]`, but output hides rating fields (`SortTeamsResponse.HasRatings=false` path).
- Reserved names matching `^placeholder\s+\d+$` (case-insensitive) are rejected to avoid collision with generated placeholders.
- Multi-rating optimization compares objective in this order: max diff -> avg diff -> sum diff -> per-metric diffs (`objectiveBetter`).

## Developer Workflows
- Build CLI: `go build -o bin/sorter ./cmd/sorter/`
- Run from file: `go run ./cmd/sorter -f example.json | jq .`
- List all optimal solutions: `go run ./cmd/sorter -f example.json -list | jq .`
- Demo run: `go run ./cmd/demo`
- Tests: `go test ./...` (currently passes)
- Quick manual checks: `./test_ratings.sh` (expects `jq` and absolute project path in script)

## Integration Notes
- Public package import path is `team_sorter/pkg/teamsorter` (module in `go.mod` is `team_sorter`).
- CLI is designed for language-agnostic integration through stdin/stdout JSON.
- Error contract is plain JSON object: `{ "error": "..." }`; preserve exact messages where possible for callers.
- `meta.min_team_rating`, `meta.max_team_rating`, `meta.rating_diff` are scalar for single-rating input and arrays for multi-rating input.

## Safe Change Checklist for Agents
- Update tests in `pkg/teamsorter/sorter_test.go` whenever changing validation, objective ranking, placeholder logic, or JSON shape.
- Keep `README.md` examples aligned with flag behavior and output fields.
- If touching search logic (`fillTeamsOptimally`), verify both optimality and `solution_count` semantics.
- Preserve dedup behavior (`solutionSignature` + `deduplicateSolutions`) so placeholder-only permutations are not overcounted.

