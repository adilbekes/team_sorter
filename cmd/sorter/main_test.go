package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain_RejectsUsingDataAndFileFlagsTogether(t *testing.T) {
	_, stderr, exitCode := runSorterMainProcess(t, "", "-d", "{}", "-f", "input.json")

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &payload); err != nil {
		t.Fatalf("stderr is not valid JSON: %v; stderr=%q", err, stderr)
	}

	if got, want := payload["error"], "cannot use both -d and -f flags"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestMain_OutputsSortTeamsResponseForValidDataFlag(t *testing.T) {
	reqJSON := `{"number_of_teams":2,"participants":[{"name":"Ali","rating":10},{"name":"Mira","rating":9},{"name":"Bek","rating":8},{"name":"Dana","rating":7}]}`

	stdout, stderr, exitCode := runSorterMainProcess(t, "", "-d", reqJSON)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; stdout=%q", err, stdout)
	}

	if _, ok := payload["teams"]; !ok {
		t.Fatalf("expected teams in output JSON")
	}
	if _, ok := payload["meta"]; !ok {
		t.Fatalf("expected meta in output JSON")
	}
}

func TestMain_ListFlagOutputsOptimalSolutionsArray(t *testing.T) {
	reqJSON := `{"number_of_teams":2,"participants":[{"name":"Ali","rating":10},{"name":"Mira","rating":9},{"name":"Bek","rating":8},{"name":"Dana","rating":7}]}`

	stdout, stderr, exitCode := runSorterMainProcess(t, "", "-d", reqJSON, "-list")

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var payload []map[string][]string
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not valid list JSON: %v; stdout=%q", err, stdout)
	}
	if len(payload) == 0 {
		t.Fatalf("expected at least one optimal solution")
	}
}

func TestMain_UsesStdinWhenNoDataOrFileFlagProvided(t *testing.T) {
	reqJSON := `{"number_of_teams":2,"participants":[{"name":"Ali","rating":10},{"name":"Mira","rating":9},{"name":"Bek","rating":8},{"name":"Dana","rating":7}]}`

	stdout, stderr, exitCode := runSorterMainProcess(t, reqJSON)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not valid JSON: %v; stdout=%q", err, stdout)
	}
	if _, ok := payload["teams"]; !ok {
		t.Fatalf("expected teams in output JSON")
	}
}

func TestMain_WritesErrorToStderrForInvalidJSON(t *testing.T) {
	_, stderr, exitCode := runSorterMainProcess(t, "", "-d", "{not-json}")

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &payload); err != nil {
		t.Fatalf("stderr is not valid JSON: %v; stderr=%q", err, stderr)
	}

	if got := payload["error"]; !strings.HasPrefix(got, "invalid input JSON:") {
		t.Fatalf("error = %q, want prefix %q", got, "invalid input JSON:")
	}
}

func TestMain_WritesResultToOutputFile(t *testing.T) {
	reqJSON := `{"number_of_teams":2,"participants":[{"name":"Ali","rating":10},{"name":"Mira","rating":9},{"name":"Bek","rating":8},{"name":"Dana","rating":7}]}`
	outputFile := filepath.Join(t.TempDir(), "out.json")

	stdout, stderr, exitCode := runSorterMainProcess(t, "", "-d", reqJSON, "-o", outputFile)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("stdout = %q, want empty when -o is set", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("output file is not valid JSON: %v; content=%q", err, string(data))
	}
	if _, ok := payload["teams"]; !ok {
		t.Fatalf("expected teams in output file JSON")
	}
}

func TestMain_WritesErrorToOutputFileWhenInputIsInvalid(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "error.json")

	_, stderr, exitCode := runSorterMainProcess(t, "", "-d", "{bad}", "-o", outputFile)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1", exitCode)
	}

	var stderrPayload map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &stderrPayload); err != nil {
		t.Fatalf("stderr is not valid JSON: %v; stderr=%q", err, stderr)
	}
	if got := stderrPayload["error"]; !strings.HasPrefix(got, "invalid input JSON:") {
		t.Fatalf("stderr error = %q, want prefix %q", got, "invalid input JSON:")
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var filePayload map[string]string
	if err := json.Unmarshal(data, &filePayload); err != nil {
		t.Fatalf("output file is not valid error JSON: %v; content=%q", err, string(data))
	}
	if got := filePayload["error"]; !strings.HasPrefix(got, "invalid input JSON:") {
		t.Fatalf("output file error = %q, want prefix %q", got, "invalid input JSON:")
	}
}

func TestSorterMainProcess(t *testing.T) {
	if os.Getenv("GO_WANT_SORTER_MAIN_PROCESS") != "1" {
		return
	}

	sep := -1
	for i, arg := range os.Args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 {
		os.Exit(2)
	}

	os.Args = append([]string{"sorter"}, os.Args[sep+1:]...)
	main()
	os.Exit(0)
}

func runSorterMainProcess(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestSorterMainProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_SORTER_MAIN_PROCESS=1")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("failed to run helper process: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}

	return stdout.String(), stderr.String(), exitCode
}

