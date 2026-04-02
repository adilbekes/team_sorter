package teamsorter

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
)

func SortTeams(req SortTeamsRequest) (*SortTeamsResponse, error) {
	if err := ValidateSortTeamsRequest(req); err != nil {
		return nil, err
	}

	// Use a per-call RNG with crypto seed to avoid repeated outputs across fast process starts.
	rng := rand.New(rand.NewSource(randomSeed()))

	participants := make([]Participant, len(req.Participants))
	copy(participants, req.Participants)

	// Assign random ratings if none are provided
	hasAnyRating := false
	for _, p := range participants {
		if p.HasRating() {
			hasAnyRating = true
			break
		}
	}
	if !hasAnyRating {
		for i := range participants {
			rating := normalizeRating(rng.Float64()*9.0 + 1.0) // [1.0, 10.0]
			participants[i].Ratings = []Rating{rating}
		}
	}

	participants = withMedianPlaceholders(participants, req.NumberOfTeams)
	placeholderCount := len(participants) - len(req.Participants)

	placeholderCap := 0
	if placeholderCount > 0 {
		placeholderCap = (placeholderCount + req.NumberOfTeams - 1) / req.NumberOfTeams
	}

	sort.SliceStable(participants, func(i, j int) bool {
		scoreI := participants[i].Score()
		scoreJ := participants[j].Score()
		if scoreI != scoreJ {
			return scoreI > scoreJ
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

	found, solutionCount, _ := fillTeamsOptimally(participants, teams, capacities, placeholderCap, rng, false)
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

// ListOptimalNameSolutions returns all optimal team assignments as name-only maps.
// Each solution item has team name as key and list of member names as value.
func ListOptimalNameSolutions(req SortTeamsRequest) ([]map[string][]string, error) {
	if err := ValidateSortTeamsRequest(req); err != nil {
		return nil, err
	}

	rng := rand.New(rand.NewSource(randomSeed()))

	participants := make([]Participant, len(req.Participants))
	copy(participants, req.Participants)

	hasAnyRating := false
	for _, p := range participants {
		if p.HasRating() {
			hasAnyRating = true
			break
		}
	}
	if !hasAnyRating {
		for i := range participants {
			rating := normalizeRating(rng.Float64()*9.0 + 1.0) // [1.0, 10.0]
			participants[i].Ratings = []Rating{rating}
		}
	}

	participants = withMedianPlaceholders(participants, req.NumberOfTeams)
	placeholderCount := len(participants) - len(req.Participants)

	placeholderCap := 0
	if placeholderCount > 0 {
		placeholderCap = (placeholderCount + req.NumberOfTeams - 1) / req.NumberOfTeams
	}

	sort.SliceStable(participants, func(i, j int) bool {
		scoreI := participants[i].Score()
		scoreJ := participants[j].Score()
		if scoreI != scoreJ {
			return scoreI > scoreJ
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

	found, _, allSolutions := fillTeamsOptimally(participants, teams, capacities, placeholderCap, rng, true)
	if !found {
		return nil, ErrNoSolution
	}

	result := make([]map[string][]string, len(allSolutions))
	for i, solution := range allSolutions {
		item := make(map[string][]string, len(solution))
		for _, team := range solution {
			names := make([]string, len(team.Members))
			for j, member := range team.Members {
				names[j] = member.Name
			}
			item[team.Name] = names
		}
		result[i] = item
	}

	return result, nil
}

func randomSeed() int64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err == nil {
		return int64(binary.LittleEndian.Uint64(b[:]))
	}
	return time.Now().UnixNano()
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
	median := medianRatings(participants)

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
			Ratings:       append([]Rating(nil), median...),
			IsPlaceholder: true,
		})
		placeholderIndex++
	}

	return participants
}

func medianRatings(participants []Participant) []Rating {
	if len(participants) == 0 || participants[0].RatingCount() == 0 {
		return nil
	}

	dimensions := participants[0].RatingCount()
	result := make([]Rating, dimensions)

	for d := 0; d < dimensions; d++ {
		ratings := make([]float64, len(participants))
		for i, participant := range participants {
			ratings[i] = participant.Ratings[d].Float64()
		}
		sort.Float64s(ratings)

		mid := len(ratings) / 2
		if len(ratings)%2 == 1 {
			result[d] = normalizeRating(ratings[mid])
		} else {
			result[d] = normalizeRating((ratings[mid-1] + ratings[mid]) / 2)
		}
	}

	return result
}

func fillTeamsOptimally(participants []Participant, teams []Team, capacities []int, placeholderCap int, rng *rand.Rand, collectAll bool) (bool, int, [][]Team) {
	const eps = 1e-9

	bestFound := false
	bestDiff := Rating(math.MaxFloat64)
	solutionCount := 0
	bestTeams := make([]Team, len(teams))
	allBestTeams := make([][]Team, 0)
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
				current := cloneTeams(teams)
				bestTeams = current
				if collectAll {
					allBestTeams = [][]Team{current}
				}
				bestFound = true
				solutionCount = 1
			} else if math.Abs(float64(diff-bestDiff)) <= eps {
				// Equally optimal solution — reservoir sampling
				solutionCount++
				current := cloneTeams(teams)
				if collectAll {
					allBestTeams = append(allBestTeams, current)
				} else if rng.Intn(solutionCount) == 0 {
					bestTeams = current
				}
			}
			return
		}

		participant := participants[index]
		participantScore := participant.Score()
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
			teams[i].TotalRating = normalizeRating(float64(teams[i].TotalRating + participantScore))
			if participant.IsPlaceholder {
				teamPlaceholders[i]++
			} else {
				teamRealMembers[i]++
			}

			search(index + 1)

			teams[i].Members = teams[i].Members[:len(teams[i].Members)-1]
			teams[i].TotalRating = normalizeRating(float64(teams[i].TotalRating - participantScore))
			if participant.IsPlaceholder {
				teamPlaceholders[i]--
			} else {
				teamRealMembers[i]--
			}
		}
	}

	search(0)

	if !bestFound {
		return false, 0, nil
	}

	for i := range teams {
		teams[i] = bestTeams[i]
	}

	return true, solutionCount, allBestTeams
}

