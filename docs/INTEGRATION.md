# Integration Guide for `team_sorter`

This document outlines integration notes for external systems, package imports, and library usage patterns.

## Module Information

- **Module Path**: `team_sorter`
- **Go Version**: 1.25.5+
- **External Dependencies**: None (standard library only)
- **License**: See LICENSE file

## Library Import

### Basic Import

```go
import "team_sorter/pkg/teamsorter"
```

### Usage Example

```go
package main

import (
    "fmt"
    "log"
    "team_sorter/pkg/teamsorter"
)

func main() {
    req := teamsorter.SortTeamsRequest{
        NumberOfTeams: 2,
        Participants: []teamsorter.Participant{
            {Name: "Alice", Ratings: []teamsorter.Rating{9.5}},
            {Name: "Bob", Ratings: []teamsorter.Rating{8.0}},
            {Name: "Charlie", Ratings: []teamsorter.Rating{7.5}},
        },
    }
    
    resp, err := teamsorter.SortTeams(req)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Teams: %d, Solution Count: %d\n", 
        resp.Meta.TeamCount, 
        resp.Meta.SolutionCount)
}
```

## Public API Surface

### Main Functions

#### `SortTeams()`

```go
func SortTeams(req SortTeamsRequest) (*SortTeamsResponse, error)
```

**Purpose**: Divides participants into balanced teams, returning one optimal solution.

**Inputs**:
- `SortTeamsRequest`: Number of teams and list of participants

**Outputs**:
- `*SortTeamsResponse`: Team assignments, metadata, or error message
- `error`: Non-nil if validation fails

**Behavior**:
- For rated input: Uses exact backtracking to find optimal solution
- For unrated input: Uses random assignment
- Each call uses independent RNG seeding

**Example**:
```go
resp, err := teamsorter.SortTeams(req)
if err != nil {
    return fmt.Errorf("sorting failed: %w", err)
}
fmt.Printf("Solution count: %d, Rating diff: %.1f\n",
    resp.Meta.SolutionCount,
    resp.Meta.RatingDiff)
```

#### `ListOptimalNameSolutions()`

```go
func ListOptimalNameSolutions(req SortTeamsRequest) ([]map[string][]string, error)
```

**Purpose**: Returns all distinct optimal team assignments as name-only maps.

**Inputs**:
- `SortTeamsRequest`: Same as `SortTeams()`

**Outputs**:
- `[]map[string][]string`: Array of solutions; each solution has team name → member names
- `error`: Non-nil if validation fails

**Behavior**:
- For rated input: Returns all solutions with the same minimum rating difference
- For unrated input: Returns all valid permutations (can be very large)
- Solutions differing only by placeholder swaps are deduplicated

**Example**:
```go
solutions, err := teamsorter.ListOptimalNameSolutions(req)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d optimal solutions\n", len(solutions))
for i, solution := range solutions {
    fmt.Printf("Solution %d: %v\n", i+1, solution)
}
```

#### `ValidateSortTeamsRequest()`

```go
func ValidateSortTeamsRequest(req SortTeamsRequest) error
```

**Purpose**: Validates input before processing.

**Returns**: First validation error encountered, or nil if valid.

**Usage**: Call manually before `SortTeams()` if you want to validate separately.

```go
if err := teamsorter.ValidateSortTeamsRequest(req); err != nil {
    return fmt.Errorf("invalid input: %w", err)
}
```

### Types

#### `SortTeamsRequest`

```go
type SortTeamsRequest struct {
    NumberOfTeams int           `json:"number_of_teams"`
    Participants  []Participant `json:"participants"`
}
```

Construction:
```go
req := teamsorter.SortTeamsRequest{
    NumberOfTeams: 3,
    Participants: []teamsorter.Participant{
        {Name: "Alice", Ratings: []teamsorter.Rating{9.5, 8.0}},
        {Name: "Bob", Ratings: []teamsorter.Rating{8.0, 9.0}},
    },
}
```

#### `Participant`

```go
type Participant struct {
    Name          string        `json:"name"`
    Ratings       []Rating      `json:"-"`              // Not marshaled
    IsPlaceholder bool          `json:"is_placeholder,omitempty"`
}
```

Methods:
- `HasRating() bool`: Returns true if Ratings is non-empty
- `RatingCount() int`: Returns length of Ratings slice
- `Score() Rating`: Returns average of all ratings
- `String() string`: Human-readable representation

Construction:
```go
// Single rating
p := teamsorter.Participant{
    Name: "Alice",
    Ratings: []teamsorter.Rating{9.5},
}

// Multi-rating
p := teamsorter.Participant{
    Name: "Bob",
    Ratings: []teamsorter.Rating{9.0, 8.5, 7.0},
}

// No rating (unrated)
p := teamsorter.Participant{
    Name: "Charlie",
    Ratings: []teamsorter.Rating{},  // or nil
}
```

