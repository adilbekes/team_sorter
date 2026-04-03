# Architecture Guide for `team_sorter`

## Overview

Team Sorter is a Go library and CLI tool for optimally dividing participants into balanced teams based on ratings. The system employs a branch-and-bound backtracking algorithm for rated inputs and randomized assignment for unrated inputs.

## Project Structure

```
team_sorter/
├── cmd/
│   ├── sorter/           # CLI binary entry point
│   │   └── main.go       # Command-line argument parsing and I/O handling
│   └── demo/             # Example/demonstration binary
├── pkg/
│   └── teamsorter/       # Core library package
│       ├── types.go      # Data structures (Rating, Participant, Team, etc.)
│       ├── sorter.go     # Main algorithm implementation
│       ├── validator.go  # Input validation logic
│       ├── errors.go     # Custom error definitions
│       └── format.go     # Utility formatting functions
├── docs/                 # Documentation for agents
│   ├── ARCHITECTURE.md   # This file
│   ├── CONVENTIONS.md    # Coding conventions
│   ├── WORKFLOWS.md      # Build/test workflows
│   ├── INTEGRATION.md    # External dependencies
│   └── SAFE_CHANGES.md   # Safe modification checklist
└── go.mod               # Go module definition
```

## Component Architecture

### 1. **CLI Layer** (`cmd/sorter/main.go`)
- **Responsibility**: Parse command-line arguments and manage I/O
- **Inputs**: JSON from stdin, `-d` flag, or `-f` file argument
- **Outputs**: JSON to stdout or `-o` file; errors to stderr
- **Key Features**:
  - Three input modes: stdin, string flag, file input
  - Two output modes: stdout, file output
  - Optional `-list` flag for enumerating all optimal solutions
  - Error reporting via JSON `{"error": "..."}` format

### 2. **Core Library** (`pkg/teamsorter/`)

#### **Types Module** (`types.go`)
Defines core data structures:
- `Rating`: Custom type wrapping `float64`, normalized to 1 decimal place
- `Participant`: Represents a person with name and optional ratings
- `Team`: Group of participants with total rating calculation
- `SortTeamsRequest`: Input structure (number_of_teams, participants)
- `SortTeamsResponse`: Output structure (teams, meta, error)
- `MetaRating`: Flexible rating wrapper supporting single values or arrays

**Key Behaviors**:
- JSON marshaling/unmarshaling with custom handling for single vs. multi-rating scenarios
- Rating normalization (1 decimal precision) via `normalizeRating()`
- Participant score calculation as average of all ratings

#### **Sorter Module** (`sorter.go`)
Implements the core algorithm:

**Main Entry Points**:
1. `SortTeams(req)` → `*SortTeamsResponse`: Single optimal solution with reservoir sampling
2. `ListOptimalNameSolutions(req)` → `[]map[string][]string`: All distinct optimal solutions

**Algorithm Paths**:
- **Rated Input** (any participant has ratings):
  - Sorts participants by average score descending
  - Uses `fillTeamsOptimally()` with exact backtracking algorithm
  - Tracks all solutions via deduplication
  - Randomly selects one via reservoir sampling
  
- **Unrated Input** (no participants have ratings):
  - Uses `fillTeamsRandomlyUnrated()` for quick random assignment
  - OR `listAllNameSolutionsUnrated()` to enumerate all permutations

**Key Functions**:
- `withMedianPlaceholders()`: Adds placeholders to fill incomplete teams
- `fillTeamsOptimally()`: Branch-and-bound backtracking with optimization objective
- `computeOptimizationObjective()`: Multi-criteria balancing (max diff, avg diff, sum diff)
- `randomSeed()`: Cryptographically secure per-call RNG seeding
- `solutionSignature()`: Canonical representation for deduplication

#### **Validator Module** (`validator.go`)
Validates input before processing:
- Team count ≥ 2
- At least 1 participant
- Participants ≥ teams + 1
- Unique (case-insensitive) participant names
- Reserved placeholder name pattern check
- Rating values in [1.0, 10.0], finite
- Consistent rating presence (all-or-none)
- Consistent rating count across participants

