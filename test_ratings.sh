#!/bin/bash
cd /Users/adilbek.es/workspace/team_sorter

echo "=== Test 1: With ratings (example.json) ==="
go run ./cmd/sorter -f example.json | jq '.meta, .teams[0]' | head -20

echo ""
echo "=== Test 2: Without ratings (example2.json) ==="
go run ./cmd/sorter -f example2.json | jq '.meta, .teams[0]' | head -20

echo ""
echo "=== Test 3: No ratings input, check output structure ==="
go run ./cmd/sorter -d '{"number_of_teams": 2, "participants": [{"name": "A"}, {"name": "B"}, {"name": "C"}]}' | jq '.' | head -40