func buildSortTeamsMeta(teams []Team, participantCount int, placeholderCount int, solutionCount int) *SortTeamsMeta {
	if len(teams) == 0 {
		return &SortTeamsMeta{
			TeamCount:        0,
			ParticipantCount: participantCount,
			PlaceholderCount: placeholderCount,
			MembersPerTeam:   0,
			SolutionCount:    0,
			MinTeamRating:    NewMetaRating(0),
			MaxTeamRating:    NewMetaRating(0),
			RatingDiff:       NewMetaRating(0),
		}
	}

	metricCount := teamMetricCount(teams[0])
	minValues := make([]Rating, metricCount)
	maxValues := make([]Rating, metricCount)
	firstMetrics := teamMetrics(teams[0])
	copy(minValues, firstMetrics)
	copy(maxValues, firstMetrics)

	for _, team := range teams[1:] {
		metrics := teamMetrics(team)
		for i := 0; i < metricCount; i++ {
			if metrics[i] < minValues[i] {
				minValues[i] = metrics[i]
			}
			if metrics[i] > maxValues[i] {
				maxValues[i] = metrics[i]
			}
		}
	}

	diffValues := make([]Rating, metricCount)
	for i := 0; i < metricCount; i++ {
		minValues[i] = normalizeRating(minValues[i].Float64())
		maxValues[i] = normalizeRating(maxValues[i].Float64())
		diffValues[i] = normalizeRating(maxValues[i].Float64() - minValues[i].Float64())
	}

	if metricCount == 2 {
		minValues = []Rating{minValues[1]}
		maxValues = []Rating{maxValues[1]}
		diffValues = []Rating{diffValues[1]}
	}

	return &SortTeamsMeta{
		TeamCount:        len(teams),
		ParticipantCount: participantCount,
		PlaceholderCount: placeholderCount,
		MembersPerTeam:   len(teams[0].Members),
		SolutionCount:    solutionCount,
		MinTeamRating:    MetaRating{Values: minValues},
		MaxTeamRating:    MetaRating{Values: maxValues},
		RatingDiff:       MetaRating{Values: diffValues},
	}
}

func teamMetricCount(team Team) int {
	for _, member := range team.Members {
		if member.RatingCount() > 0 {
			// n criteria + 1 total score metric
			return member.RatingCount() + 1
		}
	}
	return 1
}

func teamMetrics(team Team) []Rating {
	metricCount := teamMetricCount(team)
	metrics := make([]Rating, metricCount)
	if metricCount == 1 {
		metrics[0] = team.TotalRating
		return metrics
	}

	for _, member := range team.Members {
		for i, rating := range member.Ratings {
			metrics[i] = normalizeRating(metrics[i].Float64() + rating.Float64())
		}
	}

	metrics[metricCount-1] = team.TotalRating
	return metrics
}
