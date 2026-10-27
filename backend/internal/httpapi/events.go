package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// events is the SSE stream for the board. Events carry ids and types only;
// every event is a refetch on the client. The server closes after ten minutes
// and the client reconnects.
func (s *Server) events(c *gin.Context) {
	flusher, okk := c.Writer.(http.Flusher)
	if !okk {
		fail(c, http.StatusInternalServerError, "internal", "Streaming unsupported.", "server-fault")
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-store")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	ch, unsub := s.bus.Subscribe(c.Request.Context())
	defer unsub()
	s.metrics.SSE.Inc()
	defer s.metrics.SSE.Dec()

	write := func(id int64, typ string, data any) bool {
		b, _ := json.Marshal(data)
		if _, err := fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", id, typ, b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// hello: tells the client whether Redis is degraded so it can start polling
	degraded := s.redis.Degraded()
	write(time.Now().UnixMilli(), "hello", gin.H{"degraded": degraded, "version": s.version})

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	deadline := time.NewTimer(10 * time.Minute)
	defer deadline.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-deadline.C:
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(c.Writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
			if d := s.redis.Degraded(); d != degraded {
				degraded = d
				write(time.Now().UnixMilli(), "degraded", gin.H{"on": d})
			}
		case e, open := <-ch:
			if !open {
				return
			}
			if !write(time.Now().UnixMilli(), e.Type, e) {
				return
			}
		}
	}
}
