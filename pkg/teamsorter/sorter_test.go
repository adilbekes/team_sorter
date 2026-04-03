package teamsorter

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
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

func TestSortTeams_RejectsTooManyTeams(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 4,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrTooManyTeams) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrTooManyTeams)
	}
}

func TestSortTeams_RejectsMixedRatingPresence(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira"},
			{Name: "Bek", Ratings: oneRating(8.0)},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrInconsistentRatings) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrInconsistentRatings)
	}
}

func TestSortTeams_RejectsDuplicateNamesCaseInsensitive(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "ali", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrDuplicateParticipantName) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrDuplicateParticipantName)
	}
}

func TestSortTeams_RejectsEmptyParticipantNameAfterTrim(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 2,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "   ", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
		},
	}

	_, err := SortTeams(req)
	if !errors.Is(err, ErrEmptyParticipantName) {
		t.Fatalf("SortTeams() error = %v, want %v", err, ErrEmptyParticipantName)
	}
}

func TestSortTeams_RejectsInvalidParticipantRatingValues(t *testing.T) {
	tests := []struct {
		name   string
		rating Rating
	}{
		{name: "below minimum", rating: Rating(0.9)},
		{name: "above maximum", rating: Rating(10.1)},
		{name: "nan", rating: Rating(math.NaN())},
		{name: "positive infinity", rating: Rating(math.Inf(1))},
		{name: "negative infinity", rating: Rating(math.Inf(-1))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := SortTeamsRequest{
				NumberOfTeams: 2,
				Participants: []Participant{
					{Name: "Ali", Ratings: []Rating{10.0}},
					{Name: "Mira", Ratings: []Rating{tc.rating}},
					{Name: "Bek", Ratings: []Rating{8.0}},
				},
			}

			_, err := SortTeams(req)
			if !errors.Is(err, ErrInvalidParticipantRating) {
				t.Fatalf("SortTeams() error = %v, want %v", err, ErrInvalidParticipantRating)
			}
		})
	}
}

func TestSortTeams_AddsMultiRatingPlaceholderUsingPerDimensionMedian(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "A", Ratings: []Rating{10.0, 2.0}},
			{Name: "B", Ratings: []Rating{8.0, 4.0}},
			{Name: "C", Ratings: []Rating{6.0, 6.0}},
			{Name: "D", Ratings: []Rating{4.0, 8.0}},
			{Name: "E", Ratings: []Rating{2.0, 10.0}},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	if got, want := resp.Meta.PlaceholderCount, 1; got != want {
		t.Fatalf("placeholder_count = %d, want %d", got, want)
	}

	placeholderFound := false
	for _, team := range resp.Teams {
		if got, want := len(team.Members), 2; got != want {
			t.Fatalf("team size = %d, want %d", got, want)
		}
		for _, member := range team.Members {
			if !member.IsPlaceholder {
				continue
			}
			placeholderFound = true
			if got, want := len(member.Ratings), 2; got != want {
				t.Fatalf("placeholder rating length = %d, want %d", got, want)
			}
			if got, want := member.Ratings[0], Rating(6.0); got != want {
				t.Fatalf("placeholder rating[0] = %.1f, want %.1f", got, want)
			}
			if got, want := member.Ratings[1], Rating(6.0); got != want {
				t.Fatalf("placeholder rating[1] = %.1f, want %.1f", got, want)
			}
		}
	}

	if !placeholderFound {
		t.Fatalf("expected exactly one placeholder in output")
	}
}

func TestSortTeams_SolutionCountMatchesListOptimalNameSolutionsForRatedInput(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(10.0)},
			{Name: "Bek", Ratings: oneRating(10.0)},
			{Name: "Dana", Ratings: oneRating(9.0)},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	solutions, err := ListOptimalNameSolutions(req)
	if err != nil {
		t.Fatalf("ListOptimalNameSolutions() error = %v", err)
	}

	if got, want := resp.Meta.SolutionCount, len(solutions); got != want {
		t.Fatalf("solution_count = %d, want %d", got, want)
	}
}

