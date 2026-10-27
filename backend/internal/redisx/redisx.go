// Package redisx holds every Redis-backed port behind a small interface with an
// in-process fallback. Every use fails open: when Redis is unreachable the
// service keeps serving with weaker guarantees and counts the fallback.
package redisx

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis with a per-call deadline and a degraded flag.
type Client struct {
	rdb      *redis.Client
	degraded atomic.Bool
	Fallback atomic.Int64 // counter for metrics
	log      *slog.Logger
}

// New connects; a nil return means "no Redis configured" and callers use the
// memory implementations. A reachable URL that fails to ping still returns a
// client in degraded mode so it can recover.
func New(ctx context.Context, url string, log *slog.Logger) *Client {
	if url == "" {
		return nil
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		log.Warn("redis.parse", "err", err)
		return nil
	}
	opt.DialTimeout = 2 * time.Second
	opt.ReadTimeout = 300 * time.Millisecond
	opt.WriteTimeout = 300 * time.Millisecond
	opt.PoolSize = 10
	opt.MinIdleConns = 2
	opt.MaxRetries = 1
	c := &Client{rdb: redis.NewClient(opt), log: log}
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := c.rdb.Ping(pctx).Err(); err != nil {
		log.Warn("redis.ping", "err", err, "mode", "degraded")
		c.degraded.Store(true)
	}
	return c
}

func (c *Client) Degraded() bool { return c != nil && c.degraded.Load() }

// Healthy reports ok/degraded/absent for /healthz.
func (c *Client) Health(ctx context.Context) string {
	if c == nil {
		return "absent"
	}
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		c.degraded.Store(true)
		return "degraded"
	}
	c.degraded.Store(false)
	return "ok"
}

func (c *Client) fail(use string, err error) {
	c.Fallback.Add(1)
	if !c.degraded.Swap(true) {
		c.log.Warn("redis.fallback", "use", use, "err", err)
	}
}

func (c *Client) ok() { c.degraded.Store(false) }

// ---------- Cache ----------

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration)
	Del(ctx context.Context, key string)
	Layer() string
}

type memCache struct {
	mu sync.Mutex
	m  map[string]memEntry
}
type memEntry struct {
	val []byte
	exp time.Time
}

func NewMemCache() Cache { return &memCache{m: map[string]memEntry{}} }

func (m *memCache) Get(_ context.Context, k string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.m[k]
	if !ok || time.Now().After(e.exp) {
		delete(m.m, k)
		return nil, false
	}
	return e.val, true
}
func (m *memCache) Set(_ context.Context, k string, v []byte, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[k] = memEntry{val: v, exp: time.Now().Add(ttl)}
}
func (m *memCache) Del(_ context.Context, k string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
}
func (m *memCache) Layer() string { return "memory" }

type redisCache struct {
	c   *Client
	mem Cache
}

// NewCache returns a Redis-backed cache that falls back to memory.
func NewCache(c *Client) Cache {
	if c == nil {
		return NewMemCache()
	}
	return &redisCache{c: c, mem: NewMemCache()}
}

func (r *redisCache) Get(ctx context.Context, k string) ([]byte, bool) {
	b, err := r.c.rdb.Get(ctx, k).Bytes()
	if err == nil {
		r.c.ok()
		return b, true
	}
	if errors.Is(err, redis.Nil) {
		r.c.ok()
		return nil, false
	}
	r.c.fail("cache", err)
	return r.mem.Get(ctx, k)
}
func (r *redisCache) Set(ctx context.Context, k string, v []byte, ttl time.Duration) {
	r.mem.Set(ctx, k, v, ttl)
	if err := r.c.rdb.Set(ctx, k, v, ttl).Err(); err != nil {
		r.c.fail("cache", err)
	}
}
func (r *redisCache) Del(ctx context.Context, k string) {
	r.mem.Del(ctx, k)
	if err := r.c.rdb.Del(ctx, k).Err(); err != nil {
		r.c.fail("cache", err)
	}
}
func (r *redisCache) Layer() string { return "redis" }

// ---------- Locks ----------

type Locker interface {
	// Acquire returns a release func and true, or nil and false if held.
	Acquire(ctx context.Context, key string, ttl time.Duration) (func(), bool)
}

type memLocker struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func NewMemLocker() Locker { return &memLocker{m: map[string]time.Time{}} }

func (l *memLocker) Acquire(_ context.Context, key string, ttl time.Duration) (func(), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if exp, ok := l.m[key]; ok && time.Now().Before(exp) {
		return nil, false
	}
	l.m[key] = time.Now().Add(ttl)
	return func() {
		l.mu.Lock()
		delete(l.m, key)
		l.mu.Unlock()
	}, true
}

var releaseScript = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)

type redisLocker struct {
	c   *Client
	mem Locker
}

func NewLocker(c *Client) Locker {
	if c == nil {
		return NewMemLocker()
	}
	return &redisLocker{c: c, mem: NewMemLocker()}
}