#### **Error Module** (`errors.go`)
Pre-defined error messages:
- `ErrInvalidTeamCount`: number_of_teams < 2
- `ErrInvalidParticipants`: empty participants list
- `ErrInsufficientParticipants`: fewer participants than teams + 1
- `ErrEmptyParticipantName`: blank participant name
- `ErrReservedPlaceholderName`: name matches "Placeholder N" pattern
- `ErrInvalidParticipantRating`: rating outside [1.0, 10.0]
- `ErrInconsistentRatings`: mixed rated/unrated participants
- `ErrInconsistentRatingCount`: differing rating list lengths
- `ErrDuplicateParticipantName`: non-unique names
- `ErrNoSolution`: algorithm found no valid assignment

#### **Format Module** (`format.go`)
Generic utility:
- `FormatItems[T]()`: Joins stringable items into comma-separated string

## Data Flow

### Rated Input Flow (Optimal Assignment)

```
SortTeamsRequest
    ↓
[Validation] → Error on invalid input
    ↓
[Copy Participants] → Immutable original
    ↓
[Add Median Placeholders] → Balanced team sizes
    ↓
[Sort by Score] → Deterministic ordering for symmetry pruning
    ↓
[fillTeamsOptimally] → Backtracking search
    ├─ [Explore Assignments] → Build teams recursively
    ├─ [Compute Objective] → Max/avg/sum rating differences
    ├─ [Prune Symmetries] → Skip empty team duplicates
    ├─ [Track Solutions] → Deduplicate via signature
    └─ [Reservoir Sampling] → Random selection
    ↓
[Build Meta] → Statistics (solution_count, rating_diff, etc.)
    ↓
SortTeamsResponse (1 solution)
```

### Unrated Input Flow (Random Assignment)

```
SortTeamsRequest (no ratings)
    ↓
[Validation] → Error on invalid input
    ↓
[Add Placeholders] → No median (unrated)
    ↓
[fillTeamsRandomlyUnrated or listAllNameSolutionsUnrated]
    ├─ [Shuffle Participants] → Per-call RNG seeding
    ├─ [Assign Greedily] → Fill teams randomly
    └─ [Verify Constraints] → Min real member per team
    ↓
[Build Meta] → Count teams, placeholders
    ↓
SortTeamsResponse or List of Solutions
```

## Key Design Decisions

1. **Per-Call RNG Seeding**: Uses `crypto/rand` to seed each call independently, preventing repeated output across fast process starts.

2. **Placeholder Deduplication**: Solutions differing only by placeholder swaps (same ratings) are treated as identical via `solutionSignature()`.

3. **Multi-Criteria Optimization**: For multi-rating input, optimizes:
   - Per-criterion spreads (attack, mid, defense, etc.)
   - Average rating spread
   - Prioritizes max diff, then avg diff, then sum diff

4. **Floating-Point Precision**: All ratings normalized to 1 decimal place to avoid precision issues in JSON and comparisons.

5. **JSON Flexibility**: Output adapts based on input—omits rating fields if input had no ratings; outputs arrays for multi-rating, scalars for single-rating.

6. **Symmetry Pruning**: Skips assignments to empty teams when previous teams are also empty (only true duplicates), but allows assignments once teams have members (since members differ).

## Constraints and Boundaries

- **Team Count**: 2 to N (practical: 2-10)
- **Participant Count**: team_count + 1 to ~20 (algorithm exponential but practical with pruning)
- **Rating Values**: [1.0, 10.0], one decimal precision
- **Placeholder Count**: Minimal; only added if participants % teams ≠ 0
- **Real Participants Per Team**: At least 1 required per team

## Error Propagation

All errors are:
1. Caught at validation stage (early return)
2. Returned as `error` interface to caller
3. Converted to JSON `{"error": "message"}` in CLI layer
4. Exit code 1 on error, 0 on success in CLI

## Performance Characteristics

- **Best case** (unrated): O(n!) with random shuffling; bounded by capacities
- **Worst case** (rated): O(n!) backtracking; pruning reduces effective complexity significantly
- **Typical** (5-20 participants, 2-6 teams): < 10ms execution
- **Memory**: O(n * t) for team structures + recursion stack depth O(n)

## Extension Points

1. **Custom Rating Types**: Modify `Rating` type or add new fields to `Participant`
2. **Alternative Algorithms**: Replace `fillTeamsOptimally()` logic
3. **New Optimization Objectives**: Modify `computeOptimizationObjective()` and comparison functions
4. **CLI Modes**: Extend `cmd/sorter/main.go` for additional flags/output formats