#### `SortTeamsResponse`

```go
type SortTeamsResponse struct {
    Teams      []Team         `json:"teams,omitempty"`
    Meta       *SortTeamsMeta `json:"meta,omitempty"`
    Error      string         `json:"error,omitempty"`
    HasRatings bool           `json:"-"`              // Internal flag
}
```

Fields:
- `Teams`: Slice of assigned teams
- `Meta`: Metadata about the solution
- `Error`: Non-empty if an error occurred
- `HasRatings`: Indicates whether input had ratings (affects JSON output)

Access pattern:
```go
if resp.Error != "" {
    log.Fatal(resp.Error)
}
for _, team := range resp.Teams {
    fmt.Printf("%s: %+v\n", team.Name, team.Members)
}
```

#### `Team`

```go
type Team struct {
    Name        string        `json:"name"`
    TotalRating Rating        `json:"total_rating"`
    Members     []Participant `json:"members"`
}
```

Methods:
- `String() string`: Formatted team summary

#### `Rating`

```go
type Rating float64
```

Behavior:
- Automatically normalized to 1 decimal place
- JSON marshals as formatted string (e.g., "9.5")
- Supports arithmetic operations like `float64`

Conversions:
```go
r := teamsorter.Rating(9.5)
f := float64(r)  // Convert to float64
```

#### `SortTeamsMeta`

```go
type SortTeamsMeta struct {
    TeamCount        int        `json:"team_count"`
    ParticipantCount int        `json:"participant_count"`
    PlaceholderCount int        `json:"placeholder_count"`
    MembersPerTeam   int        `json:"members_per_team"`
    SolutionCount    int        `json:"solution_count"`
    MinTeamRating    MetaRating `json:"min_team_rating"`
    MaxTeamRating    MetaRating `json:"max_team_rating"`
    RatingDiff       MetaRating `json:"rating_diff"`
}
```

Fields:
- `TeamCount`: Number of teams created
- `ParticipantCount`: Count of real (non-placeholder) participants
- `PlaceholderCount`: Auto-added placeholders to balance teams
- `MembersPerTeam`: Size of each team (equal across all)
- `SolutionCount`: Number of distinct optimal solutions
- `MinTeamRating`: Lowest team total rating (scalar or array)
- `MaxTeamRating`: Highest team total rating (scalar or array)
- `RatingDiff`: Difference between max and min (rating_diff metric)

For multi-rating input (e.g., 3 criteria), these are `[]Rating` with values: `[criterion1, criterion2, criterion3, avg]`.

#### `MetaRating`

```go
type MetaRating struct {
    Values []Rating
}
```

Behavior:
- Marshals as scalar if single value, array if multiple
- Used for flexible single/multi-rating output

Access pattern:
```go
// Type-safe access
minValues := resp.Meta.MinTeamRating.Values
// For single-rating: []Rating with 1 value
// For multi-rating: []Rating with N+1 values (criteria + avg)
```

### Error Variables

Pre-defined errors in `errors.go`:

```go
var (
    ErrInvalidTeamCount         // number_of_teams < 2
    ErrInvalidParticipants      // empty participants list
    ErrTooManyTeams             // teams > participants
    ErrInsufficientParticipants // < teams + 1 participants
    ErrEmptyParticipantName     // blank name
    ErrReservedPlaceholderName  // matches "Placeholder N"
    ErrInvalidParticipantRating // rating not in [1.0, 10.0]
    ErrInconsistentRatings      // mixed rated/unrated
    ErrInconsistentRatingCount  // different rating list lengths
    ErrDuplicateParticipantName // non-unique names
    ErrNoSolution               // algorithm found no valid assignment
)
```

Check errors:
```go
if errors.Is(err, teamsorter.ErrInvalidTeamCount) {
    fmt.Println("Need at least 2 teams")
}
```

## JSON I/O

### Input Format

```json
{
  "number_of_teams": 3,
  "participants": [
    {"name": "Alice", "rating": 9.5},
    {"name": "Bob", "rating": [8.0, 7.5, 9.0]},
    {"name": "Charlie", "rating": 8.2}
  ]
}
```

Key behaviors:
- `rating` can be a single number or array
- If any participant has a rating array, all must have the same length
- All participants must have ratings or none must have ratings
- Missing `rating` field is treated as unrated

### Output Format

```json
{
  "teams": [
    {
      "name": "Team 1",
      "total_rating": 19.5,
      "members": [
        {"name": "Alice", "rating": 9.5},
        {"name": "Charlie", "rating": 10.0}
      ]
    }
  ],
  "meta": {
    "team_count": 3,
    "participant_count": 3,
    "placeholder_count": 0,
    "members_per_team": 1,
    "solution_count": 12,
    "min_team_rating": 18.0,
    "max_team_rating": 19.5,
    "rating_diff": 1.5
  }
}
```

