package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestOneRating_ReturnsSingleValueSlice(t *testing.T) {
	got := oneRating(7.4)

	if got == nil {
		t.Fatalf("oneRating() returned nil slice")
	}
	if len(got) != 1 {
		t.Fatalf("oneRating() length = %d, want 1", len(got))
	}
	if got[0] != 7.4 {
		t.Fatalf("oneRating()[0] = %.1f, want 7.4", got[0])
	}
}

func TestMain_PrintsParticipantsTeamsAndMetaSections(t *testing.T) {
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w

	main()

	if err := w.Close(); err != nil {
		t.Fatalf("close stdout writer error = %v", err)
	}
	os.Stdout = originalStdout

	outputBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close stdout reader error = %v", err)
	}

	output := string(outputBytes)

	if !strings.Contains(output, "Participants:") {
		t.Fatalf("output should contain Participants section")
	}
	if !strings.Contains(output, "Number of teams:") {
		t.Fatalf("output should contain number of teams line")
	}
	if !strings.Contains(output, "Team 1 | Total:") {
		t.Fatalf("output should contain team lines")
	}
	if !strings.Contains(output, "Meta:") {
		t.Fatalf("output should contain Meta section")
	}
	if !strings.Contains(output, "Optimal solutions:") {
		t.Fatalf("output should contain optimal solutions line")
	}
}

