# Workflows Guide for `team_sorter`

This document outlines developer workflows for building, running, and testing the team_sorter project.

## Prerequisites

- **Go**: Version 1.25.5 or later
- **Shell**: bash or zsh
- **Make** (optional): For convenience shortcuts
- **jq** (optional): For pretty-printing JSON output

## Building

### Build the CLI Binary

```bash
go build -o bin/sorter ./cmd/sorter/
```

Output: `./bin/sorter` executable

### Build the Demo Binary

```bash
go build -o bin/demo ./cmd/demo/
```

Output: `./bin/demo` executable

### Build Both Binaries

```bash
go build -o bin/sorter ./cmd/sorter/
go build -o bin/demo ./cmd/demo/
```

### Clean Binaries

```bash
rm -rf bin/
```

## Running the CLI

### Via stdin

```bash
echo '{"number_of_teams": 2, "participants": [{"name": "Alice", "rating": 9.5}, {"name": "Bob", "rating": 8.0}]}' | ./bin/sorter
```

### Via JSON string flag (`-d`)

```bash
./bin/sorter -d '{"number_of_teams": 2, "participants": [{"name": "Alice", "rating": 9.5}]}'
```

### Via JSON file (`-f`)

```bash
./bin/sorter -f input.json
```

### With output file (`-o`)

```bash
./bin/sorter -f input.json -o output.json
```

### List all optimal solutions (`-list`)

```bash
./bin/sorter -f input.json -list | jq .
```

### Example Workflow

```bash
# 1. Create input file
cat > input.json << 'EOF'
{
  "number_of_teams": 3,
  "participants": [
    {"name": "Ali", "rating": 10.0},
    {"name": "Mira", "rating": 10.0},
    {"name": "Bek", "rating": 10.0},
    {"name": "Dana", "rating": 9.0}
  ]
}
EOF

# 2. Build binary
go build -o bin/sorter ./cmd/sorter/

# 3. Run sorter
./bin/sorter -f input.json -o output.json

# 4. View output
jq . output.json
```

## Testing

### Run All Tests

```bash
go test ./...
```

Output includes:
- Unit test results (✓ or ✗)
- Coverage summary
- Any failures with stack traces

### Run Tests for Specific Package

```bash
go test ./pkg/teamsorter/
```

### Run Tests for Specific File

```bash
go test -run TestSortTeams ./pkg/teamsorter/
```

### Verbose Test Output

```bash
go test -v ./...
```

Displays each test name and result as it runs.

### Test with Coverage Report

```bash
go test -cover ./...
```

Shows coverage percentage for each package.

### Generate Coverage Profile

```bash
go test -coverprofile=coverage.out ./...
```

Then view in HTML:

```bash
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
```

### Run Tests in Parallel

```bash
go test -parallel 4 ./...
```

Default is number of CPU cores.

### Run Tests with Timeout

```bash
go test -timeout 30s ./...
```

### Test-Driven Development Pattern

```bash
# 1. Write test in XXX_test.go
# 2. Run test (will fail)
go test -run TestNewFeature -v ./pkg/teamsorter/

# 3. Implement function in XXX.go
# 4. Run test again
go test -run TestNewFeature -v ./pkg/teamsorter/

# 5. Run all tests to ensure no regressions
go test ./...
```

## Code Quality

### Format Code

```bash
go fmt ./...
```

Auto-formats all Go files to standard style.

### Static Analysis

```bash
go vet ./...
```

Checks for common programming errors:
- Unused variables
- Unreachable code
- Type mismatches
- Suspicious constructs

### Lint with golangci-lint (optional)

```bash
# Install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run
golangci-lint run ./...
```

### Code Review Checklist

Before committing:

```bash
# 1. Format
go fmt ./...

# 2. Lint/vet
go vet ./...

# 3. Test
go test -v ./...

# 4. Coverage (should be > 80%)
go test -cover ./...

# 5. Build
go build -o bin/sorter ./cmd/sorter/
go build -o bin/demo ./cmd/demo/
```

## Managing Dependencies

### View Dependencies

```bash
go list -m all
```

Shows module versions used in project.

### Update Go Version in go.mod

```bash
go get -u all
```

Updates all dependencies to latest versions.

### Download Dependencies

```bash
go mod download
```

Pre-downloads all dependencies (useful for CI/CD).

### Vendor Dependencies (optional)

```bash
go mod vendor
```

Copies dependencies into `vendor/` directory for offline builds.

## Debugging

### Print Debug Info

Use `fmt.Printf()` for temporary debugging:

```go
fmt.Printf("DEBUG: teams = %+v\n", teams)
```

Remove before committing.

### Run with Go Debugger (dlv)

```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Run binary with debugger
dlv debug ./cmd/sorter/

# Set breakpoint
(dlv) break main.main

# Continue to breakpoint
(dlv) continue

# Print variable
(dlv) print dataFlag

# Exit
(dlv) exit
```

### Trace Execution

```bash
# With -race flag to detect race conditions
go test -race ./...
```

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...

