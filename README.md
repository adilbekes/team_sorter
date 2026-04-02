# Team Sorter

Team Sorter divides participants into balanced teams based on ratings, achieving optimal minimum rating difference across all teams.

## Features

- **Optimal balancing**: Uses exact branch-and-bound backtracking algorithm to find the global minimum rating difference across teams
- **Flexible ratings**: Supports participant `rating` as either a decimal or a list of decimals (1.0 to 10.0) with one-decimal precision
- **Placeholder padding**: Automatically adds median-rated placeholder participants when participant count is not divisible by team count
- **Equal team sizes**: Maintains equal team sizes; each team must have at least 1 real (non-placeholder) participant
- **Solution enumeration**: Counts all optimal solutions and randomly selects one per run via reservoir sampling
- **JSON API**: Language-agnostic interface via JSON stdin/stdout or files
- **CLI & Library**: Available as both a standalone binary and importable Go package

## Validation Rules

Input must satisfy:

- `number_of_teams` > 0
- `participants` is not empty
- Number of participants ≥ `number_of_teams` + 1 (minimum 2 real participants per configuration)
- Each participant:
  - Has a unique name (case-insensitive)
  - Name is not empty
  - Rating value(s) are in range [1.0, 10.0] (finite, not NaN/Inf)
- Ratings are all-or-none: either all participants have ratings, or none have ratings
- If `rating` is a list for one participant, all participants must provide the same list length

## Algorithm

The sorter uses **exact backtracking with pruning** to guarantee the globally optimal solution:

1. **Padding**: If `participants % teams != 0`, adds `(teams - remainder)` placeholder participants with the median rating of real participants
2. **Optimal assignment**: Explores all valid assignments respecting:
   - Equal team sizes (or size + 1 for first `remainder` teams)
   - Placeholder cap per team: `ceil(placeholder_count / team_count)`
   - Each team must include ≥ 1 real participant
   - Multi-rating objective: balances per-criterion team spreads (e.g., attack/mid/defense) and total team spread together
3. **Solution counting**: Tracks all distinct assignments that achieve the minimum `rating_diff`
4. **Random selection**: Uses reservoir sampling to randomly pick one among all optimal solutions

## Input Format

```json
{
  "number_of_teams": 3,
  "participants": [
    {"name": "Alice", "rating": [9.5, 8.0, 7.5]},
    {"name": "Bob", "rating": [8.0, 8.5, 7.0]},
    {"name": "Charlie", "rating": [7.5, 7.0, 8.0]},
    {"name": "Diana", "rating": [10.0, 9.0, 8.5]}
  ]
}
```

**Fields:**
- `number_of_teams` (int): Number of teams to create
- `participants` (array):
  - `name` (string): Unique participant identifier
  - `rating` (number or number[]): Skill/strength rating value(s) in range [1.0, 10.0]
    - Number keeps single-rating behavior
    - List enables multi-criteria input; balancing uses the participant's average rating

## Output Format

```json
{
  "teams": [
    {
      "name": "Team 1",
      "total_rating": 18.5,
      "members": [
        {"name": "Alice", "rating": 9.5},
        {"name": "Charlie", "rating": 9.0}
      ]
    },
    ...
  ],
  "meta": {
    "team_count": 3,
    "participant_count": 4,
    "placeholder_count": 2,
    "members_per_team": 2,
    "solution_count": 12,
    "min_team_rating": 17.5,
    "max_team_rating": 19.0,
    "rating_diff": 1.5
  }
}
```

**Team Fields:**
- `name`: Auto-generated team identifier (e.g., "Team 1")
- `total_rating`: Sum of all member ratings
- `members`: List of assigned participants with `is_placeholder` flag (omitted for real participants)

**Meta Fields:**
- `team_count`: Number of teams created
- `participant_count`: Count of real (non-placeholder) participants from input
- `placeholder_count`: Number of auto-added placeholder participants
- `members_per_team`: Exact number of members in each team
- `solution_count`: Total number of distinct optimal solutions with the same minimum `rating_diff`
- `min_team_rating`: Lowest team rating summary
- `max_team_rating`: Highest team rating summary
- `rating_diff`: Difference between max and min summaries (0 = perfect balance)