func TestSortTeams_SelectedSolutionIsOneOfOptimalNameSolutions(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(10.0)},
			{Name: "Bek", Ratings: oneRating(10.0)},
			{Name: "Dana", Ratings: oneRating(9.0)},
		},
	}

	resp, err := SortTeams(req)
	if err != nil {
		t.Fatalf("SortTeams() error = %v", err)
	}

	solutions, err := ListOptimalNameSolutions(req)
	if err != nil {
		t.Fatalf("ListOptimalNameSolutions() error = %v", err)
	}

	selected := canonicalSolutionFromTeams(resp.Teams)
	for _, solution := range solutions {
		if selected == canonicalSolutionFromNameMap(solution) {
			return
		}
	}

	t.Fatalf("selected SortTeams solution was not found in optimal name-only solutions")
}

func canonicalSolutionFromTeams(teams []Team) string {
	parts := make([]string, 0, len(teams))
	for _, team := range teams {
		names := make([]string, 0, len(team.Members))
		for _, member := range team.Members {
			names = append(names, member.Name)
		}
		sort.Strings(names)
		parts = append(parts, team.Name+":"+strings.Join(names, ","))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func canonicalSolutionFromNameMap(solution map[string][]string) string {
	parts := make([]string, 0, len(solution))
	for teamName, members := range solution {
		names := append([]string(nil), members...)
		sort.Strings(names)
		parts = append(parts, teamName+":"+strings.Join(names, ","))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func TestParticipantUnmarshalJSON_WithNullRating_ReturnsEmptyRatings(t *testing.T) {
	data := []byte(`{"name": "Ali", "rating": null}`)
	var p Participant
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got, want := p.Name, "Ali"; got != want {
		t.Fatalf("name = %s, want %s", got, want)
	}
	if got, want := len(p.Ratings), 0; got != want {
		t.Fatalf("rating count = %d, want %d", got, want)
	}
}

func TestParticipantUnmarshalJSON_WithSingleRating_ReturnsSingleRating(t *testing.T) {
	data := []byte(`{"name": "Ali", "rating": 9.5}`)
	var p Participant
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got, want := len(p.Ratings), 1; got != want {
		t.Fatalf("rating count = %d, want %d", got, want)
	}
	if got, want := p.Ratings[0], Rating(9.5); got != want {
		t.Fatalf("rating = %.1f, want %.1f", got, want)
	}
}

func TestParticipantUnmarshalJSON_WithMultipleRatings_ReturnsMultipleRatings(t *testing.T) {
	data := []byte(`{"name": "Ali", "rating": [10.0, 9.0, 8.0]}`)
	var p Participant
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got, want := len(p.Ratings), 3; got != want {
		t.Fatalf("rating count = %d, want %d", got, want)
	}
	expectedRatings := []Rating{10.0, 9.0, 8.0}
	for i, expected := range expectedRatings {
		if got := p.Ratings[i]; got != expected {
			t.Fatalf("rating[%d] = %.1f, want %.1f", i, got, expected)
		}
	}
}

func TestParticipantUnmarshalJSON_WithIsPlaceholder_SetsFlag(t *testing.T) {
	data := []byte(`{"name": "Placeholder 1", "rating": 5.0, "is_placeholder": true}`)
	var p Participant
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if !p.IsPlaceholder {
		t.Fatalf("IsPlaceholder = false, want true")
	}
}

func TestParticipantString_WithNoRatings_ReturnsNameOnly(t *testing.T) {
	p := &Participant{Name: "Ali"}
	if got, want := p.String(), "Ali"; got != want {
		t.Fatalf("String() = %s, want %s", got, want)
	}
}

func TestParticipantString_WithSingleRating_ReturnsNameAndRating(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{9.5}}
	result := p.String()
	if !strings.Contains(result, "Ali") || !strings.Contains(result, "9.5") {
		t.Fatalf("String() = %s, should contain name and rating", result)
	}
}

func TestParticipantString_WithMultipleRatings_ReturnsNameAndRatings(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{10.0, 9.0, 8.0}}
	result := p.String()
	if !strings.Contains(result, "Ali") || !strings.Contains(result, "10.0") {
		t.Fatalf("String() = %s, should contain name and ratings", result)
	}
}

func TestParticipantString_WithNilPointer_ReturnsEmpty(t *testing.T) {
	var p *Participant
	if got := p.String(); got != "" {
		t.Fatalf("String() = %s, want empty string for nil", got)
	}
}

func TestRatingMarshalJSON_WithNilPointer_ReturnsZero(t *testing.T) {
	var r *Rating
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	// When marshaling nil, standard json.Marshal returns "null"
	if got := string(data); got != "null" {
		t.Fatalf("MarshalJSON() = %s, want null", got)
	}
}

func TestRatingMarshalJSON_WithValidRating_FormatsToOneDecimal(t *testing.T) {
	r := Rating(9.5)
	data, err := json.Marshal(&r)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if got, want := string(data), "9.5"; got != want {
		t.Fatalf("MarshalJSON() = %s, want %s", got, want)
	}
}

func TestRatingUnmarshalJSON_WithValidFloat_ParsesCorrectly(t *testing.T) {
	data := []byte("9.5")
	var r Rating
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if got, want := r, Rating(9.5); got != want {
		t.Fatalf("rating = %.1f, want %.1f", got, want)
	}
}

func TestRatingUnmarshalJSON_WithInvalidJSON_ReturnsError(t *testing.T) {
	data := []byte("invalid")
	var r Rating
	if err := json.Unmarshal(data, &r); err == nil {
		t.Fatalf("UnmarshalJSON() expected error, got nil")
	}
}

func TestParticipantHasRating_WithRatings_ReturnsTrue(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{10.0}}
	if !p.HasRating() {
		t.Fatalf("HasRating() = false, want true")
	}
}

func TestParticipantHasRating_WithoutRatings_ReturnsFalse(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{}}
	if p.HasRating() {
		t.Fatalf("HasRating() = true, want false")
	}
}

