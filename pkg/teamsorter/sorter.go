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

	hasAnyRating := hasAnyRatings(participants)

	participants = withMedianPlaceholders(participants, req.NumberOfTeams)
	placeholderCount := len(participants) - len(req.Participants)

	placeholderCap := 0
	if placeholderCount > 0 {
		placeholderCap = (placeholderCount + req.NumberOfTeams - 1) / req.NumberOfTeams
	}

	teams, capacities := initializeTeams(len(participants), req.NumberOfTeams)

	if !hasAnyRating {
		if ok := fillTeamsRandomlyUnrated(participants, teams, capacities, placeholderCap, rng); !ok {
			return nil, ErrNoSolution
		}

		meta := buildSortTeamsMeta(teams, len(req.Participants), placeholderCount, 1)
		return &SortTeamsResponse{
			Teams:      teams,
			Meta:       meta,
			HasRatings: false,
		}, nil
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
	hasAnyRating := hasAnyRatings(participants)

	participants = withMedianPlaceholders(participants, req.NumberOfTeams)
	placeholderCount := len(participants) - len(req.Participants)

	placeholderCap := 0
	if placeholderCount > 0 {
		placeholderCap = (placeholderCount + req.NumberOfTeams - 1) / req.NumberOfTeams
	}

	if !hasAnyRating {
		return listAllNameSolutionsUnrated(participants, req.NumberOfTeams, placeholderCap), nil
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

	teams, capacities := initializeTeams(len(participants), req.NumberOfTeams)

	found, _, allSolutions := fillTeamsOptimally(participants, teams, capacities, placeholderCap, rng, true)
	if !found {
		return nil, ErrNoSolution
	}

	// Deduplicate solutions that differ only by placeholder swaps
	allSolutions = deduplicateSolutions(allSolutions)

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

func hasAnyRatings(participants []Participant) bool {
	for _, participant := range participants {
		if participant.HasRating() {
			return true
		}
	}
	return false
}

func initializeTeams(participantCount int, teamCount int) ([]Team, []int) {
	teams := make([]Team, teamCount)
	baseSize := participantCount / teamCount
	extraSlots := participantCount % teamCount
	capacities := make([]int, teamCount)

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

	return teams, capacities
}

func fillTeamsRandomlyUnrated(participants []Participant, teams []Team, capacities []int, placeholderCap int, rng *rand.Rand) bool {
	realParticipants := make([]Participant, 0, len(participants))
	placeholders := make([]Participant, 0, len(participants))
	for _, participant := range participants {
		if participant.IsPlaceholder {
			placeholders = append(placeholders, participant)
		} else {
			realParticipants = append(realParticipants, participant)
		}
	}

	if len(realParticipants) < len(teams) {
		return false
	}

	rng.Shuffle(len(realParticipants), func(i, j int) {
		realParticipants[i], realParticipants[j] = realParticipants[j], realParticipants[i]
	})
	rng.Shuffle(len(placeholders), func(i, j int) {
		placeholders[i], placeholders[j] = placeholders[j], placeholders[i]
	})

	teamOrder := rng.Perm(len(teams))
	teamPlaceholders := make([]int, len(teams))
	teamSizes := make([]int, len(teams))

	for i := 0; i < len(teams); i++ {
		teamIdx := teamOrder[i]
		participant := realParticipants[i]
		teams[teamIdx].Members = append(teams[teamIdx].Members, participant)
		teams[teamIdx].TotalRating = normalizeRating(float64(teams[teamIdx].TotalRating + participant.Score()))
		teamSizes[teamIdx]++
	}

	remaining := append([]Participant(nil), realParticipants[len(teams):]...)
	remaining = append(remaining, placeholders...)
	rng.Shuffle(len(remaining), func(i, j int) {
		remaining[i], remaining[j] = remaining[j], remaining[i]
	})

	for _, participant := range remaining {
		candidates := make([]int, 0, len(teams))
		for i := range teams {
			if teamSizes[i] >= capacities[i] {
				continue
			}
			if participant.IsPlaceholder && placeholderCap > 0 && teamPlaceholders[i] >= placeholderCap {
				continue
			}
			candidates = append(candidates, i)
		}

		if len(candidates) == 0 {
			return false
		}

		teamIdx := candidates[rng.Intn(len(candidates))]
		teams[teamIdx].Members = append(teams[teamIdx].Members, participant)
		teams[teamIdx].TotalRating = normalizeRating(float64(teams[teamIdx].TotalRating + participant.Score()))
		teamSizes[teamIdx]++
		if participant.IsPlaceholder {
			teamPlaceholders[teamIdx]++
		}
	}

	for i := range teams {
		if teamSizes[i] != capacities[i] {
			return false
		}
	}

	return true
}

func listAllNameSolutionsUnrated(participants []Participant, teamCount int, placeholderCap int) []map[string][]string {
	sortedParticipants := append([]Participant(nil), participants...)
	sort.SliceStable(sortedParticipants, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(sortedParticipants[i].Name)) <
			strings.ToLower(strings.TrimSpace(sortedParticipants[j].Name))
	})

	teams, capacities := initializeTeams(len(sortedParticipants), teamCount)
	teamPlaceholders := make([]int, len(teams))
	teamRealMembers := make([]int, len(teams))

	hasAtLeastOneRealMember := func() bool {
		for i := range teams {
			if teamRealMembers[i] == 0 {
				return false
			}
		}
		return true
	}

	results := make([]map[string][]string, 0)

	var search func(index int)
	search = func(index int) {
		if index == len(sortedParticipants) {
			if !hasAtLeastOneRealMember() {
				return
			}

			solution := make(map[string][]string, len(teams))
			for _, team := range teams {
				names := make([]string, len(team.Members))
				for i, member := range team.Members {
					names[i] = member.Name
				}
				solution[team.Name] = names
			}
			results = append(results, solution)
			return
		}

		participant := sortedParticipants[index]
		for i := range teams {
			if len(teams[i].Members) >= capacities[i] {
				continue
			}
			if participant.IsPlaceholder && placeholderCap > 0 && teamPlaceholders[i] >= placeholderCap {
				continue
			}

			teams[i].Members = append(teams[i].Members, participant)
			if participant.IsPlaceholder {
				teamPlaceholders[i]++
			} else {
				teamRealMembers[i]++
			}

			search(index + 1)

			teams[i].Members = teams[i].Members[:len(teams[i].Members)-1]
			if participant.IsPlaceholder {
				teamPlaceholders[i]--
			} else {
				teamRealMembers[i]--
			}
		}
	}

	search(0)
	return results
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
			ratings[i] = float64(participant.Ratings[d])
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
	bestObjective := optimizationObjective{MaxDiff: Rating(math.MaxFloat64)}
	solutionCount := 0
	bestTeams := make([]Team, len(teams))
	allBestTeams := make([][]Team, 0)
	seenOptimalSignatures := make(map[string]struct{})
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

			objective := computeOptimizationObjective(teams)
			signature := solutionSignature(teams)

			if !bestFound || objectiveBetter(objective, bestObjective, eps) {
				// Strictly better solution — reset
				bestObjective = objective
				current := cloneTeams(teams)
				bestTeams = current
				if collectAll {
					allBestTeams = [][]Team{current}
				}
				bestFound = true
				solutionCount = 1
				seenOptimalSignatures = map[string]struct{}{signature: {}}
			} else if objectiveEqual(objective, bestObjective, eps) {
				if _, exists := seenOptimalSignatures[signature]; exists {
					return
				}
				seenOptimalSignatures[signature] = struct{}{}

				// New unique equally optimal solution — reservoir sampling on unique set
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

// solutionSignature creates a canonical string representation of a solution
// that treats placeholders with the same rating as interchangeable.
func solutionSignature(teams []Team) string {
	var parts []string
	for _, team := range teams {
		// Sort members by name for canonical ordering
		members := make([]Participant, len(team.Members))
		copy(members, team.Members)
		sort.Slice(members, func(i, j int) bool {
			return members[i].Name < members[j].Name
		})

		// Build signature, ignoring placeholder-specific names
		var memberSigs []string
		for _, m := range members {
			if m.IsPlaceholder {
				// Treat all placeholders with same rating as identical
				memberSigs = append(memberSigs, fmt.Sprintf("PH:%.1f", float64(m.Score())))
			} else {
				memberSigs = append(memberSigs, m.Name)
			}
		}
		parts = append(parts, fmt.Sprintf("%s:[%s]", team.Name, strings.Join(memberSigs, ",")))
	}
	return strings.Join(parts, "|")
}

// deduplicateSolutions removes solutions that differ only by placeholder swaps
func deduplicateSolutions(allTeams [][]Team) [][]Team {
	seen := make(map[string]bool)
	var result [][]Team

	for _, teams := range allTeams {
		sig := solutionSignature(teams)
		if !seen[sig] {
			seen[sig] = true
			result = append(result, teams)
		}
	}

	return result
}

type optimizationObjective struct {
	MaxDiff Rating
	AvgDiff Rating
	SumDiff Rating
	Diffs   []Rating
}

func computeOptimizationObjective(teams []Team) optimizationObjective {
	diffs := optimizationDiffs(teams)
	obj := optimizationObjective{Diffs: diffs}
	if len(diffs) == 0 {
		return obj
	}

	maxDiff := diffs[0]
	sum := 0.0
	for _, d := range diffs {
		if d > maxDiff {
			maxDiff = d
		}
		sum += float64(d)
	}

	obj.MaxDiff = normalizeRating(float64(maxDiff))
	obj.AvgDiff = normalizeRating(float64(diffs[len(diffs)-1]))
	obj.SumDiff = normalizeRating(sum)
	return obj
}

func optimizationDiffs(teams []Team) []Rating {
	if len(teams) == 0 {
		return []Rating{0}
	}

	metricCount := teamMetricCount(teams[0])
	minValues := make([]Rating, metricCount)
	maxValues := make([]Rating, metricCount)
	first := teamMetrics(teams[0])
	copy(minValues, first)
	copy(maxValues, first)

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

	diffs := make([]Rating, metricCount)
	for i := 0; i < metricCount; i++ {
		diffs[i] = normalizeRating(float64(maxValues[i]) - float64(minValues[i]))
	}

	// Single-rating case has [criterion,avg] where criterion == avg; optimize by avg only.
	if metricCount == 2 {
		return []Rating{diffs[1]}
	}

	return diffs
}

func objectiveBetter(a optimizationObjective, b optimizationObjective, eps float64) bool {
	if float64(a.MaxDiff) < float64(b.MaxDiff)-eps {
		return true
	}
	if math.Abs(float64(a.MaxDiff-b.MaxDiff)) > eps {
		return false
	}

	if float64(a.AvgDiff) < float64(b.AvgDiff)-eps {
		return true
	}
	if math.Abs(float64(a.AvgDiff-b.AvgDiff)) > eps {
		return false
	}

	if float64(a.SumDiff) < float64(b.SumDiff)-eps {
		return true
	}
	if math.Abs(float64(a.SumDiff-b.SumDiff)) > eps {
		return false
	}

	for i := range a.Diffs {
		if i >= len(b.Diffs) {
			return false
		}
		if float64(a.Diffs[i]) < float64(b.Diffs[i])-eps {
			return true
		}
		if math.Abs(float64(a.Diffs[i]-b.Diffs[i])) > eps {
			return false
		}
	}

	return false
}

func objectiveEqual(a optimizationObjective, b optimizationObjective, eps float64) bool {
	if len(a.Diffs) != len(b.Diffs) {
		return false
	}
	if math.Abs(float64(a.MaxDiff-b.MaxDiff)) > eps {
		return false
	}
	if math.Abs(float64(a.AvgDiff-b.AvgDiff)) > eps {
		return false
	}
	if math.Abs(float64(a.SumDiff-b.SumDiff)) > eps {
		return false
	}
	for i := range a.Diffs {
		if math.Abs(float64(a.Diffs[i]-b.Diffs[i])) > eps {
			return false
		}
	}
	return true
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
		minValues[i] = normalizeRating(float64(minValues[i]))
		maxValues[i] = normalizeRating(float64(maxValues[i]))
		diffValues[i] = normalizeRating(float64(maxValues[i]) - float64(minValues[i]))
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
			metrics[i] = normalizeRating(float64(metrics[i]) + float64(rating))
		}
	}

	metrics[metricCount-1] = team.TotalRating
	return metrics
}
