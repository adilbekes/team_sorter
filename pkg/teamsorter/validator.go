package teamsorter

import (
	"math"
	"strings"
)

func ValidateSortTeamsRequest(req SortTeamsRequest) error {
	if req.NumberOfTeams <= 0 {
		return ErrInvalidTeamCount
	}

	if len(req.Participants) == 0 {
		return ErrInvalidParticipants
	}

	if req.NumberOfTeams > len(req.Participants) {
		return ErrTooManyTeams
	}

	if len(req.Participants) < req.NumberOfTeams+1 {
		return ErrInsufficientParticipants
	}

	seenNames := make(map[string]struct{}, len(req.Participants))

	// Check rating consistency: all have ratings or none have ratings
	hasRating := false
	hasNoRating := false
	for _, participant := range req.Participants {
		if participant.Rating != nil {
			hasRating = true
		} else {
			hasNoRating = true
		}
	}
	if hasRating && hasNoRating {
		return ErrInconsistentRatings
	}

	for _, participant := range req.Participants {
		name := strings.TrimSpace(participant.Name)
		if name == "" {
			return ErrEmptyParticipantName
		}

		// Only validate rating if provided
		if participant.Rating != nil {
			if math.IsNaN(participant.Rating.Float64()) || math.IsInf(participant.Rating.Float64(), 0) {
				return ErrInvalidParticipantRating
			}

			rating := participant.Rating.Float64()
			if rating < 1.0 || rating > 10.0 {
				return ErrInvalidParticipantRating
			}
		}

		key := strings.ToLower(name)
		if _, exists := seenNames[key]; exists {
			return ErrDuplicateParticipantName
		}
		seenNames[key] = struct{}{}
	}

	return nil
}
