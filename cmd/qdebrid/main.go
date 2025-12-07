package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"qdebrid/internal/cache"
	"qdebrid/internal/config"
	"qdebrid/internal/debrid"
	"qdebrid/internal/logger"
	"qdebrid/internal/qbittorrent"
	"qdebrid/internal/servarr"
)

var (
	configPath    = flag.String("config", "", "Path to config file")
	exampleConfig = flag.Bool("example-config", false, "Print example configuration and exit")
	writeConfig   = flag.String("write-config", "", "Write example config to file and exit")
	version       = "2.0.0"
	buildTime     = "unknown"
)

func main() {
	flag.Parse()

	// Handle example config
	if *exampleConfig {
		fmt.Println(config.ExampleConfig())
		os.Exit(0)
	}

	// Handle write config
	if *writeConfig != "" {
		if err := config.WriteExample(*writeConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Example config written to: %s\n", *writeConfig)
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadFromEnv()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nUse -example-config to see configuration format\n")
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logging.Level, cfg.Logging.OutputPath, cfg.Logging.JSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("qDebrid starting",
		zap.String("version", version),
		zap.String("build_time", buildTime),
	)

	// Run application
	if err := run(cfg, log); err != nil {
		log.Fatal("application error", zap.Error(err))
	}

	log.Info("qDebrid stopped")
}

func run(cfg *config.Config, log *zap.Logger) error {
	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize cache
	cacheInstance := cache.New(5*time.Minute, log.Named("cache"))
	defer cacheInstance.Close()

	// Initialize Real-Debrid client
	debridClient := debrid.NewClient(&cfg.RealDebrid, log.Named("debrid"))
	defer debridClient.Shutdown()

	// Initialize Servarr client
	servarrClient := servarr.NewClient(log.Named("servarr"))

	// Initialize qBittorrent handler
	handler := qbittorrent.NewHandler(
		debridClient,
		servarrClient,
		cacheInstance,
		cfg,
		log.Named("handler"),
	)

	// Create HTTP server
	server := qbittorrent.NewServer(
		handler,
		cfg.Server.Host,
		cfg.Server.Port,
		log.Named("server"),
	)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-sigChan:
		log.Info("received signal", zap.String("signal", sig.String()))

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		log.Info("shutting down gracefully...")

		// Shutdown HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", zap.Error(err))
			return err
		}

		// Real-Debrid client and cache cleanup handled by defer statements
	}

	return nil
}
