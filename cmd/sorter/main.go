// sorter is a language-agnostic CLI binary for balancing participants into teams.
//
// Usage:
//
//	echo '<json>' | sorter                           # stdin
//	sorter -d '<json>'                               # JSON string flag
//	sorter -f request.json                           # JSON file flag
//	sorter -f request.json -o result.json            # file input and output
//
// Input  – JSON object on stdin/flag (see SortTeamsRequest)
// Output – JSON object on stdout (see SortTeamsResponse), or {"error":"..."} on failure
// Exit   – 0 on success, 1 on error
//
// Example input:
//
//	{
//	  "number_of_teams": 2,
//	  "participants": [
//	    {"name": "Ali", "rating": 10},
//	    {"name": "Mira", "rating": 9},
//	    {"name": "Bek", "rating": 8},
//	    {"name": "Dana", "rating": 7}
//	  ]
//	}
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"team_sorter/pkg/teamsorter"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeErrorToStderr(msg string) {
	_ = json.NewEncoder(os.Stderr).Encode(errorResponse{Error: msg})
}

func writeErrorToFile(filename string, msg string) (err error) {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	err = json.NewEncoder(file).Encode(errorResponse{Error: msg})
	return err
}

func reportError(msg string, outputFile string) {
	writeErrorToStderr(msg)
	if outputFile != "" {
		_ = writeErrorToFile(outputFile, msg)
	}
}

func main() {
	dataFlag := flag.String("d", "", "JSON request string")
	fileFlag := flag.String("f", "", "JSON request file")
	outputFlag := flag.String("o", "", "JSON output file (if not set, output to stdout)")
	listFlag := flag.Bool("list", false, "output all optimal solutions as [{\"Team N\": [\"Name\", ...]}, ...]")
	flag.Parse()

	var inputData io.Reader

	// Determine input source: -d flag > -f flag > stdin
	if *dataFlag != "" && *fileFlag != "" {
		reportError("cannot use both -d and -f flags", *outputFlag)
		os.Exit(1)
	}

	if *dataFlag != "" {
		inputData = strings.NewReader(*dataFlag)
	} else if *fileFlag != "" {
		file, err := os.Open(*fileFlag)
		if err != nil {
			reportError(fmt.Sprintf("failed to open file: %s", err), *outputFlag)
			os.Exit(1)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				writeErrorToStderr(fmt.Sprintf("failed to close input file: %s", closeErr))
			}
		}()
		inputData = file
	} else {
		inputData = os.Stdin
	}

	var req teamsorter.SortTeamsRequest
	if err := json.NewDecoder(inputData).Decode(&req); err != nil {
		reportError(fmt.Sprintf("invalid input JSON: %s", err), *outputFlag)
		os.Exit(1)
	}

	var payload interface{}
	if *listFlag {
		solutions, err := teamsorter.ListOptimalNameSolutions(req)
		if err != nil {
			reportError(err.Error(), *outputFlag)
			os.Exit(1)
		}
		payload = solutions
	} else {
		result, err := teamsorter.SortTeams(req)
		if err != nil {
			reportError(err.Error(), *outputFlag)
			os.Exit(1)
		}
		payload = result
	}

	// Write output
	if *outputFlag != "" {
		file, err := os.Create(*outputFlag)
		if err != nil {
			reportError(fmt.Sprintf("failed to create output file: %s", err), *outputFlag)
			os.Exit(1)
		}

		if err := json.NewEncoder(file).Encode(payload); err != nil {
			_ = file.Close()
			reportError(fmt.Sprintf("failed to encode result: %s", err), *outputFlag)
			os.Exit(1)
		}

		if err := file.Close(); err != nil {
			reportError(fmt.Sprintf("failed to close output file: %s", err), *outputFlag)
			os.Exit(1)
		}
	} else {
		if err := json.NewEncoder(os.Stdout).Encode(payload); err != nil {
			writeErrorToStderr(fmt.Sprintf("failed to encode result: %s", err))
			os.Exit(1)
		}
	}
}
