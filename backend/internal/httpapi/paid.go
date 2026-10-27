package httpapi

import (
	"errors"
	"html/template"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

var refRe = regexp.MustCompile(`^[A-Z]{2,5}-\d{4,}-[a-f0-9]{4}$`)

var paidTpl = template.Must(template.New("paid").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}}</title>
<style>
:root{color-scheme:light}
body{margin:0;background:#fbfaf7;color:#1b2430;font-family:Nunito,system-ui,sans-serif;display:flex;min-height:100vh;align-items:center;justify-content:center;padding:16px}
.card{max-width:420px;width:100%;background:#fff;border:2px solid #1b2430;border-radius:20px 6px 20px 6px;box-shadow:3px 4px 0 #1b2430;padding:28px}
.mark{width:40px;height:40px;border-radius:10px;background:#1f8a5b;color:#fff;font-weight:800;display:flex;align-items:center;justify-content:center;margin-bottom:14px;font-size:18px}
h1{font-size:22px;margin:0 0 8px;line-height:1.2}
p{margin:0 0 10px;color:#3d4c5c;line-height:1.5}
.ref{font-family:ui-monospace,monospace;background:#f3f1ea;padding:3px 8px;border-radius:6px}
.big{display:inline-block;background:#e3f4ea;color:#14603f;border:1.5px solid #b4dfc6;border-radius:999px;padding:6px 14px;font-weight:800;margin-top:6px}
</style></head><body><div class="card"><div class="mark">TP</div>
<h1>{{.Heading}}</h1>
<p>{{.Body}}</p>
{{if .OrderNo}}<p>Order <span class="ref">{{.OrderNo}}</span></p>{{end}}
<p class="big">Reply <b>paid</b> on WhatsApp</p>
</div></body></html>`))

// paidPage is where Paystack sends the customer after checkout. It never
// claims the payment succeeded; only confirm_order does that.
func (s *Server) paidPage(c *gin.Context) {
	ref := c.Query("ref")
	data := gin.H{"Title": "Thanks", "Heading": "Thanks!", "Body": "Go back to WhatsApp and reply 'paid' so the TPlus assistant can confirm your order.", "OrderNo": ""}
	if refRe.MatchString(ref) {
		if s.fake != nil && c.Query("fake") == "1" {
			s.fake.MarkPaid(ref)
		}
		o, err := s.db.OrderByReference(c.Request.Context(), ref)
		if err == nil {
			data["Heading"] = "Payment received"
			data["Body"] = "Go back to WhatsApp and reply 'paid' so the TPlus assistant can confirm order " + o.OrderNo + "."
			data["OrderNo"] = o.OrderNo
			data["Title"] = "Payment received"
		} else if !errors.Is(err, store.ErrNotFound) {
			s.log.Warn("paid.lookup", "err", err)
		}
	}
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	_ = paidTpl.Execute(c.Writer, data)
}
