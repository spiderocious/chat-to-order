// Package httpapi is the Gin transport: REST + SSE for the admin panel, the
// MCP mount, the paid page, health and metrics.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/spiderocious/chat-to-order/backend/internal/config"
	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
	"github.com/spiderocious/chat-to-order/backend/internal/paystack"
	"github.com/spiderocious/chat-to-order/backend/internal/reconcile"
	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

type Server struct {
	cfg     config.Config
	db      *store.DB
	svc     *ordering.Service
	rec     *reconcile.Reconciler
	redis   *redisx.Client
	limiter redisx.Limiter
	idem    redisx.Idempotency
	auth    authDeps
	bus     redisx.Bus
	fake    *paystack.Fake // nil in production
	log     *slog.Logger
	version string
	started time.Time
	ready   func() bool
	metrics *Metrics
	mcp     http.Handler
}

type Deps struct {
	Cfg     config.Config
	DB      *store.DB
	Svc     *ordering.Service
	Rec     *reconcile.Reconciler
	Redis   *redisx.Client
	Bus     redisx.Bus
	Fake    *paystack.Fake
	Log     *slog.Logger
	Version string
	Ready   func() bool
	Metrics *Metrics
	MCP     http.Handler
}

func New(d Deps) *Server {
	return &Server{
		cfg: d.Cfg, db: d.DB, svc: d.Svc, rec: d.Rec, redis: d.Redis, bus: d.Bus, fake: d.Fake, log: d.Log,
		version: d.Version, started: time.Now(), ready: d.Ready, metrics: d.Metrics, mcp: d.MCP,
		limiter: redisx.NewLimiter(d.Redis), idem: redisx.NewIdempotency(d.Redis),
		auth: authDeps{secret: []byte(d.Cfg.JWTSecret), revoked: redisx.NewRevoked(d.Redis)},
	}
}

func (s *Server) Router() *gin.Engine {
	if s.cfg.Release {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), requestID(), securityHeaders(), accessLog(s.log), cors(s.cfg.CORSOrigins), s.metrics.middleware())
	_ = r.SetTrustedProxies(nil)

	r.GET("/healthz", s.health)
	r.GET("/metrics", s.metricsHandler())
	r.GET("/paid", rateLimit(s.limiter, "paid", 60, time.Minute, byIP), s.paidPage)
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/healthz") })

	// MCP: stateless streamable HTTP; GET/DELETE are 405 by design.
	mcpGroup := r.Group("/mcp", bodyLimit(64<<10), rateLimit(s.limiter, "mcp", 120, time.Minute, func(*gin.Context) string { return "all" }))
	mcpGroup.POST("", gin.WrapH(s.mcp))
	mcpGroup.GET("", func(c *gin.Context) { c.AbortWithStatus(http.StatusMethodNotAllowed) })
	mcpGroup.DELETE("", func(c *gin.Context) { c.AbortWithStatus(http.StatusMethodNotAllowed) })

	api := r.Group("/api/v1", bodyLimit(256<<10))
	api.POST("/auth/login", rateLimit(s.limiter, "login", 10, time.Minute, byIP), s.login)

	admin := api.Group("", s.requireJWT(), rateLimit(s.limiter, "admin", 300, time.Minute, func(c *gin.Context) string { return c.GetString("admin_id") }))
	admin.POST("/auth/logout", s.logout)
	admin.GET("/auth/me", s.me)

	admin.GET("/vendor", s.getVendor)
	admin.PUT("/vendor", s.idempotent(), s.putVendor)

	admin.GET("/menu", s.listMenu)
	admin.POST("/menu", s.idempotent(), s.createItem)
	admin.PATCH("/menu/:id", s.idempotent(), s.patchItem)

	admin.GET("/zones", s.listZones)
	admin.POST("/zones", s.idempotent(), s.createZone)
	admin.PATCH("/zones/:id", s.idempotent(), s.patchZone)

	// Literal sub-paths are registered before the parameterised detail so
	// /orders/IBK-0231/resolve is never swallowed by /orders/:order_no.
	admin.POST("/orders/:order_no/resolve", s.idempotent(), s.resolveOrder)
	admin.POST("/orders/:order_no/cancel", s.idempotent(), s.cancelOrder)
	admin.GET("/orders", s.listOrders)
	admin.GET("/orders/:order_no", s.getOrder)

	admin.GET("/stats/today", s.statsToday)
	admin.GET("/activity", s.listActivity)
	admin.GET("/events", s.events)
	admin.POST("/admin/reset", s.reset)

	return r
}

// ---------- health and metrics ----------

func (s *Server) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	dbState := "ok"
	if err := s.db.Ping(ctx); err != nil {
		dbState = "error: " + err.Error()
	}
	pay := "configured"
	if s.fake != nil {
		pay = "fake"
	}
	body := gin.H{
		"ok": dbState == "ok", "ready": s.ready() && dbState == "ok", "version": s.version,
		"db": dbState, "redis": s.redis.Health(ctx), "paystack": pay, "mcp_tools": 4,
		"reconciler_last_run": s.rec.LastRun(), "uptime_s": int(time.Since(s.started).Seconds()),
	}
	status := http.StatusOK
	if dbState != "ok" || !s.ready() {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, body)
}

func (s *Server) metricsHandler() gin.HandlerFunc {
	h := promhttp.HandlerFor(s.metrics.reg, promhttp.HandlerOpts{})
	return func(c *gin.Context) {
		if s.cfg.MetricsToken != "" && c.GetHeader("Authorization") != "Bearer "+s.cfg.MetricsToken {
			fail(c, http.StatusUnauthorized, "unauthenticated", "Metrics token required.", "auth")
			return
		}
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Metrics is the Prometheus registry and the counters the spec names.
type Metrics struct {
	reg       *prometheus.Registry
	httpReqs  *prometheus.CounterVec
	httpDur   *prometheus.HistogramVec
	ToolCalls *prometheus.CounterVec
	ToolDur   *prometheus.HistogramVec
	Confirmed *prometheus.CounterVec
	Fallback  prometheus.CounterFunc
	SSE       prometheus.Gauge
}

func NewMetrics(redisFallback func() float64) *Metrics {
	m := &Metrics{reg: prometheus.NewRegistry()}
	m.httpReqs = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "http_requests_total"}, []string{"route", "method", "status"})
	m.httpDur = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "http_request_duration_seconds", Buckets: prometheus.DefBuckets}, []string{"route"})
	m.ToolCalls = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "mcp_tool_calls_total"}, []string{"tool", "ok", "error_code"})
	m.ToolDur = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "mcp_tool_duration_seconds", Buckets: []float64{.05, .1, .25, .5, 1, 2, 4, 8}}, []string{"tool"})
	m.Confirmed = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "orders_confirmed_total"}, []string{"status", "by"})
	m.Fallback = prometheus.NewCounterFunc(prometheus.CounterOpts{Name: "redis_fallback_total"}, redisFallback)
	m.SSE = prometheus.NewGauge(prometheus.GaugeOpts{Name: "sse_clients"})
	m.reg.MustRegister(m.httpReqs, m.httpDur, m.ToolCalls, m.ToolDur, m.Confirmed, m.Fallback, m.SSE)
	return m
}

func (m *Metrics) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		m.httpReqs.WithLabelValues(route, c.Request.Method, http.StatusText(c.Writer.Status())).Inc()
		m.httpDur.WithLabelValues(route).Observe(time.Since(start).Seconds())
	}
}
