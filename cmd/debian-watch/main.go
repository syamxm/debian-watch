package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/syamxm/debian-watch/internal/auth"
	"github.com/syamxm/debian-watch/internal/collect"
	"github.com/syamxm/debian-watch/internal/config"
	"github.com/syamxm/debian-watch/internal/docker"
	"github.com/syamxm/debian-watch/internal/httpx"
	"github.com/syamxm/debian-watch/internal/monitor"
)

const (
	loginMaxFailures = 5
	loginWindow      = 15 * time.Minute
	sweepInterval    = 5 * time.Minute
	shutdownTimeout  = 10 * time.Second
)

func main() {
	health := flag.Bool("health", false, "probe the local /healthz endpoint and exit")
	flag.Parse()

	if *health {
		if err := runHealthCheck(envAddr()); err != nil {
			fmt.Fprintln(os.Stderr, "health check failed:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "startup failed:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	if err := useHostFilesystem(cfg.HostRoot); err != nil {
		return err
	}

	creds, err := auth.NewCredentials(cfg.AdminUser, cfg.AdminPassHash)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sessions := auth.NewSessionStore(cfg.SessionTTL)
	limiter := auth.NewLoginLimiter(loginMaxFailures, loginWindow)
	go sessions.Sweep(ctx, sweepInterval)
	go limiter.Sweep(ctx, sweepInterval)

	dockerMonitor, err := docker.NewMonitor(cfg.DockerHost, cfg.DockerInterval, log)
	if err != nil {
		return err
	}
	if !dockerMonitor.Enabled() {
		log.Info("docker panel disabled", "reason", "DW_DOCKER_HOST is not set")
	}
	go dockerMonitor.Run(ctx)

	metrics := monitor.New(collect.New(ctx, log, cfg.HostRoot, cfg.NetDevPath), dockerMonitor, cfg.SampleInterval, cfg.HistorySize, log)
	go metrics.Run(ctx)

	server, err := httpx.NewServer(cfg, log, metrics, sessions, limiter, creds)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           http.TimeoutHandler(server.Handler(), 10*time.Second, "request timed out"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	listenErr := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", cfg.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErr <- err
		}
	}()

	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
