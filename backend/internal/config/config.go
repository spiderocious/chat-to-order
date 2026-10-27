// Package config loads the service configuration from the environment.
// Boot fails on any missing required key, on purpose: a half-configured
// service that starts is worse than one that refuses to.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port          string
	DatabaseURL   string
	RedisURL      string // optional; empty means degraded mode from boot
	PublicBaseURL string
	CORSOrigins   []string

	PaystackSecretKey string
	PaystackFake      bool

	MCPBearerToken string
	JWTSecret      string
	MetricsToken   string

	AdminEmail    string
	AdminPassword string // bootstrap only
	AdminName     string

	SeedDemoOrders bool
	LogLevel       string
	Release        bool
}

func Load() (Config, error) {
	c := Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		PublicBaseURL:  strings.TrimRight(getenv("PUBLIC_BASE_URL", "http://localhost:8080"), "/"),
		CORSOrigins:    splitList(getenv("CORS_ORIGINS", "http://localhost:5173")),
		PaystackSecretKey: os.Getenv("PAYSTACK_SECRET_KEY"),
		PaystackFake:   boolenv("PAYSTACK_FAKE"),
		MCPBearerToken: os.Getenv("MCP_BEARER_TOKEN"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
		AdminEmail:     os.Getenv("ADMIN_EMAIL"),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		AdminName:      getenv("ADMIN_NAME", "Iya Bisi"),
		SeedDemoOrders: boolenv("SEED_DEMO_ORDERS"),
		LogLevel:       getenv("LOG_LEVEL", "info"),
		Release:        os.Getenv("GIN_MODE") == "release",
	}

	var missing []string
	for k, v := range map[string]string{
		"DATABASE_URL":     c.DatabaseURL,
		"MCP_BEARER_TOKEN": c.MCPBearerToken,
		"JWT_SECRET":       c.JWTSecret,
		"ADMIN_EMAIL":      c.AdminEmail,
	} {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if !c.PaystackFake && c.PaystackSecretKey == "" {
		missing = append(missing, "PAYSTACK_SECRET_KEY (or PAYSTACK_FAKE=1)")
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("config: missing %s", strings.Join(missing, ", "))
	}
	if c.Release && c.PaystackFake {
		return c, errors.New("config: PAYSTACK_FAKE is not allowed in release mode")
	}
	if len(c.JWTSecret) < 32 {
		return c, errors.New("config: JWT_SECRET must be at least 32 characters")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func boolenv(k string) bool {
	v, _ := strconv.ParseBool(os.Getenv(k))
	return v
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