func (l *redisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (func(), bool) {
	token := time.Now().UnixNano()
	ok, err := l.c.rdb.SetNX(ctx, "lock:"+key, token, ttl).Result()
	if err != nil {
		l.c.fail("lock", err)
		return l.mem.Acquire(ctx, key, ttl)
	}
	l.c.ok()
	if !ok {
		return nil, false
	}
	return func() {
		rctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		_ = releaseScript.Run(rctx, l.c.rdb, []string{"lock:" + key}, token).Err()
	}, true
}

// AcquireWait retries Acquire until it succeeds or the wait elapses.
func AcquireWait(ctx context.Context, l Locker, key string, ttl, wait time.Duration) (func(), bool) {
	deadline := time.Now().Add(wait)
	for {
		if rel, ok := l.Acquire(ctx, key, ttl); ok {
			return rel, true
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return nil, false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ---------- Rate limit ----------

type Limiter interface {
	// Allow counts a hit for key in a fixed window and reports whether it is
	// within limit, plus seconds until the window resets.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int)
}

type memLimiter struct {
	mu sync.Mutex
	m  map[string]*memWindow
}
type memWindow struct {
	n   int
	exp time.Time
}

func NewMemLimiter() Limiter { return &memLimiter{m: map[string]*memWindow{}} }

func (l *memLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.m[key]
	if !ok || time.Now().After(w.exp) {
		w = &memWindow{exp: time.Now().Add(window)}
		l.m[key] = w
	}
	w.n++
	return w.n <= limit, int(time.Until(w.exp).Seconds()) + 1
}

var incrScript = redis.NewScript(`local n = redis.call("incr", KEYS[1]) if n == 1 then redis.call("pexpire", KEYS[1], ARGV[1]) end local ttl = redis.call("pttl", KEYS[1]) return {n, ttl}`)

type redisLimiter struct {
	c   *Client
	mem Limiter
}

func NewLimiter(c *Client) Limiter {
	if c == nil {
		return NewMemLimiter()
	}
	return &redisLimiter{c: c, mem: NewMemLimiter()}
}

func (l *redisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int) {
	res, err := incrScript.Run(ctx, l.c.rdb, []string{"rl:" + key}, window.Milliseconds()).Int64Slice()
	if err != nil || len(res) != 2 {
		l.c.fail("ratelimit", err)
		return l.mem.Allow(ctx, key, limit, window)
	}
	l.c.ok()
	retry := int(res[1]/1000) + 1
	return res[0] <= int64(limit), retry
}

// ---------- Idempotency ----------

type StoredResponse struct {
	Status   int    `json:"status"`
	BodyHash string `json:"body_hash"`
	Body     []byte `json:"body"`
}

// Idempotency stores admin write responses for 24 h under a client key.
type Idempotency interface {
	Get(ctx context.Context, key string) (StoredResponse, bool)
	Put(ctx context.Context, key string, r StoredResponse)
}

func NewIdempotency(c *Client) Idempotency {
	return &idem{cache: NewCache(c)}
}

type idem struct{ cache Cache }

func (i *idem) Get(ctx context.Context, key string) (StoredResponse, bool) {
	b, ok := i.cache.Get(ctx, "idem:"+key)
	if !ok {
		return StoredResponse{}, false
	}
	var r StoredResponse
	if err := jsonUnmarshal(b, &r); err != nil {
		return StoredResponse{}, false
	}
	return r, true
}
func (i *idem) Put(ctx context.Context, key string, r StoredResponse) {
	b, _ := jsonMarshal(r)
	i.cache.Set(ctx, "idem:"+key, b, 24*time.Hour)
}

// ---------- Revocation ----------

// Revoked tracks logged-out JWT ids for their remaining lifetime.
type Revoked interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration)
	IsRevoked(ctx context.Context, jti string) bool
}

type revoked struct{ cache Cache }

func NewRevoked(c *Client) Revoked { return &revoked{cache: NewCache(c)} }

func (r *revoked) Revoke(ctx context.Context, jti string, ttl time.Duration) {
	r.cache.Set(ctx, "jwt:revoked:"+jti, []byte("1"), ttl)
}
func (r *revoked) IsRevoked(ctx context.Context, jti string) bool {
	_, ok := r.cache.Get(ctx, "jwt:revoked:"+jti)
	return ok
}

// ---------- Event bus ----------

type Event struct {
	Type    string `json:"type"`
	OrderNo string `json:"order_no,omitempty"`
	Status  string `json:"status,omitempty"`
	At      string `json:"at"`
	On      *bool  `json:"on,omitempty"`
}

type Bus interface {
	Publish(ctx context.Context, e Event)
	Subscribe(ctx context.Context) (<-chan Event, func())
}

type memBus struct {
	mu   sync.Mutex
	subs map[int]chan Event
	next int
}

func NewMemBus() *memBus { return &memBus{subs: map[int]chan Event{}} }

func (b *memBus) Publish(_ context.Context, e Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default: // slow subscriber: drop; the client refetches on reconnect
		}
	}
}

func (b *memBus) Subscribe(_ context.Context) (<-chan Event, func()) {
	ch := make(chan Event, 32)
	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = ch
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subs, id)
		b.mu.Unlock()
	}
}

type redisBus struct {
	c    *Client
	mem  *memBus
	once sync.Once
}

// NewBus returns a bus that publishes through Redis so every replica's SSE
// clients hear every event, and delivers locally when Redis is down.
func NewBus(ctx context.Context, c *Client) Bus {
	if c == nil {
		return NewMemBus()
	}
	b := &redisBus{c: c, mem: NewMemBus()}
	go b.pump(ctx)
	return b
}

const channel = "events"

func (b *redisBus) pump(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		sub := b.c.rdb.Subscribe(ctx, channel)
		ch := sub.Channel()
		for msg := range ch {
			var e Event
			if err := jsonUnmarshal([]byte(msg.Payload), &e); err == nil {
				b.mem.Publish(ctx, e)
			}
		}
		_ = sub.Close()
		time.Sleep(time.Second)
	}
}

func (b *redisBus) Publish(ctx context.Context, e Event) {
	payload, _ := jsonMarshal(e)
	if err := b.c.rdb.Publish(ctx, channel, payload).Err(); err != nil {
		b.c.fail("events", err)
		b.mem.Publish(ctx, e) // local delivery only
		return
	}
	b.c.ok()
}

func (b *redisBus) Subscribe(ctx context.Context) (<-chan Event, func()) {
	return b.mem.Subscribe(ctx)
}
