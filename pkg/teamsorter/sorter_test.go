package teamsorter

import (
	"encoding/json"
	"errors"
	"testing"
)

func oneRating(r float64) []Rating {
	return []Rating{Rating(r)}
}

func TestSortTeams_AddsMedianPlaceholderWhenNotDivisible(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
			{Name: "Dana", Ratings: oneRating(7.0)},
			{Name: "Nurlan", Ratings: oneRating(6.0)},
			{Name: "Aruzhan", Ratings: oneRating(5.0)},
			{Name: "Timur", Ratings: oneRating(4.0)},
			{Name: "Aida", Ratings: oneRating(3.0)},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	if got, want := resp.Meta.ParticipantCount, 8; got != want {
		t.Fatalf("participant_count = %d, want %d", got, want)
	}

	if got, want := resp.Meta.PlaceholderCount, 1; got != want {
		t.Fatalf("placeholder_count = %d, want %d", got, want)
	}

	placeholderCount := 0
	for _, team := range resp.Teams {
		if got, want := len(team.Members), 3; got != want {
			t.Fatalf("team size = %d, want %d", got, want)
		}
		for _, member := range team.Members {
			if len(member.Name) >= 11 && member.Name[:11] == "Placeholder" {
				placeholderCount++
				want := Rating(6.5)
				if len(member.Ratings) != 1 || member.Ratings[0] != want {
					t.Fatalf("placeholder rating = %v, want %.1f", member.Ratings, want)
				}
			}
		}
	}

	if got, want := placeholderCount, 1; got != want {
		t.Fatalf("placeholder count = %d, want %d", got, want)
	}
}

func TestSortTeams_DoesNotAddPlaceholderWhenDivisible(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
			{Name: "Dana", Ratings: oneRating(7.0)},
			{Name: "Nurlan", Ratings: oneRating(6.0)},
			{Name: "Aruzhan", Ratings: oneRating(5.0)},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	if got, want := resp.Meta.ParticipantCount, 6; got != want {
		t.Fatalf("participant_count = %d, want %d", got, want)
	}

	for _, team := range resp.Teams {
		if got, want := len(team.Members), 2; got != want {
			t.Fatalf("team size = %d, want %d", got, want)
		}
		for _, member := range team.Members {
			if len(member.Name) >= 11 && member.Name[:11] == "Placeholder" {
				t.Fatalf("unexpected placeholder member: %s", member.Name)
			}
		}
	}
}

func TestSortTeams_RejectsInconsistentRatingListLength(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: []Rating{10.0, 9.0, 8.0}},
			{Name: "Mira", Ratings: []Rating{9.0, 8.0}},
			{Name: "Bek", Ratings: []Rating{8.0, 7.0, 6.0}},
			{Name: "Dana", Ratings: []Rating{7.0, 6.0, 5.0}},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrInconsistentRatingCount) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrInconsistentRatingCount)
	}
}

func TestSortTeams_MetaRatingsAreListForMultiRatingInput(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Bek", Ratings: []Rating{10.0, 7.0, 5.0}},
			{Name: "Ali", Ratings: []Rating{9.0, 6.0, 5.0}},
			{Name: "Mira", Ratings: []Rating{8.0, 7.0, 6.0}},
			{Name: "Dana", Ratings: []Rating{7.0, 8.0, 5.0}},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	meta := payload["meta"].(map[string]any)
	for _, field := range []string{"min_team_rating", "max_team_rating", "rating_diff"} {
		values, ok := meta[field].([]any)
		if !ok {
			t.Fatalf("%s should be a list for multi-rating input", field)
		}
		if got, want := len(values), 4; got != want {
			t.Fatalf("%s length = %d, want %d", field, got, want)
		}
	}
}

func TestSortTeams_MetaRatingsStayScalarForSingleRatingInput(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Bek", Ratings: []Rating{10.0}},
			{Name: "Ali", Ratings: []Rating{9.0}},
			{Name: "Mira", Ratings: []Rating{8.0}},
			{Name: "Dana", Ratings: []Rating{7.0}},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	meta := payload["meta"].(map[string]any)
	for _, field := range []string{"min_team_rating", "max_team_rating", "rating_diff"} {
		if _, ok := meta[field].([]any); ok {
			t.Fatalf("%s should remain scalar for single-rating input", field)
		}
	}
}

func TestListOptimalNameSolutions_CountMatchesKnownOptimalCount(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(10.0)},
			{Name: "Bek", Ratings: oneRating(10.0)},
			{Name: "Dana", Ratings: oneRating(9.0)},
		},
	}

	solutions, err := ListOptimalNameSolutions(req)
	if err != nil {
		t.Fatalf("ListOptimalNameSolutions() error = %v", err)
	}

	// 12 raw solutions, but duplicates from placeholder swaps reduce to 6 unique solutions
	if got, want := len(solutions), 6; got != want {
		t.Fatalf("solution length = %d, want %d", got, want)
	}
}

