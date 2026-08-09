package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr        = ":8080"
	defaultPoolSize        = 1
	defaultExPoolSize      = 1
	defaultDialTimeout     = 5 * time.Second
	defaultShutdownTimeout = 10 * time.Second
)

type config struct {
	addr            string
	hosts           []string
	poolSize        int
	exHqHosts       []string
	exPoolSize      int
	dialTimeout     time.Duration
	shutdownTimeout time.Duration
}

type envLookup func(string) (string, bool)

func loadConfig() (config, error) {
	return loadConfigFromEnv(os.LookupEnv)
}

func loadConfigFromEnv(lookup envLookup) (config, error) {
	cfg := config{
		addr:            defaultHTTPAddr,
		poolSize:        defaultPoolSize,
		exPoolSize:      defaultExPoolSize,
		dialTimeout:     defaultDialTimeout,
		shutdownTimeout: defaultShutdownTimeout,
	}

	if value, ok := lookup("TDX_HTTP_ADDR"); ok {
		cfg.addr = strings.TrimSpace(value)
		if cfg.addr == "" {
			return config{}, fmt.Errorf("TDX_HTTP_ADDR must not be empty")
		}
	}
	if value, ok := lookup("TDX_HOSTS"); ok {
		cfg.hosts = splitList(value)
	}
	if value, ok := lookup("TDX_EXHQ_HOSTS"); ok {
		cfg.exHqHosts = splitList(value)
	}

	var err error
	cfg.poolSize, err = positiveInt(lookup, "TDX_POOL_SIZE", defaultPoolSize)
	if err != nil {
		return config{}, err
	}
	cfg.exPoolSize, err = positiveInt(lookup, "TDX_EXHQ_POOL_SIZE", defaultExPoolSize)
	if err != nil {
		return config{}, err
	}
	cfg.dialTimeout, err = positiveDuration(lookup, "TDX_DIAL_TIMEOUT", defaultDialTimeout)
	if err != nil {
		return config{}, err
	}
	cfg.shutdownTimeout, err = positiveDuration(lookup, "TDX_SHUTDOWN_TIMEOUT", defaultShutdownTimeout)
	if err != nil {
		return config{}, err
	}

	return cfg, nil
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func positiveInt(lookup envLookup, key string, fallback int) (int, error) {
	value, ok := lookup(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return parsed, nil
}

func positiveDuration(lookup envLookup, key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}
