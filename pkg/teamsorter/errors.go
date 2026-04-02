package teamsorter

import "errors"

var (
	ErrInvalidTeamCount         = errors.New("number_of_teams must be at least 2")
	ErrInvalidParticipants      = errors.New("participants must not be empty")
	ErrTooManyTeams             = errors.New("number_of_teams must be less than or equal to participants count")
	ErrInsufficientParticipants = errors.New("participants count must be at least number_of_teams + 1")
	ErrEmptyParticipantName     = errors.New("participant name must not be empty")
	ErrInvalidParticipantRating = errors.New("participant rating must be between 1.0 and 10.0")
	ErrInconsistentRatings      = errors.New("all participants must have ratings or all must have no ratings")
	ErrInconsistentRatingCount  = errors.New("all participants must have the same number of rating values")
	ErrDuplicateParticipantName = errors.New("participant name must be unique")
	ErrNoSolution               = errors.New("no team sorting solution found")
)
