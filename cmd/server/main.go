package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aaron/gamehub/internal/atlas"
	"github.com/aaron/gamehub/internal/config"
	"github.com/aaron/gamehub/internal/handlers"
	"github.com/aaron/gamehub/internal/live"
	"github.com/aaron/gamehub/internal/middleware"
)

func main() {
	secret := os.Getenv("ATLAS_API_KEY")
	if secret == "" {
		log.Fatal("ATLAS_API_KEY must be set")
	}

	client := atlas.NewClient(secret)
	liveSvc := live.NewService(client, config.LiveCacheTTL)
	h := handlers.New(client, liveSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /series/live", h.SeriesLive)
	mux.HandleFunc("GET /players/live", h.PlayersLive)
	mux.HandleFunc("GET /teams/live", h.TeamsLive)

	limiter := middleware.NewLimiter(config.InboundRateLimitRequests, config.InboundRateLimitPer)
	handler := limiter.Middleware(mux)

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
