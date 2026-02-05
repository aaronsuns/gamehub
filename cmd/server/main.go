package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	liveSvc := live.NewService(client, config.LiveCacheTTL())
	h := handlers.New(client, liveSvc)

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /series/live", h.SeriesLive)
	apiMux.HandleFunc("GET /players/live", h.PlayersLive)
	apiMux.HandleFunc("GET /teams/live", h.TeamsLive)

	limiter := middleware.NewLimiter(config.InboundRateLimitRequests(), config.InboundRateLimitPer())
	mainMux := http.NewServeMux()
	mainMux.HandleFunc("GET /health", handlers.Health)
	mainMux.Handle("/", limiter.Middleware(apiMux))

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	srv := &http.Server{Addr: addr, Handler: mainMux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Printf("Server stopped")
}
