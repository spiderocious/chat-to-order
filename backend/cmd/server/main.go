// Command server boots the chat-to-order backend: config, Postgres, Redis,
// migrations, the domain, the MCP server, the reconciler and the HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spiderocious/chat-to-order/backend/internal/activity"
	"github.com/spiderocious/chat-to-order/backend/internal/config"
	"github.com/spiderocious/chat-to-order/backend/internal/httpapi"
	"github.com/spiderocious/chat-to-order/backend/internal/mcpserver"
	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
	"github.com/spiderocious/chat-to-order/backend/internal/paystack"
	"github.com/spiderocious/chat-to-order/backend/internal/reconcile"
	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("boot", "err", err)
		os.Exit(1)
	}
	var lvl slog.Level
	_ = lvl.UnmarshalText([]byte(strings.ToUpper(cfg.LogLevel)))
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("boot.db", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb := redisx.New(ctx, cfg.RedisURL, log)
	locks := redisx.NewLocker(rdb)

	// Migrate under a lock so two replicas booting together do not race goose.
	if rel, ok := redisx.AcquireWait(ctx, locks, "migrate", 2*time.Minute, 90*time.Second); ok {
		err = db.Migrate(ctx)
		rel()
	} else {
		err = db.Migrate(ctx)
	}
	if err != nil {
		log.Error("boot.migrate", "err", err)
		os.Exit(1)
	}

	// Admin bootstrap from env; the password is hashed in the store and never
	// stored in the repo. See store.BootstrapAdmin for the rotation rule.
	if err := db.BootstrapAdmin(ctx, cfg.AdminEmail, cfg.AdminName, cfg.AdminPassword); err != nil {
		log.Error("boot.admin", "err", err)
		os.Exit(1)
	}

	if cfg.SeedDemoOrders {
		if empty, err := db.OrdersEmpty(ctx); err == nil && empty {
			if err := db.SeedDemo(ctx); err != nil {
				log.Warn("boot.seed_demo", "err", err)
			} else {
				log.Info("boot.seed_demo", "orders", 6)
			}
		}
	}

	var pay paystack.Client
	var fake *paystack.Fake
	if cfg.PaystackFake {
		fake = paystack.NewFake(cfg.PublicBaseURL)
		pay = fake
		log.Warn("boot.paystack", "mode", "fake")
	} else {
		pay = paystack.NewHTTP(cfg.PaystackSecretKey)
	}

	cache := redisx.NewCache(rdb)
	bus := redisx.NewBus(ctx, rdb)
	svc := ordering.New(db, pay, cache, locks, bus, log, cfg.PublicBaseURL)
	svc.SetEmailDomain(cfg.OrderEmailDomain)
	metrics := httpapi.NewMetrics(func() float64 {
		if rdb == nil {
			return 0
		}
		return float64(rdb.Fallback.Load())
	})
	svc.SetHooks(ordering.Hooks{OnConfirmed: func(by, status string) { metrics.Confirmed.WithLabelValues(status, by).Inc() }})

	act := activity.New(db, log)
	mcpHandler := mcpserver.Handler(mcpserver.Deps{Svc: svc, Activity: act, Token: cfg.MCPBearerToken, Version: version,
		OnCall: func(tool string, ok bool, code string, d time.Duration) {
			metrics.ToolCalls.WithLabelValues(tool, boolStr(ok), code).Inc()
			metrics.ToolDur.WithLabelValues(tool).Observe(d.Seconds())
		}})

	rec := reconcile.New(db, svc, locks, log)
	go rec.Run(ctx)

	var ready atomic.Bool
	srv := httpapi.New(httpapi.Deps{Cfg: cfg, DB: db, Svc: svc, Rec: rec, Redis: rdb, Bus: bus, Fake: fake, Log: log,
		Version: version, Ready: ready.Load, Metrics: metrics, MCP: mcpHandler})

	hs := &http.Server{
		Addr: ":" + cfg.Port, Handler: srv.Router(),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 0, IdleTimeout: 120 * time.Second,
	}
	// warm the cache, then mark ready
	if _, err := svc.Snapshot(ctx); err != nil {
		log.Warn("boot.warm", "err", err)
	}
	ready.Store(true)
	log.Info("boot.listen", "port", cfg.Port, "version", version, "redis", rdb.Health(ctx), "paystack_fake", cfg.PaystackFake)

	go func() {
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	ready.Store(false)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = hs.Shutdown(shutdownCtx)
	log.Info("shutdown")
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