func TestParticipantHasRating_WithNilPointer_ReturnsFalse(t *testing.T) {
	var p *Participant
	if p.HasRating() {
		t.Fatalf("HasRating() = true, want false for nil")
	}
}

func TestParticipantHasRating_WithMultipleRatings_ReturnsTrue(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{10.0, 9.0, 8.0}}
	if !p.HasRating() {
		t.Fatalf("HasRating() = false, want true for multiple ratings")
	}
}

func TestParticipantRatingCount_WithMultipleRatings_ReturnsCount(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{10.0, 9.0, 8.0}}
	if got, want := p.RatingCount(), 3; got != want {
		t.Fatalf("RatingCount() = %d, want %d", got, want)
	}
}

func TestParticipantRatingCount_WithoutRatings_ReturnsZero(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{}}
	if got, want := p.RatingCount(), 0; got != want {
		t.Fatalf("RatingCount() = %d, want %d", got, want)
	}
}

func TestParticipantRatingCount_WithNilPointer_ReturnsZero(t *testing.T) {
	var p *Participant
	if got := p.RatingCount(); got != 0 {
		t.Fatalf("RatingCount() = %d, want 0 for nil", got)
	}
}

func TestParticipantRatingCount_WithSingleRating_ReturnsOne(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{9.5}}
	if got, want := p.RatingCount(), 1; got != want {
		t.Fatalf("RatingCount() = %d, want %d", got, want)
	}
}

func TestParticipantScore_WithMultipleRatings_ReturnsAverage(t *testing.T) {
	p := &Participant{Name: "Ali", Ratings: []Rating{10.0, 8.0, 6.0}}
	expected := Rating(8.0)
	if got := p.Score(); got != expected {
		t.Fatalf("Score() = %.1f, want %.1f", got, expected)
	}
}

func TestParticipantScore_WithNilPointer_ReturnsZero(t *testing.T) {
	var p *Participant
	if got := p.Score(); got != 0 {
		t.Fatalf("Score() = %.1f, want 0 for nil", got)
	}
}

