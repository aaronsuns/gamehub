// Stress test tool for GameHub API
// Configurable via environment variables or command-line flags
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type config struct {
	url         string
	path        string
	requests    int
	duration    time.Duration
	concurrency int
	delay       time.Duration
	timeout     time.Duration
	verbose     bool
	showProgress bool
}

type stats struct {
	ok          int64
	rateLimited int64
	errors      int64
}

func main() {
	cfg := parseConfig()
	
	separator := strings.Repeat("=", 60)
	fmt.Println(separator)
	fmt.Println("GameHub Stress Test")
	fmt.Println(separator)
	fmt.Printf("URL:        %s%s\n", cfg.url, cfg.path)
	if cfg.duration > 0 {
		fmt.Printf("Duration:   %v\n", cfg.duration)
		fmt.Printf("Mode:       Continuous\n")
	} else {
		fmt.Printf("Requests:   %d\n", cfg.requests)
		fmt.Printf("Mode:       Fixed count\n")
	}
	fmt.Printf("Concurrency: %d\n", cfg.concurrency)
	fmt.Printf("Delay:      %v\n", cfg.delay)
	fmt.Printf("Timeout:    %v\n", cfg.timeout)
	fmt.Println(separator)
	fmt.Println()

	start := time.Now()
	results := runStressTest(cfg)
	duration := time.Since(start)

	printResults(results, duration, cfg)
}

func parseConfig() *config {
	cfg := &config{}

	flag.StringVar(&cfg.url, "url", getEnv("STRESS_URL", "http://localhost:8080"), "Base URL")
	flag.StringVar(&cfg.path, "path", getEnv("STRESS_PATH", "/players/live"), "API path to test")
	flag.IntVar(&cfg.requests, "n", getEnvInt("STRESS_N", 0), "Total requests (0 = use duration)")
	flag.DurationVar(&cfg.duration, "duration", getEnvDuration("STRESS_DURATION", 0), "Duration to run (e.g., 5m, 0 = use request count)")
	flag.IntVar(&cfg.concurrency, "c", getEnvInt("STRESS_CONCURRENCY", 10), "Concurrent workers")
	flag.DurationVar(&cfg.delay, "delay", getEnvDuration("STRESS_DELAY", 50*time.Millisecond), "Delay between requests")
	flag.DurationVar(&cfg.timeout, "timeout", getEnvDuration("STRESS_TIMEOUT", 30*time.Second), "HTTP client timeout")
	flag.BoolVar(&cfg.verbose, "v", getEnvBool("STRESS_VERBOSE", false), "Verbose output")
	flag.BoolVar(&cfg.showProgress, "progress", getEnvBool("STRESS_PROGRESS", true), "Show progress updates")
	flag.Parse()

	// Default to 5 minutes if neither duration nor requests specified
	if cfg.duration == 0 && cfg.requests == 0 {
		cfg.duration = 5 * time.Minute
	}

	return cfg
}

func runStressTest(cfg *config) *stats {
	stats := &stats{}
	client := &http.Client{Timeout: cfg.timeout}
	targetURL := cfg.url + cfg.path

	ctx := make(chan struct{})
	var wg sync.WaitGroup

	// Progress reporter
	if cfg.showProgress && cfg.duration > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			start := time.Now()
			for {
				select {
				case <-ctx:
					return
				case <-ticker.C:
					elapsed := time.Since(start)
					total := atomic.LoadInt64(&stats.ok) + atomic.LoadInt64(&stats.rateLimited) + atomic.LoadInt64(&stats.errors)
					ok := atomic.LoadInt64(&stats.ok)
					rateLimited := atomic.LoadInt64(&stats.rateLimited)
					errors := atomic.LoadInt64(&stats.errors)
					fmt.Printf("[%v] Total: %d | OK: %d | 429: %d | Errors: %d\n", 
						elapsed.Round(time.Second), total, ok, rateLimited, errors)
				}
			}
		}()
	}

	// Worker pool
	for i := 0; i < cfg.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			requestCount := 0
			startTime := time.Now()
			
			for {
				// Check if we should stop
				if cfg.duration > 0 {
					if time.Since(startTime) >= cfg.duration {
						return
					}
				} else {
					if requestCount >= cfg.requests {
						return
					}
				}

				reqStart := time.Now()
				resp, err := client.Get(targetURL)
				reqDuration := time.Since(reqStart)

				if err != nil {
					atomic.AddInt64(&stats.errors, 1)
					if cfg.verbose {
						fmt.Printf("Error: %v\n", err)
					}
				} else {
					switch resp.StatusCode {
					case http.StatusOK:
						atomic.AddInt64(&stats.ok, 1)
					case http.StatusTooManyRequests:
						atomic.AddInt64(&stats.rateLimited, 1)
					default:
						atomic.AddInt64(&stats.errors, 1)
						if cfg.verbose {
							fmt.Printf("HTTP %d\n", resp.StatusCode)
						}
					}
					if cfg.verbose {
						fmt.Printf("[%d] %s %d (%v)\n", 
							atomic.LoadInt64(&stats.ok)+atomic.LoadInt64(&stats.rateLimited)+atomic.LoadInt64(&stats.errors),
							resp.Status, resp.StatusCode, reqDuration)
					}
					if err := resp.Body.Close(); err != nil {
						// Log but don't fail on body close errors
						if cfg.verbose {
							fmt.Printf("Error closing response body: %v\n", err)
						}
					}
				}

				requestCount++
				time.Sleep(cfg.delay)
			}
		}()
	}

	// Wait for duration or all requests
	if cfg.duration > 0 {
		time.Sleep(cfg.duration)
		close(ctx) // Signal progress reporter to stop
	}

	wg.Wait()
	return stats
}

func printResults(stats *stats, duration time.Duration, cfg *config) {
	total := stats.ok + stats.rateLimited + stats.errors
	rate := float64(total) / duration.Seconds()

	fmt.Println()
	separator := strings.Repeat("=", 60)
	fmt.Println(separator)
	fmt.Println("Results")
	fmt.Println(separator)
	fmt.Printf("Duration:   %v\n", duration.Round(time.Millisecond))
	fmt.Printf("Total:      %d requests\n", total)
	fmt.Printf("Rate:       %.2f req/s\n", rate)
	fmt.Println()
	fmt.Printf("200 OK:     %d (%.1f%%)\n", stats.ok, percent(stats.ok, total))
	fmt.Printf("429 Rate Limited: %d (%.1f%%)\n", stats.rateLimited, percent(stats.rateLimited, total))
	fmt.Printf("Errors:     %d (%.1f%%)\n", stats.errors, percent(stats.errors, total))
	fmt.Println(separator)
	fmt.Println()

	if stats.rateLimited > 0 {
		fmt.Println("✓ Rate limiting is working!")
	} else if stats.ok == total {
		fmt.Println("⚠ No rate limiting observed. All requests succeeded.")
		fmt.Println("  Try increasing requests (-n) or reducing delay (-delay)")
	}
}

func percent(n, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "1" || v == "true" || v == "yes"
	}
	return defaultValue
}
