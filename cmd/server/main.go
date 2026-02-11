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
	"github.com/aaron/gamehub/internal/metrics"
	"github.com/aaron/gamehub/internal/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	secret := os.Getenv("ATLAS_API_KEY")
	if secret == "" {
		log.Fatal("ATLAS_API_KEY must be set")
	}

	client := atlas.NewClient(secret)
	liveSvc := live.NewService(client, config.LiveCacheTTL())
	h := handlers.New(client, liveSvc)

	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Apply metrics middleware globally
	router.Use(metrics.Middleware())

	// Health and monitoring endpoints (no rate limiting)
	router.GET("/health", handlers.Health)
	router.GET("/monitor", metrics.ServeMonitor)
	router.GET("/stats", metrics.ServeJSON)

	// API endpoints with rate limiting
	limiter := middleware.NewLimiter(config.InboundRateLimitRequests(), config.InboundRateLimitPer())
	api := router.Group("/")
	api.Use(limiter.Middleware())
	{
		api.GET("/series/live", h.SeriesLive)
		api.GET("/players/live", h.PlayersLive)
		api.GET("/teams/live", h.TeamsLive)
	}

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	srv := &http.Server{Addr: addr, Handler: router}

	// Determine the host for the monitor URL
	host := "localhost"
	if port := os.Getenv("PORT"); port != "" {
		host = host + ":" + port
	} else {
		host = host + ":8080"
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Brief delay to ensure server starts listening before showing message
	time.Sleep(100 * time.Millisecond)
	log.Printf("Server started on %s", addr)
	log.Printf("Monitor dashboard: http://%s/monitor", host)

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
