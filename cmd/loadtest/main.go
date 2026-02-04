// Load test: hammer the server to trigger inbound rate limit (429).
//
//	go run ./cmd/loadtest
//	go run ./cmd/loadtest -url http://localhost:8080 -n 80
package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "Base URL of the server")
	n := flag.Int("n", 80, "Number of requests to send")
	delay := flag.Duration("delay", 50*time.Millisecond, "Delay between requests")
	flag.Parse()

	base := *url + "/series/live"
	fmt.Printf("Load test: %d requests to %s\n", *n, base)
	fmt.Println("Expect: first ~60 succeed (200), rest get 429")
	fmt.Println()

	var ok, rateLimited, other int
	client := &http.Client{Timeout: 10 * time.Second}

	for i := 0; i < *n; i++ {
		resp, err := client.Get(base)
		if err != nil {
			fmt.Printf("Request %d: %v\n", i+1, err)
			other++
			time.Sleep(*delay)
			continue
		}
		switch resp.StatusCode {
		case http.StatusOK:
			ok++
		case http.StatusTooManyRequests:
			rateLimited++
		default:
			other++
			fmt.Printf("Request %d: HTTP %d\n", i+1, resp.StatusCode)
		}
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("close response body: %v\n", err)
		}
		time.Sleep(*delay)
	}

	fmt.Println()
	fmt.Println("Results:")
	fmt.Printf("  200 OK: %d\n", ok)
	fmt.Printf("  429 Rate Limited: %d\n", rateLimited)
	fmt.Printf("  Other: %d\n", other)
	fmt.Println()

	if rateLimited > 0 {
		fmt.Println("Rate limiting is working.")
	} else {
		fmt.Println("No 429s observed. Inbound limit may be higher than requests sent or server not running.")
	}
}
