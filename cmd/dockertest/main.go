// dockertest hits the live endpoints of a running server and prints a summary.
// Used by "make docker-test" after starting the container.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const baseURL = "http://localhost:8080"
const maxShow = 8

func itemName(path string, m map[string]interface{}) string {
	switch {
	case strings.Contains(path, "series"):
		if s, ok := m["title"].(string); ok && s != "" {
			return s
		}
	case strings.Contains(path, "players"):
		if s, ok := m["nick_name"].(string); ok && s != "" {
			return s
		}
		if f, _ := m["first_name"].(string); f != "" {
			if l, _ := m["last_name"].(string); l != "" {
				return f + " " + l
			}
			return f
		}
	case strings.Contains(path, "teams"):
		if s, ok := m["name"].(string); ok && s != "" {
			return s
		}
	}
	if id, ok := m["id"].(float64); ok {
		return fmt.Sprintf("id:%.0f", id)
	}
	return "?"
}

func fetchAndSummarize(path string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(baseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: want 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		return fmt.Errorf("Content-Type: want application/json, got %s", ct)
	}

	var arr []json.RawMessage
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&arr); err != nil {
		return fmt.Errorf("invalid JSON array: %w", err)
	}

	fmt.Printf("  %d items\n", len(arr))
	var names []string
	for i, raw := range arr {
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		name := itemName(path, m)
		if i < maxShow {
			names = append(names, name)
		}
	}
	if len(names) > 0 {
		summary := strings.Join(names, ", ")
		if len(arr) > maxShow {
			summary += fmt.Sprintf(" ... (+%d more)", len(arr)-maxShow)
		}
		fmt.Printf("  → %s\n", summary)
	}
	return nil
}

func main() {
	endpoints := []string{"/series/live", "/players/live", "/teams/live"}
	var failed bool
	for i, path := range endpoints {
		if i > 0 {
			time.Sleep(time.Second) // avoid inbound rate limit
		}
		fmt.Printf("%s\n", path)
		if err := fetchAndSummarize(path); err != nil {
			fmt.Printf("  ERROR: %v\n", err)
			failed = true
		}
		fmt.Println()
	}
	if failed {
		os.Exit(1)
	}
	fmt.Println("All 3 endpoints OK")
}
