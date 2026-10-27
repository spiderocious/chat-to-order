package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oklog/ulid/v2"

	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
)

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 64 {
			id = "req_" + ulid.Make().String()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/metrics" {
			return
		}
		lvl := slog.LevelInfo
		if c.Writer.Status() >= 500 {
			lvl = slog.LevelError
		}
		attrs := []any{"method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(), "request_id", c.GetString("request_id"), "ip", c.ClientIP()}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "err", c.Errors.String())
		}
		log.Log(c.Request.Context(), lvl, "http", attrs...)
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}

// cors allows exact origins plus *.netlify.app previews, on /api only.
func cors(origins []string) gin.HandlerFunc {
	allowed := func(o string) bool {
		if o == "" {
			return false
		}
		for _, a := range origins {
			if strings.EqualFold(a, o) {
				return true
			}
			if strings.HasPrefix(a, "*.") && strings.HasSuffix(strings.ToLower(o), strings.ToLower(a[1:])) && strings.HasPrefix(o, "https://") {
				return true
			}
		}
		return false
	}
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		c.Header("Vary", "Origin")
		if allowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, If-Match, Last-Event-ID, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID, Retry-After, ETag")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func bodyLimit(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

func rateLimit(l redisx.Limiter, name string, limit int, window time.Duration, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := name + ":" + keyFn(c)
		okk, retry := l.Allow(c.Request.Context(), key, limit, window)
		if !okk {
			c.Header("Retry-After", strconv.Itoa(retry))
			failWith(c, http.StatusTooManyRequests, apiError{Code: "rate_limited",
				Message: fmt.Sprintf("Too many attempts. Try again in %d seconds.", retry), Severity: "rate-limited", RetryAfter: retry})
			return
		}
		c.Next()
	}
}

func byIP(c *gin.Context) string { return c.ClientIP() }

// ---------- JWT ----------

type Claims struct {
	Email        string `json:"email"`
	TokenVersion int    `json:"tv"`
	jwt.RegisteredClaims
}

type authDeps struct {
	secret  []byte
	revoked redisx.Revoked
}

func (a authDeps) issue(adminID, email string, tv int, ttl time.Duration) (string, time.Time, string, error) {
	exp := time.Now().Add(ttl)
	jti := ulid.Make().String()
	claims := Claims{Email: email, TokenVersion: tv, RegisteredClaims: jwt.RegisteredClaims{
		Subject: adminID, ID: jti, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(exp)}}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
	return tok, exp, jti, err
}

func (a authDeps) parse(raw string) (*Claims, error) {
	var claims Claims
	tok, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("bad alg")
		}
		return a.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return &claims, nil
}

// requireJWT accepts the bearer from the header, or ?token= for SSE
// fallback clients that cannot set headers (not used by our client, kept for curl).
func (s *Server) requireJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if raw == "" {
			fail(c, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.", "auth")
			return
		}
		claims, err := s.auth.parse(raw)
		if err != nil {
			fail(c, http.StatusUnauthorized, "unauthenticated", "Your session has expired. Sign in again.", "auth")
			return
		}
		if s.auth.revoked.IsRevoked(c.Request.Context(), claims.ID) {
			fail(c, http.StatusUnauthorized, "token_revoked", "You signed out. Sign in again.", "auth")
			return
		}
		c.Set("admin_id", claims.Subject)
		c.Set("admin_email", claims.Email)
		c.Set("jti", claims.ID)
		c.Set("token_exp", claims.ExpiresAt.Time)
		c.Next()
	}
}

// ---------- Idempotency-Key replay for admin writes ----------

type bodyRecorder struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (b *bodyRecorder) Write(p []byte) (int, error) {
	b.buf.Write(p)
	return b.ResponseWriter.Write(p)
}

func (s *Server) idempotent() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" || len(key) > 128 {
			c.Next()
			return
		}
		raw, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		sum := sha256.Sum256(append([]byte(c.Request.Method+c.Request.URL.Path+"|"), raw...))
		hash := hex.EncodeToString(sum[:])
		full := c.GetString("admin_id") + ":" + key
		if prev, found := s.idem.Get(c.Request.Context(), full); found {
			if prev.BodyHash != hash {
				fail(c, http.StatusUnprocessableEntity, "idempotency_mismatch", "This Idempotency-Key was already used with a different request.", "suspicious-validation")
				return
			}
			c.Header("Idempotent-Replayed", "true")
			c.Data(prev.Status, "application/json", prev.Body)
			c.Abort()
			return
		}
		rec := &bodyRecorder{ResponseWriter: c.Writer}
		c.Writer = rec
		c.Next()
		if st := c.Writer.Status(); st >= 200 && st < 300 {
			s.idem.Put(c.Request.Context(), full, redisx.StoredResponse{Status: st, BodyHash: hash, Body: rec.buf.Bytes()})
		}
	}
}

// ifMatch parses If-Match: "3" → 3. Returns -1 when absent.
func ifMatch(c *gin.Context) int {
	v := strings.Trim(c.GetHeader("If-Match"), `" `)
	if v == "" {
		return -1
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return -1
	}
	return n
}