func TestNewMetaRating_WithValues_CreatesMetaRating(t *testing.T) {
	mr := NewMetaRating(Rating(10.0), Rating(9.0))
	if got, want := len(mr.Values), 2; got != want {
		t.Fatalf("Values length = %d, want %d", got, want)
	}
	if got, want := mr.Values[0], Rating(10.0); got != want {
		t.Fatalf("Values[0] = %.1f, want %.1f", got, want)
	}
}

func TestNewMetaRating_WithNoValues_CreatesEmptyMetaRating(t *testing.T) {
	mr := NewMetaRating()
	if got, want := len(mr.Values), 0; got != want {
		t.Fatalf("Values length = %d, want %d", got, want)
	}
}

func TestMetaRatingMarshalJSON_WithSingleValue_SerializesAsScalar(t *testing.T) {
	mr := NewMetaRating(Rating(9.5))
	data, err := json.Marshal(mr)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var value float64
	if err := json.Unmarshal(data, &value); err == nil {
		if got, want := Rating(value), Rating(9.5); got != want {
			t.Fatalf("MarshalJSON() = %.1f, want %.1f", got, want)
		}
	} else {
		t.Fatalf("MarshalJSON() produced invalid scalar: %v", err)
	}
}

func TestMetaRatingMarshalJSON_WithMultipleValues_SerializesAsArray(t *testing.T) {
	mr := NewMetaRating(Rating(10.0), Rating(9.0))
	data, err := json.Marshal(mr)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var values []float64
	if err := json.Unmarshal(data, &values); err == nil {
		if got, want := len(values), 2; got != want {
			t.Fatalf("MarshalJSON() array length = %d, want %d", got, want)
		}
	} else {
		t.Fatalf("MarshalJSON() did not produce array: %v", err)
	}
}

func TestMetaRatingMarshalJSON_WithNoValues_SerializesAsZero(t *testing.T) {
	mr := NewMetaRating()
	data, err := json.Marshal(mr)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if got, want := string(data), "0"; got != want {
		t.Fatalf("MarshalJSON() = %s, want %s", got, want)
	}
}

func TestFormatItems_WithEmptyList_ReturnsEmpty(t *testing.T) {
	result := FormatItems([]*Participant{})
	if got, want := result, ""; got != want {
		t.Fatalf("FormatItems() = %s, want empty string", got)
	}
}

func TestFormatItems_WithSingleItem_ReturnsSingleString(t *testing.T) {
	items := []*Participant{
		{Name: "Ali", Ratings: []Rating{10.0}},
	}
	result := FormatItems(items)
	if !strings.Contains(result, "Ali") {
		t.Fatalf("FormatItems() = %s, should contain item name", result)
	}
	if strings.Contains(result, ",") {
		t.Fatalf("FormatItems() = %s, should not contain comma for single item", result)
	}
}

func TestFormatItems_WithMultipleItems_FormatsWithCommas(t *testing.T) {
	items := []*Participant{
		{Name: "Ali", Ratings: []Rating{10.0}},
		{Name: "Mira", Ratings: []Rating{9.0}},
		{Name: "Bek", Ratings: []Rating{8.0}},
	}
	result := FormatItems(items)
	if !strings.Contains(result, ", ") {
		t.Fatalf("FormatItems() = %s, should contain comma separator", result)
	}
	if !strings.Contains(result, "Ali") || !strings.Contains(result, "Mira") || !strings.Contains(result, "Bek") {
		t.Fatalf("FormatItems() = %s, should contain all item names", result)
	}
}

func TestObjectiveBetter_WithLowerMaxDiff_ReturnsTrue(t *testing.T) {
	a := optimizationObjective{MaxDiff: Rating(2.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0), Diffs: []Rating{}}
	b := optimizationObjective{MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0), Diffs: []Rating{}}
	if !objectiveBetter(a, b, 1e-9) {
		t.Fatalf("objectiveBetter() = false, want true for lower max diff")
	}
}

