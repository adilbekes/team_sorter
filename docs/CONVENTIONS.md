# Coding Conventions for `team_sorter`

This document outlines project-specific coding standards and patterns to maintain consistency across the codebase.

## Go Version & Dependencies

- **Go Version**: 1.25.5 or later
- **Module**: `team_sorter`
- **External Dependencies**: None (standard library only)
- **Test Framework**: Go's built-in `testing` package

## Code Organization

### Package Structure

- **`pkg/teamsorter/`**: Public library package
  - Exports all public functions and types needed for library usage
  - No dependencies on CLI code
  - All functions and types start with uppercase (exported)

- **`cmd/sorter/`**: Standalone CLI binary
  - Imports `pkg/teamsorter` only
  - Contains `main()` and argument parsing
  - Does not import from other `cmd/` packages

- **`cmd/demo/`**: Example/demonstration binary
  - Educational usage only
  - Can import both `pkg/teamsorter` and showcase patterns

### File Naming

- **Source files**: Lowercase with underscores (e.g., `validator.go`, `sorter_test.go`)
- **Binary sources**: Place in `cmd/<name>/main.go` or `cmd/<name>/main_test.go`
- **Test files**: Suffix with `_test.go` (e.g., `sorter_test.go`)

## Naming Conventions

### Types

```go
// Public types: PascalCase
type Participant struct { ... }
type Rating float64
type Team struct { ... }

// Private types: camelCase
type errorResponse struct { ... }
type optimizationObjective struct { ... }
```

### Functions

```go
// Public functions: PascalCase
func SortTeams(req SortTeamsRequest) (*SortTeamsResponse, error) { ... }
func ValidateSortTeamsRequest(req SortTeamsRequest) error { ... }

// Private functions: camelCase
func fillTeamsOptimally(...) (bool, int, [][]Team) { ... }
func randomSeed() int64 { ... }
func normalizeRating(v float64) Rating { ... }
```

### Variables & Constants

```go
// Constants: PascalCase for exported, camelCase for private
const eps = 1e-9  // Optimization epsilon for float comparison

// Error variables: ErrXxx naming pattern
var (
    ErrInvalidTeamCount = errors.New("...")
    ErrNoSolution = errors.New("...")
)

// Local variables: camelCase
var participants []Participant
var bestObjective optimizationObjective
```

### struct Tags

- **JSON**: Use snake_case with `,omitempty` where appropriate
  ```go
  type Team struct {
      Name        string        `json:"name"`
      TotalRating Rating        `json:"total_rating"`
      Members     []Participant `json:"members"`
  }
  ```

- **Omit unexported fields**: Use `json:"-"` for internal-only fields
  ```go
  type Participant struct {
      Ratings   []Rating `json:"-"`  // Not serialized
  }
  ```

## Code Style & Patterns

### Error Handling

1. **Error Definitions**: Pre-define in `errors.go`, use `errors.New()`
   ```go
   var ErrInvalidTeamCount = errors.New("number_of_teams must be at least 2")
   ```

2. **Error Returns**: Return early on validation errors
   ```go
   if err := ValidateSortTeamsRequest(req); err != nil {
       return nil, err
   }
   ```

3. **Error Messages**: Lowercase, descriptive, actionable
   ```go
   "participant rating must be between 1.0 and 10.0"
   ```

### JSON Marshaling

- **Custom Marshal/Unmarshal**: Use receiver methods on types that need special JSON handling
  ```go
  func (r *Rating) MarshalJSON() ([]byte, error) { ... }
  func (p *Participant) UnmarshalJSON(data []byte) error { ... }
  ```

- **Adapter Structs**: For conditional output, use nested types
  ```go
  type participantJSON struct {
      Name   string      `json:"name"`
      Rating interface{} `json:"rating,omitempty"`
  }
  ```

### String Formatting

- **Printf-style**: Use for debug/logging strings
  ```go
  fmt.Sprintf("Team %d", i+1)
  fmt.Printf("%.1f", rating)
  ```

- **Stringer Interface**: Implement `String()` for custom types
  ```go
  func (p *Participant) String() string {
      return fmt.Sprintf("%s(%.1f)", p.Name, p.Ratings[0])
  }
  ```

### Floating-Point Handling

1. **Normalization**: Always normalize ratings to 1 decimal place
   ```go
   func normalizeRating(v float64) Rating {
       return Rating(math.Round(v*10) / 10)
   }
   ```

2. **Comparison**: Use epsilon-based equality for floats
   ```go
   const eps = 1e-9
   if math.Abs(float64(a - b)) > eps {
       // Not equal
   }
   ```

3. **JSON Output**: Format as strings with 1 decimal place
   ```go
   return []byte(fmt.Sprintf("%.1f", *r))
   ```

### Slices and Arrays

- **Pre-allocate when possible** (for performance)
  ```go
  teams := make([]Team, teamCount)
  capacities := make([]int, teamCount)
  ```

- **Use `append()` carefully**: Know the capacity to avoid reallocation
  ```go
  members := make([]Participant, 0, capacity)
  members = append(members, participant)
  ```

- **Immutability**: Copy slices when modifying to avoid side effects
  ```go
  participants := make([]Participant, len(req.Participants))
  copy(participants, req.Participants)
  ```

### Recursion & Backtracking

- **Depth-First Search**: Use named inner functions with closure over state
  ```go
  var search func(index int)
  search = func(index int) {
      if index == len(participants) {
          // Base case
          return
      }
      // Recursive case
      search(index + 1)
  }
  search(0)
  ```

