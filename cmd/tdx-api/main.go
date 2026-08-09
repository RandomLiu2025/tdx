package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/injoyai/tdx/extend/httpserver"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Printf("tdx-api stopped with error: %v", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	options := []httpserver.Option{
		httpserver.WithAddr(cfg.addr),
		httpserver.WithPoolSize(cfg.poolSize),
		httpserver.WithDialTimeout(cfg.dialTimeout),
	}
	if len(cfg.hosts) > 0 {
		options = append(options, httpserver.WithHosts(cfg.hosts...))
	}
	if len(cfg.exHqHosts) > 0 {
		options = append(options,
			httpserver.WithExHqHosts(cfg.exHqHosts...),
			httpserver.WithExPoolSize(cfg.exPoolSize),
		)
	}

	apiServer, err := httpserver.Default(options...)
	if err != nil {
		return fmt.Errorf("create HTTP server: %w", err)
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- apiServer.Run()
	}()

	log.Printf("tdx-api listening on %s", cfg.addr)
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("run HTTP server: %w", err)
	case <-ctx.Done():
		log.Printf("tdx-api shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("run HTTP server: %w", err)
	}
	return nil
}