func TestObjectiveBetter_WithEqualMaxDiffAndLowerAvgDiff_ReturnsTrue(t *testing.T) {
	a := optimizationObjective{MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0), Diffs: []Rating{}}
	b := optimizationObjective{MaxDiff: Rating(3.0), AvgDiff: Rating(2.0), SumDiff: Rating(5.0), Diffs: []Rating{}}
	if !objectiveBetter(a, b, 1e-9) {
		t.Fatalf("objectiveBetter() = false, want true for lower avg diff")
	}
}

func TestObjectiveBetter_WithEqualMaxAndAvgDiffAndLowerSumDiff_ReturnsTrue(t *testing.T) {
	a := optimizationObjective{MaxDiff: Rating(3.0), AvgDiff: Rating(1.5), SumDiff: Rating(4.0), Diffs: []Rating{}}
	b := optimizationObjective{MaxDiff: Rating(3.0), AvgDiff: Rating(1.5), SumDiff: Rating(5.0), Diffs: []Rating{}}
	if !objectiveBetter(a, b, 1e-9) {
		t.Fatalf("objectiveBetter() = false, want true for lower sum diff")
	}
}

func TestObjectiveBetter_WithGreaterMaxDiff_ReturnsFalse(t *testing.T) {
	a := optimizationObjective{MaxDiff: Rating(4.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0), Diffs: []Rating{}}
	b := optimizationObjective{MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0), Diffs: []Rating{}}
	if objectiveBetter(a, b, 1e-9) {
		t.Fatalf("objectiveBetter() = true, want false for greater max diff")
	}
}

func TestObjectiveBetter_WithLowerIndividualDiff_ReturnsTrue(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.5), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0, 2.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.5), SumDiff: Rating(5.0),
		Diffs: []Rating{2.0, 2.0},
	}
	if !objectiveBetter(a, b, 1e-9) {
		t.Fatalf("objectiveBetter() = false, want true for lower individual diff")
	}
}

func TestObjectiveEqual_WithEqualObjectives_ReturnsTrue(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0, 2.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0, 2.0},
	}
	if !objectiveEqual(a, b, 1e-9) {
		t.Fatalf("objectiveEqual() = false, want true for equal objectives")
	}
}

func TestObjectiveEqual_WithDifferentMaxDiff_ReturnsFalse(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(4.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0},
	}
	if objectiveEqual(a, b, 1e-9) {
		t.Fatalf("objectiveEqual() = true, want false for different max diff")
	}
}

func TestObjectiveEqual_WithDifferentAvgDiff_ReturnsFalse(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(2.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0},
	}
	if objectiveEqual(a, b, 1e-9) {
		t.Fatalf("objectiveEqual() = true, want false for different avg diff")
	}
}

func TestObjectiveEqual_WithDifferentSumDiff_ReturnsFalse(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(6.0),
		Diffs: []Rating{1.0},
	}
	if objectiveEqual(a, b, 1e-9) {
		t.Fatalf("objectiveEqual() = true, want false for different sum diff")
	}
}

func TestObjectiveEqual_WithDifferentDiffsCounts_ReturnsFalse(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0, 2.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0},
	}
	if objectiveEqual(a, b, 1e-9) {
		t.Fatalf("objectiveEqual() = true, want false for different diffs counts")
	}
}

func TestObjectiveEqual_WithDifferentIndividualDiff_ReturnsFalse(t *testing.T) {
	a := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0, 2.0},
	}
	b := optimizationObjective{
		MaxDiff: Rating(3.0), AvgDiff: Rating(1.0), SumDiff: Rating(5.0),
		Diffs: []Rating{1.0, 3.0},
	}
	if objectiveEqual(a, b, 1e-9) {
		t.Fatalf("objectiveEqual() = true, want false for different individual diff")
	}
}

func TestRandomSeed_ReturnsDifferentValuesOnMultipleCalls(t *testing.T) {
	seed1 := randomSeed()
	seed2 := randomSeed()
	if seed1 == seed2 {
		t.Fatalf("randomSeed() returned same value twice: %d == %d", seed1, seed2)
	}
}

func TestRandomSeed_ReturnsValidInt64(t *testing.T) {
	seed := randomSeed()
	if seed == 0 {
		t.Logf("randomSeed() returned 0, which is valid but unusual")
	}
}
