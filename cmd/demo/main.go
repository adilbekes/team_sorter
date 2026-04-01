package main

import (
	"fmt"
	"log"
	"team_sorter/pkg/teamsorter"
)

func ptrRating(r float64) *teamsorter.Rating {
	rating := teamsorter.Rating(r)
	return &rating
}

func main() {
	req := teamsorter.SortTeamsRequest{
		NumberOfTeams: 3,
		Participants: []teamsorter.Participant{
			{Name: "Ali", Rating: ptrRating(10.0)},
			{Name: "Mira", Rating: ptrRating(9.0)},
			{Name: "Bek", Rating: ptrRating(8.0)},
			{Name: "Dana", Rating: ptrRating(7.0)},
			{Name: "Nurlan", Rating: ptrRating(6.0)},
			{Name: "Aruzhan", Rating: ptrRating(5.0)},
		},
	}

	fmt.Printf("Participants: %s\n", teamsorter.FormatItems(req.Participants))
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