# Analyze CPU profile
go tool pprof cpu.prof
(pprof) top
(pprof) list functionName
```

## Demo Binary Workflow

The demo binary showcases library usage:

```bash
go run ./cmd/demo/
```

Use as a reference for:
- Creating `SortTeamsRequest` structs
- Handling `SortTeamsResponse`
- Error handling patterns
- Multi-rating inputs

## CI/CD Workflow

Typical GitHub Actions / CI pipeline:

```bash
#!/bin/bash
set -e

# Build
go build -o bin/sorter ./cmd/sorter/
go build -o bin/demo ./cmd/demo/

# Format check
go fmt ./...

# Vet
go vet ./...

# Test with coverage
go test -v -coverprofile=coverage.out ./...

# Optional: enforce minimum coverage
# if [ $(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//') -lt 80 ]; then
#   echo "Coverage below 80%"
#   exit 1
# fi
```

## Quick Commands Cheat Sheet

| Task | Command |
|------|---------|
| Build | `go build -o bin/sorter ./cmd/sorter/` |
| Run | `./bin/sorter -f input.json \| jq .` |
| Test | `go test ./...` |
| Test verbose | `go test -v ./...` |
| Coverage | `go test -cover ./...` |
| Format | `go fmt ./...` |
| Lint | `go vet ./...` |
| Demo | `go run ./cmd/demo/` |
| Clean | `rm -rf bin/ coverage.out` |

## Common Workflows

### Adding a New Feature

1. **Create test file** (if not exists): `pkg/teamsorter/feature_test.go`
2. **Write failing test** (TDD approach)
3. **Run test**: `go test -run TestFeature -v ./pkg/teamsorter/`
4. **Implement function** in `pkg/teamsorter/feature.go`
5. **Run test again**: `go test -run TestFeature -v ./pkg/teamsorter/`
6. **Run all tests**: `go test ./...`
7. **Format & vet**: `go fmt ./... && go vet ./...`
8. **Build binaries**: `go build -o bin/sorter ./cmd/sorter/`

### Fixing a Bug

1. **Write failing test** that reproduces the bug
2. **Run test to confirm failure**
3. **Fix code** in the relevant module
4. **Run test to confirm fix**
5. **Run full test suite** to check for regressions
6. **Format & vet**
7. **Manual testing** if applicable

### Performance Investigation

1. **Create benchmark** in `*_test.go`:
   ```go
   func BenchmarkSortTeams(b *testing.B) {
       for i := 0; i < b.N; i++ {
           SortTeams(req)
       }
   }
   ```

2. **Run benchmark**: `go test -bench=. ./pkg/teamsorter/`
3. **Profile**: `go test -cpuprofile=cpu.prof -bench=. ./pkg/teamsorter/`
4. **Analyze**: `go tool pprof cpu.prof`

### Integration Testing with Examples

Use example files in the repository:

```bash
# Run with example1.json
./bin/sorter -f example1.json | jq .

# Run with example2.json
./bin/sorter -f example2.json -list | jq .

# Compare outputs
./bin/sorter -f example1.json -o output1.json
./bin/sorter -f example2.json -o output2.json
diff output1.json output2.json
```

## Makefile (Optional)

For convenience, create a `Makefile`:

```makefile
.PHONY: build test clean fmt vet run

build:
	go build -o bin/sorter ./cmd/sorter/
	go build -o bin/demo ./cmd/demo/

test:
	go test -v ./...

test-coverage:
	go test -cover ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin/ coverage.out

run: build
	./bin/sorter -f input.json | jq .

all: clean fmt vet test build
```

Then use:

```bash
make build   # Build binaries
make test    # Run tests
make clean   # Remove artifacts
make all     # Format, vet, test, build
```

## Troubleshooting

### "command not found: go"
- **Fix**: Install Go from https://golang.org/

### Test failures with "import not found"
- **Fix**: Run `go mod download`

### Binary not found after build
- **Fix**: Check output path; default is `./bin/sorter`

### Tests timeout
- **Fix**: Increase timeout with `-timeout 60s`

### Race condition detected
- **Fix**: Run `go test -race ./...` to identify, then fix synchronization

### JSON parsing errors in CLI
- **Fix**: Validate JSON with `jq . input.json` before passing to sorter

## Environment Variables

| Variable | Purpose | Example |
|----------|---------|---------|
| `GOFLAGS` | Compilation flags | `GOFLAGS="-v"` |
| `GOCACHE` | Cache directory | `GOCACHE="~/.cache/go-build"` |

## Performance Tips

1. **Use `-ldflags` to strip debug info** (reduces binary size):
   ```bash
   go build -ldflags="-s -w" -o bin/sorter ./cmd/sorter/
   ```

2. **Disable cgo** for pure Go compilation:
   ```bash
   CGO_ENABLED=0 go build -o bin/sorter ./cmd/sorter/
   ```

3. **Cross-compile** for different OS:
   ```bash
   GOOS=linux GOARCH=amd64 go build -o bin/sorter-linux ./cmd/sorter/
   GOOS=darwin GOARCH=amd64 go build -o bin/sorter-macos ./cmd/sorter/
   ```

