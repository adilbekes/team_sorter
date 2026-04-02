package teamsorter

import (
	"math"
	"regexp"
	"strings"
)

var reservedPlaceholderNamePattern = regexp.MustCompile(`(?i)^placeholder\s+\d+$`)

func ValidateSortTeamsRequest(req SortTeamsRequest) error {
	if req.NumberOfTeams < 2 {
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
	expectedRatingCount := 0
	for _, participant := range req.Participants {
		if participant.HasRating() {
			hasRating = true
			if expectedRatingCount == 0 {
				expectedRatingCount = participant.RatingCount()
			}
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

		if reservedPlaceholderNamePattern.MatchString(name) {
			return ErrReservedPlaceholderName
		}

		if participant.HasRating() {
			if participant.RatingCount() != expectedRatingCount {
				return ErrInconsistentRatingCount
			}

			for _, value := range participant.Ratings {
				rating := value.Float64()
				if math.IsNaN(rating) || math.IsInf(rating, 0) {
					return ErrInvalidParticipantRating
				}
				if rating < 1.0 || rating > 10.0 {
					return ErrInvalidParticipantRating
				}
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