func TestListOptimalNameSolutions_Format(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
			{Name: "Dana", Ratings: oneRating(7.0)},
		},
	}

	solutions, err := ListOptimalNameSolutions(req)
	if err != nil {
		t.Fatalf("ListOptimalNameSolutions() error = %v", err)
	}
	if len(solutions) == 0 {
		t.Fatalf("expected at least one solution")
	}

	first := solutions[0]
	if len(first) != 2 {
		t.Fatalf("solution team count = %d, want 2", len(first))
	}

	for teamName, members := range first {
		if teamName == "" {
			t.Fatalf("team name must not be empty")
		}
		if len(members) != 2 {
			t.Fatalf("members in %s = %d, want 2", teamName, len(members))
		}
	}
}

func TestSortTeams_MultiRatingBalancesRolesAndTotal(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "A", Ratings: []Rating{10.0, 1.0, 5.5}},
			{Name: "B", Ratings: []Rating{1.0, 10.0, 5.5}},
			{Name: "C", Ratings: []Rating{10.0, 1.0, 5.5}},
			{Name: "D", Ratings: []Rating{1.0, 10.0, 5.5}},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	if got, want := len(resp.Meta.RatingDiff.Values), 4; got != want {
		t.Fatalf("rating_diff length = %d, want %d", got, want)
	}

	for i, diff := range resp.Meta.RatingDiff.Values {
		if diff != 0 {
			t.Fatalf("rating_diff[%d] = %.1f, want 0.0", i, diff)
		}
	}
}

func TestSortTeams_RejectsTeamCountLessThanTwo(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 1,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(9.0)},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrInvalidTeamCount) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrInvalidTeamCount)
	}
}

func TestSortTeams_RejectsReservedPlaceholderNamePattern(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Placeholder 1", Ratings: oneRating(9.0)},
			{Name: "Dana", Ratings: oneRating(8.0)},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrReservedPlaceholderName) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrReservedPlaceholderName)
	}
}

func TestSortTeams_AllowsNamesThatAreNotPlaceholderIntPattern(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Placeholder X", Ratings: oneRating(9.0)},
			{Name: "Dana", Ratings: oneRating(8.0)},
			{Name: "Mira", Ratings: oneRating(7.0)},
		},
	}

	_, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() unexpected error = %v", err)
	}
}

func TestSortTeams_NoRatings_AssignsTeamsAndOmitsRatingFields(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali"},
			{Name: "Mira"},
			{Name: "Bek"},
			{Name: "Dana"},
			{Name: "Nurlan"},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	if resp.HasRatings {
		t.Fatalf("HasRatings = true, want false")
	}
	if got, want := resp.Meta.PlaceholderCount, 1; got != want {
		t.Fatalf("placeholder_count = %d, want %d", got, want)
	}
	if got, want := resp.Meta.MembersPerTeam, 3; got != want {
		t.Fatalf("members_per_team = %d, want %d", got, want)
	}
	if got, want := resp.Meta.SolutionCount, 1; got != want {
		t.Fatalf("solution_count = %d, want %d", got, want)
	}

	seenRealNames := make(map[string]struct{}, len(req.Participants))
	for _, team := range resp.Teams {
		if got, want := len(team.Members), 3; got != want {
			t.Fatalf("team size = %d, want %d", got, want)
		}
		for _, member := range team.Members {
			if len(member.Ratings) != 0 {
				t.Fatalf("member %q has unexpected ratings: %v", member.Name, member.Ratings)
			}
			if member.IsPlaceholder {
				continue
			}
			if _, exists := seenRealNames[member.Name]; exists {
				t.Fatalf("real participant %q appears more than once", member.Name)
			}
			seenRealNames[member.Name] = struct{}{}
		}
	}
	if got, want := len(seenRealNames), len(req.Participants); got != want {
		t.Fatalf("real participant assignments = %d, want %d", got, want)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	teamsAny := payload["teams"].([]any)
	for _, item := range teamsAny {
		team := item.(map[string]any)
		if _, ok := team["total_rating"]; ok {
			t.Fatalf("total_rating must be omitted when input has no ratings")
		}
		members := team["members"].([]any)
		for _, m := range members {
			member := m.(map[string]any)
			if _, ok := member["rating"]; ok {
				t.Fatalf("member rating must be omitted when input has no ratings")
			}
		}
	}

	meta := payload["meta"].(map[string]any)
	if _, ok := meta["min_team_rating"]; ok {
		t.Fatalf("min_team_rating must be omitted when input has no ratings")
	}
	if _, ok := meta["max_team_rating"]; ok {
		t.Fatalf("max_team_rating must be omitted when input has no ratings")
	}
	if _, ok := meta["rating_diff"]; ok {
		t.Fatalf("rating_diff must be omitted when input has no ratings")
	}
}

func TestListOptimalNameSolutions_NoRatings_ReturnsAllPermutations(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali"},
			{Name: "Mira"},
			{Name: "Bek"},
			{Name: "Dana"},
		},
	}

	solutions, err := ListOptimalNameSolutions(req)
	if err != nil {
		t.Fatalf("ListOptimalNameSolutions() error = %v", err)
	}

	if got, want := len(solutions), 6; got != want {
		t.Fatalf("solution length = %d, want %d", got, want)
	}
}

