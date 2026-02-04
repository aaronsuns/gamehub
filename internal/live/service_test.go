package live

import (
	"testing"
)

func TestExtractRosterIDsFromSeries(t *testing.T) {
	data := []byte(`[
		{"participants":[{"roster":{"id":149001}},{"roster":{"id":139151}}]},
		{"participants":[{"roster":{"id":148648}},{"roster":{"id":149000}}]},
		{"participants":[]}
	]`)
	ids := extractRosterIDsFromSeries(data)
	if len(ids) != 4 {
		t.Errorf("want 4 unique roster IDs, got %d: %v", len(ids), ids)
	}
	seen := make(map[int]bool)
	for _, id := range ids {
		seen[id] = true
	}
	for _, want := range []int{149001, 139151, 148648, 149000} {
		if !seen[want] {
			t.Errorf("missing roster ID %d", want)
		}
	}
}

func TestExtractRosterIDsFromSeries_Empty(t *testing.T) {
	data := []byte(`[]`)
	ids := extractRosterIDsFromSeries(data)
	if len(ids) != 0 {
		t.Errorf("want 0 roster IDs, got %v", ids)
	}
}

func TestExtractTeamAndPlayerIDsFromRosters(t *testing.T) {
	data := []byte(`[
		{"team":{"id":100},"line_up":{"players":[{"id":1},{"id":2}]}},
		{"team":{"id":101},"line_up":{"players":[{"id":2},{"id":3}]}}
	]`)
	teamIDs, playerIDs := extractTeamAndPlayerIDsFromRosters(data)
	if len(teamIDs) != 2 {
		t.Errorf("want 2 team IDs, got %v", teamIDs)
	}
	if len(playerIDs) != 3 {
		t.Errorf("want 3 unique player IDs, got %v", playerIDs)
	}
}

func TestExtractTeamAndPlayerIDsFromRosters_Empty(t *testing.T) {
	data := []byte(`[]`)
	teamIDs, playerIDs := extractTeamAndPlayerIDsFromRosters(data)
	if len(teamIDs) != 0 {
		t.Errorf("want 0 team IDs, got %v", teamIDs)
	}
	if len(playerIDs) != 0 {
		t.Errorf("want 0 player IDs, got %v", playerIDs)
	}
}
