package atlas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestGetAllPages_ContinuesPastFullPage(t *testing.T) {
	// Simulate Atlas API: first page returns 50 (full), second returns 15.
	// Per pagination doc we must request skip=50 and stop when len < take.
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		skip := 0
		if s := r.URL.Query().Get("skip"); s != "" {
			if n, err := parseInt(s); err == nil {
				skip = n
			}
		}
		take := 50
		if s := r.URL.Query().Get("take"); s != "" {
			if n, err := parseInt(s); err == nil {
				take = n
			}
		}
		// Total 65 items: skip 0 -> 50, skip 50 -> 15
		start := skip
		end := skip + take
		if end > 65 {
			end = 65
		}
		var items []map[string]interface{}
		for i := start; i < end; i++ {
			items = append(items, map[string]interface{}{"id": i + 1})
		}
		if err := json.NewEncoder(w).Encode(items); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	client := NewClientWithURL("test-secret", server.URL)
	body, _, err := client.GetPlayersAll(context.Background(), map[string]string{"filter": "id<={1}"})
	if err != nil {
		t.Fatal(err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if len(result) != 65 {
		t.Errorf("got %d items, want 65 (pagination truncated)", len(result))
	}
	if requestCount != 2 {
		t.Errorf("got %d requests, want 2 (second request needed when first returns exactly 50)", requestCount)
	}
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func TestGet_OutboundRateLimit(t *testing.T) {
	// Server returns 429 with Retry-After: 50ms.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "50")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	client := NewClientWithURL("test-secret", server.URL)
	ctx := context.Background()

	// First request: gets 429, sets backoff
	_, _, err := client.Get(ctx, "/test")
	if err == nil {
		t.Fatal("want ErrRateLimited, got nil")
	}
	if _, ok := err.(*ErrRateLimited); !ok {
		t.Fatalf("want ErrRateLimited, got %T", err)
	}

	// Second request: must wait ~50ms (backoff) before sending
	start := time.Now()
	_, _, err = client.Get(ctx, "/test")
	if err == nil {
		t.Fatal("want ErrRateLimited, got nil")
	}
	elapsed := time.Since(start)

	// Allow 5ms tolerance
	if elapsed < 45*time.Millisecond {
		t.Errorf("outbound backoff: elapsed %v, want >= 45ms (should wait Retry-After)", elapsed)
	}
}

func TestFilterIDIn(t *testing.T) {
	tests := []struct {
		ids  []int
		want string
	}{
		{[]int{1, 2, 3}, "id<={1,2,3}"},
		{[]int{149001}, "id<={149001}"},
		{[]int{}, ""},
	}
	for _, tt := range tests {
		got := FilterIDIn(tt.ids)
		if got != tt.want {
			t.Errorf("FilterIDIn(%v) = %q, want %q", tt.ids, got, tt.want)
		}
	}
}