- **State Management**: Track state in outer scope; restore after recursion
  ```go
  teams[i].Members = append(teams[i].Members, participant)
  search(index + 1)
  teams[i].Members = teams[i].Members[:len(teams[i].Members)-1]  // Restore
  ```

### Random Number Generation

- **Per-Call Seeding**: Use cryptographically secure seed for each call
  ```go
  func randomSeed() int64 {
      var b [8]byte
      if _, err := crand.Read(b[:]); err == nil {
          return int64(binary.LittleEndian.Uint64(b[:]))
      }
      return time.Now().UnixNano()
  }
  
  rng := rand.New(rand.NewSource(randomSeed()))
  ```

- **Avoid Global RNG**: Never use `rand.Int()` directly; always use local `*rand.Rand`

### Case-Insensitive String Comparison

- **Normalize & Compare**: Use `strings.ToLower()` and `strings.TrimSpace()`
  ```go
  key := strings.ToLower(strings.TrimSpace(name))
  if _, exists := seenNames[key]; exists {
      return ErrDuplicateParticipantName
  }
  ```

## Documentation & Comments

### Function Documentation

- **Public functions**: Start with function name, describe behavior and return values
  ```go
  // SortTeams divides participants into balanced teams based on ratings,
  // achieving optimal minimum rating difference across all teams.
  func SortTeams(req SortTeamsRequest) (*SortTeamsResponse, error) {
  ```

- **Private functions**: Optional but recommended for complex logic
  ```go
  // fillTeamsOptimally explores all valid assignments using backtracking
  // to find the globally optimal solution minimizing rating variance.
  func fillTeamsOptimally(...) (bool, int, [][]Team) {
  ```

### Inline Comments

- **Explain "why" not "what"**: Code should be clear; comments explain decisions
  ```go
  // Use per-call RNG with crypto seed to avoid repeated outputs
  // across fast process starts.
  rng := rand.New(rand.NewSource(randomSeed()))
  
  // Only prune when BOTH teams are completely empty.
  // Once any member is assigned, teams differ and must not be skipped.
  if len(teams[j].Members) == 0 && len(teams[i].Members) == 0 {
      symmetric = true
  }
  ```

- **Algorithm notes**: Explain non-obvious logic
  ```go
  // Single-rating case has [criterion,avg] where criterion == avg;
  // optimize by avg only to avoid redundant comparisons.
  if metricCount == 2 {
      return []Rating{diffs[1]}
  }
  ```

## Testing Conventions

### Test File Structure

```go
package teamsorter

import (
    "testing"
)

func TestSortTeams(t *testing.T) {
    // Arrange
    req := SortTeamsRequest{
        NumberOfTeams: 2,
        Participants: []Participant{...},
    }
    
    // Act
    resp, err := SortTeams(req)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Meta.TeamCount != 2 {
        t.Errorf("expected 2 teams, got %d", resp.Meta.TeamCount)
    }
}
```

### Test Naming

- **Test functions**: `TestXxx` where `Xxx` is the function/behavior being tested
  ```go
  func TestValidateSortTeamsRequest(t *testing.T) { ... }
  func TestSortTeamsWithRatings(t *testing.T) { ... }
  func TestSortTeamsUnrated(t *testing.T) { ... }
  ```

### Assertion Patterns

- **Use `t.Fatal` for setup errors** (test cannot continue)
  ```go
  resp, err := SortTeams(req)
  if err != nil {
      t.Fatalf("unexpected error: %v", err)
  }
  ```

- **Use `t.Error` for assertion failures** (test continues)
  ```go
  if resp.Meta.TeamCount != expected {
      t.Errorf("expected %d teams, got %d", expected, resp.Meta.TeamCount)
  }
  ```

## CLI Conventions

### Command-Line Arguments

- **Flags**: Use `flag` package with descriptive names
  ```go
  dataFlag := flag.String("d", "", "JSON request string")
  fileFlag := flag.String("f", "", "JSON request file")
  outputFlag := flag.String("o", "", "JSON output file")
  listFlag := flag.Bool("list", false, "output all optimal solutions")
  ```

- **Priority**: `-d` > `-f` > stdin
- **Mutual exclusivity**: Validate conflicting flags
  ```go
  if *dataFlag != "" && *fileFlag != "" {
      reportError("cannot use both -d and -f flags", *outputFlag)
      os.Exit(1)
  }
  ```

### Exit Codes

- **0**: Success
- **1**: Any error (input validation, I/O, algorithm failure)

### Error Output

- **To stderr**: Use `json.NewEncoder(os.Stderr)` for JSON errors
- **To file**: Use `-o` flag; write `{"error": "message"}` JSON
- **Format**: Always `{"error": "descriptive message"}`

## Import Organization

1. **Standard library**: Alphabetical
2. **Blank line**
3. **Local packages**: `team_sorter/...`

```go
import (
    "encoding/json"
    "fmt"
    "math"
    "sort"
    
    "team_sorter/pkg/teamsorter"
)
```

## Performance Considerations

1. **Avoid unnecessary allocations**: Pre-allocate slices when size is known
2. **Use value types for small types**: `Rating` is `float64`; pass by value
3. **Use pointers for large types**: `Team`, `Participant` use pointers in slices
4. **Clone when needed**: Deep copy teams/participants to avoid aliasing bugs
5. **Index comparisons over map lookups**: Use integer indices when iterating

## Backward Compatibility

- **No breaking changes** to public API functions or types
- **Adding new fields**: Use `omitempty` in JSON tags
- **Modifying error messages**: Test code relying on exact messages
- **Algorithm changes**: Ensure solution quality metrics don't degrade

