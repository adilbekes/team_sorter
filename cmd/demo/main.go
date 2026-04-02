package main

import (
	"fmt"
	"log"
	"strings"
	"team_sorter/pkg/teamsorter"
)

func oneRating(r float64) []teamsorter.Rating {
	return []teamsorter.Rating{teamsorter.Rating(r)}
}

func main() {
	req := teamsorter.SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []teamsorter.Participant{
			{Name: "Ali", Ratings: oneRating(10.0)},
			{Name: "Mira", Ratings: oneRating(9.0)},
			{Name: "Bek", Ratings: oneRating(8.0)},
			{Name: "Dana", Ratings: oneRating(7.0)},
			{Name: "Nurlan", Ratings: oneRating(6.0)},
			{Name: "Aruzhan", Ratings: oneRating(5.0)},
		},
	}

	parts := make([]string, len(req.Participants))
	for i := range req.Participants {
		parts[i] = req.Participants[i].String()
	}
	fmt.Printf("Participants: %s\n", strings.Join(parts, ", "))
	fmt.Printf("Number of teams: %d\n\n", req.NumberOfTeams)

	result, err := teamsorter.SortTeams(req)
	if err != nil {
		log.Fatalf("sorting failed: %v", err)
	}

	for _, team := range result.Teams {
		fmt.Println(team.String())
	}

	fmt.Println("\nMeta:")
	fmt.Printf("  Teams: %d\n", result.Meta.TeamCount)
	fmt.Printf("  Participants: %d\n", result.Meta.ParticipantCount)
	fmt.Printf("  Placeholders: %d\n", result.Meta.PlaceholderCount)
	fmt.Printf("  Members per team: %d\n", result.Meta.MembersPerTeam)
	fmt.Printf("  Optimal solutions: %d\n", result.Meta.SolutionCount)
	fmt.Printf("  Min rating: %.1f\n", result.Meta.MinTeamRating)
	fmt.Printf("  Max rating: %.1f\n", result.Meta.MaxTeamRating)
	fmt.Printf("  Rating diff: %.1f\n", result.Meta.RatingDiff)
}
