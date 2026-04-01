package teamsorter

import (
	"encoding/json"
	"fmt"
	"math"
)

type Rating float64

func normalizeRating(v float64) Rating {
	return Rating(math.Round(v*10) / 10)
}

func (r Rating) Float64() float64 {
	return float64(r)
}

func (r Rating) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.1f", r)), nil
}

func (r *Rating) UnmarshalJSON(data []byte) error {
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*r = normalizeRating(value)
	return nil
}

type Participant struct {
	Name          string `json:"name"`
	Rating        *Rating `json:"rating,omitempty"`
	IsPlaceholder bool   `json:"is_placeholder,omitempty"`
}

func (p Participant) String() string {
	if p.Rating == nil {
		return p.Name
	}
	return fmt.Sprintf("%s(%.1f)", p.Name, *p.Rating)
}

type SortTeamsRequest struct {
	NumberOfTeams int           `json:"number_of_teams"`
	Participants  []Participant `json:"participants"`
}

type Team struct {
	Name        string        `json:"name"`
	TotalRating Rating        `json:"total_rating"`
	Members     []Participant `json:"members"`
}

func (t Team) String() string {
	return fmt.Sprintf(
		"%s | Total: %.1f | Members: [%s]",
		t.Name,
		t.TotalRating,
		FormatItems(t.Members),
	)
}

type SortTeamsMeta struct {
	TeamCount        int    `json:"team_count"`
	ParticipantCount int    `json:"participant_count"`
	PlaceholderCount int    `json:"placeholder_count"`
	MembersPerTeam   int    `json:"members_per_team"`
	SolutionCount    int    `json:"solution_count"`
	MinTeamRating    Rating `json:"min_team_rating"`
	MaxTeamRating    Rating `json:"max_team_rating"`
	RatingDiff       Rating `json:"rating_diff"`
}

type SortTeamsResponse struct {
	Teams      []Team         `json:"teams,omitempty"`
	Meta       *SortTeamsMeta `json:"meta,omitempty"`
	Error      string         `json:"error,omitempty"`
	HasRatings bool           `json:"-"`
}

// MarshalJSON implements custom JSON marshaling to omit rating fields when not provided
func (r SortTeamsResponse) MarshalJSON() ([]byte, error) {
	if r.HasRatings {
		type Alias SortTeamsResponse
		return json.Marshal((*Alias)(&r))
	}

	// Remove rating fields from response when not originally provided
	type TeamNoRating struct {
		Name    string        `json:"name"`
		Members []Participant `json:"members"`
	}

	type MetaNoRating struct {
		TeamCount        int `json:"team_count"`
		ParticipantCount int `json:"participant_count"`
		PlaceholderCount int `json:"placeholder_count"`
		MembersPerTeam   int `json:"members_per_team"`
		SolutionCount    int `json:"solution_count"`
	}

	teams := make([]TeamNoRating, len(r.Teams))
	for i, t := range r.Teams {
		// Strip rating from members
		members := make([]Participant, len(t.Members))
		for j, m := range t.Members {
			members[j] = Participant{
				Name:          m.Name,
				Rating:        nil,
				IsPlaceholder: m.IsPlaceholder,
			}
		}
		teams[i] = TeamNoRating{
			Name:    t.Name,
			Members: members,
		}
	}

	meta := &MetaNoRating{
		TeamCount:        r.Meta.TeamCount,
		ParticipantCount: r.Meta.ParticipantCount,
		PlaceholderCount: r.Meta.PlaceholderCount,
		MembersPerTeam:   r.Meta.MembersPerTeam,
		SolutionCount:    r.Meta.SolutionCount,
	}

	if r.Error != "" {
		return json.Marshal(map[string]string{"error": r.Error})
	}

	return json.Marshal(map[string]interface{}{
		"teams": teams,
		"meta":  meta,
	})
}
