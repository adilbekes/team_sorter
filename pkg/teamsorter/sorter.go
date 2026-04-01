package teamsorter

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
)

func SortTeams(req SortTeamsRequest) (*SortTeamsResponse, error) {
	if err := ValidateSortTeamsRequest(req); err != nil {
		return nil, err
	}

	participants := make([]Participant, len(req.Participants))
	copy(participants, req.Participants)

	// Assign random ratings if none are provided
	hasAnyRating := false
	for _, p := range participants {
		if p.Rating != nil {
			hasAnyRating = true
			break
		}
	}
	if !hasAnyRating {
		for i := range participants {
			rating := normalizeRating(rand.Float64()*9.0 + 1.0) // [1.0, 10.0]
			participants[i].Rating = &rating
		}
	}

	participants = withMedianPlaceholders(participants, req.NumberOfTeams)
	placeholderCount := len(participants) - len(req.Participants)

	placeholderCap := 0
	if placeholderCount > 0 {
		placeholderCap = (placeholderCount + req.NumberOfTeams - 1) / req.NumberOfTeams
	}

	sort.SliceStable(participants, func(i, j int) bool {
		if *participants[i].Rating != *participants[j].Rating {
			return *participants[i].Rating > *participants[j].Rating
		}

		return strings.ToLower(strings.TrimSpace(participants[i].Name)) <
			strings.ToLower(strings.TrimSpace(participants[j].Name))
	})

	teams := make([]Team, req.NumberOfTeams)
	baseSize := len(participants) / req.NumberOfTeams
	extraSlots := len(participants) % req.NumberOfTeams
	capacities := make([]int, req.NumberOfTeams)

	for i := range teams {
		capacity := baseSize
		if i < extraSlots {
			capacity++
		}
		capacities[i] = capacity

		teams[i] = Team{
			Name:        fmt.Sprintf("Team %d", i+1),
			TotalRating: 0,
			Members:     make([]Participant, 0, capacity),
		}
	}

	found, solutionCount := fillTeamsOptimally(participants, teams, capacities, placeholderCap)
	if !found {
		return nil, ErrNoSolution
	}

	meta := buildSortTeamsMeta(teams, len(req.Participants), placeholderCount, solutionCount)

	return &SortTeamsResponse{
		Teams:      teams,
		Meta:       meta,
		HasRatings: hasAnyRating,
	}, nil
}

func withMedianPlaceholders(participants []Participant, teamCount int) []Participant {
	if teamCount <= 0 || len(participants) == 0 {
		return participants
	}

	remainder := len(participants) % teamCount
	if remainder == 0 {
		return participants
	}

	missing := teamCount - remainder
	median := medianRating(participants)

	usedNames := make(map[string]struct{}, len(participants)+missing)
	for _, p := range participants {
		usedNames[strings.ToLower(strings.TrimSpace(p.Name))] = struct{}{}
	}

	placeholderIndex := 1
	for i := 0; i < missing; i++ {
		name := fmt.Sprintf("Placeholder %d", placeholderIndex)
		for {
			key := strings.ToLower(strings.TrimSpace(name))
			if _, exists := usedNames[key]; !exists {
				usedNames[key] = struct{}{}
				break
			}
			placeholderIndex++
			name = fmt.Sprintf("Placeholder %d", placeholderIndex)
		}

		participants = append(participants, Participant{
			Name:          name,
			Rating:        median,
			IsPlaceholder: true,
		})
		placeholderIndex++
	}

	return participants
}

func medianRating(participants []Participant) *Rating {
	ratings := make([]float64, len(participants))
	for i, participant := range participants {
		ratings[i] = participant.Rating.Float64()
	}
	sort.Float64s(ratings)

	mid := len(ratings) / 2
	var result Rating
	if len(ratings)%2 == 1 {
		result = normalizeRating(ratings[mid])
	} else {
		result = normalizeRating((ratings[mid-1] + ratings[mid]) / 2)
	}
	return &result
}

