package main

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfigDefaults(test *testing.T) {
	cfg, err := loadConfigFromEnv(func(string) (string, bool) { return "", false })
	if err != nil {
		test.Fatalf("loadConfigFromEnv() error = %v", err)
	}

	if cfg.addr != ":18080" {
		test.Errorf("addr = %q, want %q", cfg.addr, ":18080")
	}
	if cfg.hosts != nil {
		test.Errorf("hosts = %#v, want nil", cfg.hosts)
	}
	if cfg.poolSize != 1 {
		test.Errorf("poolSize = %d, want 1", cfg.poolSize)
	}
	if cfg.exHqHosts != nil {
		test.Errorf("exHqHosts = %#v, want nil", cfg.exHqHosts)
	}
	if cfg.exPoolSize != 1 {
		test.Errorf("exPoolSize = %d, want 1", cfg.exPoolSize)
	}
	if cfg.dialTimeout != 5*time.Second {
		test.Errorf("dialTimeout = %s, want 5s", cfg.dialTimeout)
	}
	if cfg.shutdownTimeout != 10*time.Second {
		test.Errorf("shutdownTimeout = %s, want 10s", cfg.shutdownTimeout)
	}
}

func TestLoadConfigFromEnv(test *testing.T) {
	env := map[string]string{
		"TDX_HTTP_ADDR":        " 0.0.0.0:9090 ",
		"TDX_HOSTS":            " 127.0.0.1:7709, ,10.0.0.1 ",
		"TDX_POOL_SIZE":        "3",
		"TDX_EXHQ_HOSTS":       " 127.0.0.1:7727,10.0.0.2 ",
		"TDX_EXHQ_POOL_SIZE":   "2",
		"TDX_DIAL_TIMEOUT":     "3s",
		"TDX_SHUTDOWN_TIMEOUT": "15s",
	}

	cfg, err := loadConfigFromEnv(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		test.Fatalf("loadConfigFromEnv() error = %v", err)
	}

	if cfg.addr != "0.0.0.0:9090" {
		test.Errorf("addr = %q, want %q", cfg.addr, "0.0.0.0:9090")
	}
	assertStrings(test, cfg.hosts, []string{"127.0.0.1:7709", "10.0.0.1"})
	if cfg.poolSize != 3 {
		test.Errorf("poolSize = %d, want 3", cfg.poolSize)
	}
	assertStrings(test, cfg.exHqHosts, []string{"127.0.0.1:7727", "10.0.0.2"})
	if cfg.exPoolSize != 2 {
		test.Errorf("exPoolSize = %d, want 2", cfg.exPoolSize)
	}
	if cfg.dialTimeout != 3*time.Second {
		test.Errorf("dialTimeout = %s, want 3s", cfg.dialTimeout)
	}
	if cfg.shutdownTimeout != 15*time.Second {
		test.Errorf("shutdownTimeout = %s, want 15s", cfg.shutdownTimeout)
	}
}

func TestLoadConfigRejectsInvalidValues(test *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		wantError string
	}{
		{name: "empty address", key: "TDX_HTTP_ADDR", value: " ", wantError: "TDX_HTTP_ADDR"},
		{name: "invalid pool size", key: "TDX_POOL_SIZE", value: "many", wantError: "TDX_POOL_SIZE"},
		{name: "zero pool size", key: "TDX_POOL_SIZE", value: "0", wantError: "TDX_POOL_SIZE"},
		{name: "negative extended pool size", key: "TDX_EXHQ_POOL_SIZE", value: "-1", wantError: "TDX_EXHQ_POOL_SIZE"},
		{name: "invalid dial timeout", key: "TDX_DIAL_TIMEOUT", value: "later", wantError: "TDX_DIAL_TIMEOUT"},
		{name: "zero dial timeout", key: "TDX_DIAL_TIMEOUT", value: "0s", wantError: "TDX_DIAL_TIMEOUT"},
		{name: "invalid timeout", key: "TDX_SHUTDOWN_TIMEOUT", value: "later", wantError: "TDX_SHUTDOWN_TIMEOUT"},
		{name: "zero timeout", key: "TDX_SHUTDOWN_TIMEOUT", value: "0s", wantError: "TDX_SHUTDOWN_TIMEOUT"},
	}

	for _, testCase := range tests {
		test.Run(testCase.name, func(test *testing.T) {
			_, err := loadConfigFromEnv(func(key string) (string, bool) {
				if key == testCase.key {
					return testCase.value, true
				}
				return "", false
			})
			if err == nil {
				test.Fatal("loadConfigFromEnv() error = nil")
			}
			if !strings.Contains(err.Error(), testCase.wantError) {
				test.Errorf("error = %q, want it to contain %q", err, testCase.wantError)
			}
		})
	}
}

func assertStrings(test *testing.T, got, want []string) {
	test.Helper()
	if len(got) != len(want) {
		test.Fatalf("values = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			test.Errorf("values[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
