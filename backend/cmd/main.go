package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/cors"
	"github.com/samwang8891/whats-the-bot-doing/internal/api"
	ch "github.com/samwang8891/whats-the-bot-doing/internal/clickhouse"
	"github.com/samwang8891/whats-the-bot-doing/internal/config"
	"github.com/samwang8891/whats-the-bot-doing/internal/geoip"
	"github.com/samwang8891/whats-the-bot-doing/internal/ingestion"
)

func main() {
	cfg := config.Load()

	// Initialize GeoIP
	geo, err := geoip.New(cfg.GeoIPPath)
	if err != nil {
		log.Fatalf("Failed to load GeoIP database: %v", err)
	}
	defer geo.Close()
	log.Println("GeoIP database loaded")

	// Initialize ClickHouse
	chClient, err := ch.New(cfg.ClickHouseAddr, cfg.ClickHouseDB, cfg.BatchSize, cfg.FlushInterval)
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer chClient.Close()
	log.Println("ClickHouse connected")

	// Trap server — catches all bot requests
	trapHandler := ingestion.NewHandler(chClient, geo)
	trapServer := &http.Server{
		Addr:    ":" + cfg.TrapPort,
		Handler: trapHandler,
	}

	// API server — serves the dashboard
	apiServer := api.NewServer(chClient.Conn())
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(apiServer.Handler())

	dashServer := &http.Server{
		Addr:    ":" + cfg.APIPort,
		Handler: corsHandler,
	}

	// Start servers
	go func() {
		log.Printf("Trap server listening on :%s", cfg.TrapPort)
		if err := trapServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Trap server error: %v", err)
		}
	}()

	go func() {
		log.Printf("API server listening on :%s", cfg.APIPort)
		if err := dashServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("API server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	trapServer.Close()
	dashServer.Close()
	log.Println("Servers stopped")
}