When input uses `rating` lists with `n` values, each meta rating field becomes a list with `n+1` values:
`[criterion_1, criterion_2, ..., criterion_n, total]`.
The last value is the same "total" metric used for optimization and team balancing.
For single-value ratings, these fields stay as scalars for backward compatibility.

## Usage

### As a Library

```go
import "team_sorter/pkg/teamsorter"

req := teamsorter.SortTeamsRequest{
  NumberOfTeams: 3,
  Participants: []teamsorter.Participant{
    {Name: "Alice", Ratings: []teamsorter.Rating{9.5, 8.0, 7.5}},
    {Name: "Bob", Ratings: []teamsorter.Rating{8.0, 8.5, 7.0}},
  },
}

resp, err := teamsorter.SortTeams(req)
if err != nil {
  log.Fatal(err)
}
fmt.Printf("Optimal solutions: %d\n", resp.Meta.SolutionCount)
```

### As a Binary

#### Build

```bash
go build -o bin/sorter ./cmd/sorter/
```

#### Run with stdin

```bash
echo '{"number_of_teams": 3, "participants": [...]}' | ./bin/sorter
```

#### Run with file input/output

```bash
./bin/sorter -f input.json -o output.json
jq . output.json
```

#### Run demo

```bash
go run ./cmd/demo
```

**CLI Options:**
- `-d '<json>'`: Pass JSON as command-line string
- `-f <file>`: Read JSON from file
- `-o <file>`: Write JSON output to file (default: stdout)
- `-all-name-solutions`: Output all optimal solutions as list items in the form `{ "Team N": ["Name", ...] }`

Example:

```bash
./bin/sorter -f input.json -all-name-solutions | jq .
```

If `solution_count` is `24`, this mode returns a JSON array with `24` objects.

## Example

Given 4 participants and 3 teams:

```json
{
  "number_of_teams": 3,
  "participants": [
    {"name": "Ali", "rating": 10.0},
    {"name": "Mira", "rating": 10.0},
    {"name": "Bek", "rating": 10.0},
    {"name": "Dana", "rating": 9.0}
  ]
}
```

The sorter:
1. Calculates that 2 placeholders are needed (6 total / 3 teams = 2 each)
2. Computes median of [10, 10, 10, 9] = 10.0
3. Adds 2 placeholders with rating 10.0
4. Finds 12 optimal solutions with `rating_diff = 1.0`:
   - Each team has 2 members
   - One team = 19.0 (real + real, e.g., Ali + Dana)
   - Two teams = 20.0 each (real + placeholder)
5. Randomly selects one solution per run

Sample meta output:
```json
{
  "team_count": 3,
  "participant_count": 4,
  "placeholder_count": 2,
  "members_per_team": 2,
  "solution_count": 12,
  "rating_diff": 1.0
}
```

## Error Handling

All validation errors return JSON with an `error` field:

```json
{"error": "participant rating must be between 1.0 and 10.0"}
```

Common errors:
- `"number_of_teams must be greater than 0"`
- `"participants must not be empty"`
- `"participants count must be at least number_of_teams + 1"`
- `"participant rating must be between 1.0 and 10.0"`
- `"all participants must have the same number of rating values"`
- `"participant name must be unique"`
- `"no team sorting solution found"`

## Floating-Point Precision

All ratings and team totals are normalized to one decimal place (e.g., 9.5, 10.0, 6.2). JSON output uses string-formatted decimals to preserve precision:

```json
{"rating": 9.5, "total_rating": 19.0}
```

## Testing

```bash
go test ./...
```

## Performance

- For typical inputs (< 20 participants, 2-6 teams), execution is instantaneous
- Exact algorithm explores all valid combinations; worst case is exponential but pruning (early termination, empty-team symmetry) makes it practical
- Placeholder padding reduces effective participant count without affecting solution quality

## License

See LICENSE file.
