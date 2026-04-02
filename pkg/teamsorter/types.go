package teamsorter

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
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
	Name          string   `json:"name"`
	Ratings       []Rating `json:"-"`
	IsPlaceholder bool     `json:"is_placeholder,omitempty"`
}

func (p Participant) String() string {
	if len(p.Ratings) == 0 {
		return p.Name
	}
	if len(p.Ratings) == 1 {
		return fmt.Sprintf("%s(%.1f)", p.Name, p.Ratings[0])
	}
	parts := make([]string, len(p.Ratings))
	for i, r := range p.Ratings {
		parts[i] = fmt.Sprintf("%.1f", r)
	}
	return fmt.Sprintf("%s([%s])", p.Name, strings.Join(parts, ","))
}

func (p Participant) HasRating() bool {
	return len(p.Ratings) > 0
}

func (p Participant) RatingCount() int {
	return len(p.Ratings)
}

func (p Participant) Score() Rating {
	if len(p.Ratings) == 0 {
		return 0
	}

	total := 0.0
	for _, rating := range p.Ratings {
		total += rating.Float64()
	}

	return normalizeRating(total / float64(len(p.Ratings)))
}

func (p Participant) MarshalJSON() ([]byte, error) {
	type participantJSON struct {
		Name          string      `json:"name"`
		Rating        interface{} `json:"rating,omitempty"`
		IsPlaceholder bool        `json:"is_placeholder,omitempty"`
	}

	payload := participantJSON{
		Name:          p.Name,
		IsPlaceholder: p.IsPlaceholder,
	}

	if len(p.Ratings) == 1 {
		rating := p.Ratings[0]
		payload.Rating = rating
	} else if len(p.Ratings) > 1 {
		payload.Rating = p.Ratings
	}

	return json.Marshal(payload)
}

func (p *Participant) UnmarshalJSON(data []byte) error {
	type participantJSON struct {
		Name          string          `json:"name"`
		Rating        json.RawMessage `json:"rating"`
		IsPlaceholder bool            `json:"is_placeholder,omitempty"`
	}

	var payload participantJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	p.Name = payload.Name
	p.IsPlaceholder = payload.IsPlaceholder
	p.Ratings = nil

	if len(payload.Rating) == 0 || string(payload.Rating) == "null" {
		return nil
	}

	var list []Rating
	if err := json.Unmarshal(payload.Rating, &list); err == nil {
		p.Ratings = list
		return nil
	}

	var single Rating
	if err := json.Unmarshal(payload.Rating, &single); err != nil {
		return err
	}
	p.Ratings = []Rating{single}
	return nil
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

type MetaRating struct {
	Values []Rating
}

func NewMetaRating(values ...Rating) MetaRating {
	return MetaRating{Values: values}
}

func (m MetaRating) MarshalJSON() ([]byte, error) {
	if len(m.Values) <= 1 {
		if len(m.Values) == 0 {
			zero := Rating(0)
			return json.Marshal(zero)
		}
		return json.Marshal(m.Values[0])
	}
	return json.Marshal(m.Values)
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
	MinTeamRating    MetaRating `json:"min_team_rating"`
	MaxTeamRating    MetaRating `json:"max_team_rating"`
	RatingDiff       MetaRating `json:"rating_diff"`
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
				Ratings:       nil,
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
