package teamsorter

import "testing"

func ptrRating(r float64) *Rating {
	rating := Rating(r)
	return &rating
}

func TestSortTeams_AddsMedianPlaceholderWhenNotDivisible(t *testing.T) {
	req := SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []Participant{
			{Name: "Ali", Rating: ptrRating(10.0)},
			{Name: "Mira", Rating: ptrRating(9.0)},
			{Name: "Bek", Rating: ptrRating(8.0)},
			{Name: "Dana", Rating: ptrRating(7.0)},
			{Name: "Nurlan", Rating: ptrRating(6.0)},
			{Name: "Aruzhan", Rating: ptrRating(5.0)},
			{Name: "Timur", Rating: ptrRating(4.0)},
			{Name: "Aida", Rating: ptrRating(3.0)},
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
				if *member.Rating != want {
					t.Fatalf("placeholder rating = %.1f, want %.1f", *member.Rating, want)
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
			{Name: "Ali", Rating: ptrRating(10.0)},
			{Name: "Mira", Rating: ptrRating(9.0)},
			{Name: "Bek", Rating: ptrRating(8.0)},
			{Name: "Dana", Rating: ptrRating(7.0)},
			{Name: "Nurlan", Rating: ptrRating(6.0)},
			{Name: "Aruzhan", Rating: ptrRating(5.0)},
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