Unrated output (no rating fields):
```json
{
  "teams": [
    {
      "name": "Team 1",
      "members": [
        {"name": "Alice"},
        {"name": "Charlie"}
      ]
    }
  ],
  "meta": {
    "team_count": 2,
    "participant_count": 2,
    "placeholder_count": 0,
    "members_per_team": 1,
    "solution_count": 1
  }
}
```

Error output:
```json
{"error": "participant rating must be between 1.0 and 10.0"}
```

### Unmarshaling JSON to Request

```go
var req teamsorter.SortTeamsRequest
if err := json.Unmarshal(data, &req); err != nil {
    log.Fatal(err)
}
```

### Marshaling Response to JSON

```go
resp, _ := teamsorter.SortTeams(req)
data, err := json.Marshal(resp)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(data))
```

## CLI Binary Integration

### Subprocess Invocation

```go
package main

import (
    "encoding/json"
    "os/exec"
)

func main() {
    input := map[string]interface{}{
        "number_of_teams": 2,
        "participants": []map[string]interface{}{
            {"name": "Alice", "rating": 9.5},
        },
    }
    
    data, _ := json.Marshal(input)
    cmd := exec.Command("./bin/sorter")
    cmd.Stdin = bytes.NewReader(data)
    
    output, err := cmd.Output()
    if err != nil {
        log.Fatal(err)
    }
    
    var resp map[string]interface{}
    json.Unmarshal(output, &resp)
    fmt.Printf("Teams: %v\n", resp["teams"])
}
```

### Shell Integration

```bash
#!/bin/bash

# Invoke with JSON string
./bin/sorter -d '{"number_of_teams": 2, "participants": [{"name": "Alice", "rating": 9.5}]}'

# Invoke with file
./bin/sorter -f input.json -o output.json

# Pipe JSON
cat input.json | ./bin/sorter | jq '.meta.solution_count'

# Check exit code
if ./bin/sorter -f input.json > /dev/null; then
    echo "Success"
else
    echo "Failed with exit code $?"
fi
```

## Language Bindings

### Calling from Other Languages

Since team_sorter is a Go binary, integrate via:

1. **CLI (JSON I/O)**: Call binary with JSON input/output
2. **HTTP Wrapper**: Wrap sorter in a simple HTTP server
3. **Cgo** (advanced): Compile as shared library; not recommended for this use case

### Example: HTTP Wrapper

```go
package main

import (
    "encoding/json"
    "net/http"
    "team_sorter/pkg/teamsorter"
)

func sortHandler(w http.ResponseWriter, r *http.Request) {
    var req teamsorter.SortTeamsRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    resp, err := teamsorter.SortTeams(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func main() {
    http.HandleFunc("/sort", sortHandler)
    http.ListenAndServe(":8080", nil)
}
```

## Testing Integration

### Mock Testing

```go
func TestSortTeamsIntegration(t *testing.T) {
    req := teamsorter.SortTeamsRequest{
        NumberOfTeams: 2,
        Participants: []teamsorter.Participant{
            {Name: "Alice", Ratings: []teamsorter.Rating{9.5}},
            {Name: "Bob", Ratings: []teamsorter.Rating{8.0}},
            {Name: "Charlie", Ratings: []teamsorter.Rating{7.5}},
        },
    }
    
    resp, err := teamsorter.SortTeams(req)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Meta.TeamCount != 2 {
        t.Errorf("expected 2 teams, got %d", resp.Meta.TeamCount)
    }
}
```

## Performance Considerations

- **Single call**: Typically < 10ms for 5-20 participants
- **Batch processing**: No built-in batching; call `SortTeams()` independently per request
- **Memory**: O(n * t) per call where n = participants, t = teams
- **Concurrency**: Safe to call from multiple goroutines (no global state)

## Version Stability

- **Current**: 1.0.0 (assumed; check README)
- **API Stability**: Committed to backward compatibility
- **Breaking Changes**: Unlikely; would increment major version
- **Deprecations**: Announced in advance

## Known Limitations

1. **Algorithm complexity**: Exponential; scales to ~20 participants practically
2. **Rating precision**: 1 decimal place (e.g., 9.5, not 9.55)
3. **No streaming**: All input must fit in memory
4. **No persistence**: Stateless function calls; no database
5. **No concurrency**: Each call is independent; internally single-threaded

## Future Extensions

Potential integration points:
- REST API wrapper
- gRPC service definition
- WebAssembly (wasm) compilation
- Database persistence layer
- Caching layer for repeated inputs

