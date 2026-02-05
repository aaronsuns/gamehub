package live

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aaron/gamehub/internal/atlas"
)

// Service derives live teams and players from live series.
type Service struct {
	client *atlas.Client
	cache  *Cache
}

// NewService creates a live service with a TTL cache.
func NewService(client *atlas.Client, ttl time.Duration) *Service {
	s := &Service{client: client}
	s.cache = NewCache(ttl, func() (LiveContext, error) {
		return s.loadLiveContext(context.Background())
	})
	return s
}

// loadLiveContext performs the full API flow: series -> roster IDs -> rosters -> team/player IDs.
func (s *Service) loadLiveContext(ctx context.Context) (LiveContext, error) {
	seriesBody, _, err := s.client.GetSeriesAll(ctx, map[string]string{"filter": "lifecycle=live"})
	if err != nil {
		return LiveContext{}, err
	}
	rosterIDs := extractRosterIDsFromSeries(seriesBody)
	if len(rosterIDs) == 0 {
		return LiveContext{TeamIDs: []int{}, PlayerIDs: []int{}}, nil
	}
	// Server-side filter: Atlas API returns only these rosters (Multiple Rosters by id).
	rostersBody, _, err := s.client.GetRostersAll(ctx, map[string]string{
		"filter": atlas.FilterIDIn(rosterIDs),
	})
	if err != nil {
		return LiveContext{}, err
	}
	teamIDs, playerIDs := extractTeamAndPlayerIDsFromRosters(rostersBody)
	return LiveContext{TeamIDs: teamIDs, PlayerIDs: playerIDs}, nil
}

// GetLiveContext returns the cached or freshly loaded live context.
func (s *Service) GetLiveContext(ctx context.Context) (LiveContext, error) {
	return s.cache.Get()
}

func extractRosterIDsFromSeries(data []byte) []int {
	var series []map[string]interface{}
	if err := json.Unmarshal(data, &series); err != nil {
		return nil
	}
	seen := make(map[int]bool)
	for _, s := range series {
		parts, ok := s["participants"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range parts {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			roster, ok := pm["roster"].(map[string]interface{})
			if !ok {
				continue
			}
			id, ok := numID(roster["id"])
			if ok {
				seen[id] = true
			}
		}
	}
	out := make([]int, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out
}

func extractTeamAndPlayerIDsFromRosters(data []byte) (teamIDs, playerIDs []int) {
	var rosters []map[string]interface{}
	if err := json.Unmarshal(data, &rosters); err != nil {
		return nil, nil
	}
	teams := make(map[int]bool)
	players := make(map[int]bool)
	for _, r := range rosters {
		if team, ok := r["team"].(map[string]interface{}); ok {
			if id, ok := numID(team["id"]); ok {
				teams[id] = true
			}
		}
		if lineUp, ok := r["line_up"].(map[string]interface{}); ok {
			if plist, ok := lineUp["players"].([]interface{}); ok {
				for _, p := range plist {
					pm, ok := p.(map[string]interface{})
					if !ok {
						continue
					}
					if id, ok := numID(pm["id"]); ok {
						players[id] = true
					}
				}
			}
		}
	}
	teamIDs = make([]int, 0, len(teams))
	for id := range teams {
		teamIDs = append(teamIDs, id)
	}
	playerIDs = make([]int, 0, len(players))
	for id := range players {
		playerIDs = append(playerIDs, id)
	}
	return teamIDs, playerIDs
}

func numID(v interface{}) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	default:
		return 0, false
	}
}
