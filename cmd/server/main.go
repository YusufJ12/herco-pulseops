package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yusufjaelani/pulseops/internal/config"
	"github.com/yusufjaelani/pulseops/internal/handler"
	"github.com/yusufjaelani/pulseops/internal/prober"
	"github.com/yusufjaelani/pulseops/internal/store"
	"github.com/yusufjaelani/pulseops/web"
)

func main() {
	// Structured JSON logging for Cloud/DevOps observability
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Load()
	slog.Info("Starting PulseOps daemon",
		"port", cfg.Port,
		"db_path", cfg.DBPath,
		"interval", cfg.ProbeInterval.String(),
	)

	// Database store initialization
	db, err := store.New(cfg.DBPath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Seed default target if empty
	seedDefaultTarget(db, cfg)

	// Prober daemon initialization
	prb := prober.New(db, cfg.ProbeInterval)
	prb.Start()
	defer prb.Stop()

	appHandler := handler.New(db, prb, web.FS)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      appHandler.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown channel
	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info(fmt.Sprintf("PulseOps server listening on http://0.0.0.0:%s", cfg.Port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server listen failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stopSig
	slog.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced shutdown", "error", err)
	}

	slog.Info("PulseOps stopped cleanly")
}

func seedDefaultTarget(s *store.Store, cfg *config.Config) {
	targets, err := s.ListTargets()
	if err != nil {
		slog.Warn("Could not check targets for seeding", "error", err)
		return
	}
	if len(targets) == 0 && cfg.DefaultTargetURL != "" {
		slog.Info("Seeding initial monitor target", "name", cfg.DefaultTargetName, "url", cfg.DefaultTargetURL)
		_, err := s.AddTarget(cfg.DefaultTargetName, cfg.DefaultTargetURL)
		if err != nil {
			slog.Warn("Failed seeding default target", "error", err)
		}
	}
}