func fillTeamsOptimally(participants []Participant, teams []Team, capacities []int, placeholderCap int) (bool, int) {
	const eps = 1e-9

	bestFound := false
	bestDiff := Rating(math.MaxFloat64)
	solutionCount := 0
	bestTeams := make([]Team, len(teams))
	teamPlaceholders := make([]int, len(teams))
	teamRealMembers := make([]int, len(teams))

	cloneTeams := func(source []Team) []Team {
		cloned := make([]Team, len(source))
		for i := range source {
			cloned[i] = Team{
				Name:        source[i].Name,
				TotalRating: source[i].TotalRating,
				Members:     append([]Participant(nil), source[i].Members...),
			}
		}
		return cloned
	}

	minMax := func(current []Team) (Rating, Rating) {
		minRating := current[0].TotalRating
		maxRating := current[0].TotalRating
		for _, t := range current[1:] {
			if t.TotalRating < minRating {
				minRating = t.TotalRating
			}
			if t.TotalRating > maxRating {
				maxRating = t.TotalRating
			}
		}
		return minRating, maxRating
	}

	hasAtLeastOneRealMember := func() bool {
		for i := range teams {
			if teamRealMembers[i] == 0 {
				return false
			}
		}
		return true
	}

	var search func(index int)
	search = func(index int) {
		if index == len(participants) {
			// Validate: each team must have at least 1 real participant
			if !hasAtLeastOneRealMember() {
				return
			}

			_, maxRating := minMax(teams)
			minRating, _ := minMax(teams)
			diff := normalizeRating(float64(maxRating - minRating))

			if !bestFound || float64(diff) < float64(bestDiff)-eps {
				// Strictly better solution — reset
				bestDiff = diff
				bestTeams = cloneTeams(teams)
				bestFound = true
				solutionCount = 1
			} else if math.Abs(float64(diff-bestDiff)) <= eps {
				// Equally optimal solution — reservoir sampling
				solutionCount++
				if rand.Intn(solutionCount) == 0 {
					bestTeams = cloneTeams(teams)
				}
			}
			return
		}

		participant := participants[index]
		for i := range teams {
			if len(teams[i].Members) >= capacities[i] {
				continue
			}

			if participant.IsPlaceholder && placeholderCap > 0 && teamPlaceholders[i] >= placeholderCap {
				continue
			}

			symmetric := false
			for j := 0; j < i; j++ {
				if len(teams[j].Members) >= capacities[j] {
					continue
				}
				// Only prune when BOTH teams are completely empty.
				// That is the only case they are truly interchangeable
				// (same future possibilities). Once any member is assigned,
				// teams with equal totals but different members produce
				// distinct solutions and must not be skipped.
				if len(teams[j].Members) == 0 && len(teams[i].Members) == 0 &&
					teamPlaceholders[j] == teamPlaceholders[i] {
					symmetric = true
					break
				}
			}
			if symmetric {
				continue
			}

			teams[i].Members = append(teams[i].Members, participant)
			teams[i].TotalRating = normalizeRating(float64(teams[i].TotalRating + *participant.Rating))
			if participant.IsPlaceholder {
				teamPlaceholders[i]++
			} else {
				teamRealMembers[i]++
			}

			search(index + 1)

			teams[i].Members = teams[i].Members[:len(teams[i].Members)-1]
			teams[i].TotalRating = normalizeRating(float64(teams[i].TotalRating - *participant.Rating))
			if participant.IsPlaceholder {
				teamPlaceholders[i]--
			} else {
				teamRealMembers[i]--
			}
		}
	}

	search(0)

	if !bestFound {
		return false, 0
	}

	for i := range teams {
		teams[i] = bestTeams[i]
	}

	return true, solutionCount
}

func buildSortTeamsMeta(teams []Team, participantCount int, placeholderCount int, solutionCount int) *SortTeamsMeta {
	if len(teams) == 0 {
		return &SortTeamsMeta{
			TeamCount:        0,
			ParticipantCount: participantCount,
			PlaceholderCount: placeholderCount,
			MembersPerTeam:   0,
			SolutionCount:    0,
			MinTeamRating:    0,
			MaxTeamRating:    0,
			RatingDiff:       0,
		}
	}

	minRating := teams[0].TotalRating
	maxRating := teams[0].TotalRating

	for _, team := range teams[1:] {
		if team.TotalRating < minRating {
			minRating = team.TotalRating
		}
		if team.TotalRating > maxRating {
			maxRating = team.TotalRating
		}
	}

	return &SortTeamsMeta{
		TeamCount:        len(teams),
		ParticipantCount: participantCount,
		PlaceholderCount: placeholderCount,
		MembersPerTeam:   len(teams[0].Members),
		SolutionCount:    solutionCount,
		MinTeamRating:    normalizeRating(float64(minRating)),
		MaxTeamRating:    normalizeRating(float64(maxRating)),
		RatingDiff:       normalizeRating(float64(maxRating - minRating)),
	}
}
