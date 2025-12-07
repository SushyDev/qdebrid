package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"qdebrid/internal/cache"
	"qdebrid/internal/config"
	"qdebrid/internal/debrid"
	"qdebrid/internal/history"
	"qdebrid/internal/logger"
	"qdebrid/internal/qbittorrent"
	"qdebrid/internal/servarr"
)

var (
	configPath    = flag.String("config", "", "Path to config file")
	exampleConfig = flag.Bool("example-config", false, "Print example configuration and exit")
	writeConfig   = flag.String("write-config", "", "Write example config to file and exit")
	healthCheck   = flag.Bool("health-check", false, "Perform health check and exit")
	version       = "2.0.0"
	buildTime     = "unknown"
)

func main() {
	flag.Parse()

	// Handle health check
	if *healthCheck {
		if err := performHealthCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OK")
		os.Exit(0)
	}

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

	// Initialize history store
	historyStore, err := history.NewStore(cfg.Data.Directory, log.Named("history"))
	if err != nil {
		return fmt.Errorf("failed to create history store: %w", err)
	}
	log.Info("history store initialized", zap.Int("records", historyStore.Count()))

	// Initialize Real-Debrid client
	debridClient := debrid.NewClient(&cfg.RealDebrid, log.Named("debrid"))
	defer debridClient.Shutdown()

	// Initialize Servarr client
	servarrClient := servarr.NewClient(log.Named("servarr"))

	// Initialize qBittorrent handler
	handler := qbittorrent.NewHandler(
		debridClient,
		servarrClient,
		historyStore,
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

	// Start periodic history save and cleanup
	autoSaveInterval, _ := time.ParseDuration(cfg.Data.AutoSaveInterval)
	cleanupMaxAge, _ := time.ParseDuration(cfg.Data.CleanupMaxAge)
	if autoSaveInterval == 0 {
		autoSaveInterval = 5 * time.Minute
	}
	if cleanupMaxAge == 0 {
		cleanupMaxAge = 168 * time.Hour // 7 days
	}

	stopHistory := make(chan struct{})
	historyDone := make(chan struct{})
	go func() {
		defer close(historyDone)
		saveTicker := time.NewTicker(autoSaveInterval)
		cleanupTicker := time.NewTicker(1 * time.Hour) // Cleanup every hour
		defer saveTicker.Stop()
		defer cleanupTicker.Stop()

		for {
			select {
			case <-stopHistory:
				// Final save on shutdown
				if err := historyStore.Save(); err != nil {
					log.Error("failed to save history on shutdown", zap.Error(err))
				} else {
					log.Info("history saved on shutdown")
				}
				return
			case <-saveTicker.C:
				if err := historyStore.Save(); err != nil {
					log.Error("failed to auto-save history", zap.Error(err))
				} else {
					log.Debug("history auto-saved")
				}
			case <-cleanupTicker.C:
				removed := historyStore.Cleanup(cleanupMaxAge)
				if removed > 0 {
					log.Info("cleaned up old history entries", zap.Int("removed", removed))
				}
			}
		}
	}()

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

		// Stop history saver
		close(stopHistory)

		// Wait for history to finish with timeout
		select {
		case <-historyDone:
			log.Info("history saver stopped")
		case <-time.After(5 * time.Second):
			log.Warn("history saver shutdown timeout")
		}

		// Shutdown HTTP server
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", zap.Error(err))
			return err
		}

		// Real-Debrid client and cache cleanup handled by defer statements
	}

	return nil
}

func performHealthCheck() error {
	// Load configuration to get server port
	var cfg *config.Config
	var err error

	if *configPath != "" {
		cfg, err = config.Load(*configPath)
	} else {
		cfg, err = config.LoadFromEnv()
	}

	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Always use 127.0.0.1 for health check (works inside container)
	url := fmt.Sprintf("http://127.0.0.1:%d/health", cfg.Server.Port)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to health endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned status %d", resp.StatusCode)
	}

	return nil
}
