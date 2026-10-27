package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/spiderocious/chat-to-order/backend/internal/clock"
	"github.com/spiderocious/chat-to-order/backend/internal/money"
	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

// ---------- auth ----------

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), 12)

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) login(c *gin.Context) {
	var in loginReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "malformed_json", "Body must be JSON with email and password.", "body-validation")
		return
	}
	in.Email = strings.TrimSpace(strings.ToLower(in.Email))
	if in.Email == "" || in.Password == "" {
		validation(c, fieldError{"email", "required"}, fieldError{"password", "required"})
		return
	}
	admin, err := s.db.AdminByEmail(c.Request.Context(), in.Email)
	hash := dummyHash
	if err == nil && admin.PasswordHash != "" {
		hash = []byte(admin.PasswordHash)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(in.Password)) != nil || err != nil || admin.PasswordHash == "" {
		fail(c, http.StatusUnauthorized, "invalid_credentials", "Email or password is wrong.", "auth")
		return
	}
	tok, exp, _, err := s.auth.issue(admin.ID, admin.Email, admin.TokenVersion, 24*time.Hour)
	if err != nil {
		renderErr(c, err)
		return
	}
	ok(c, http.StatusOK, gin.H{"token": tok, "expires_at": exp.UTC(), "admin": gin.H{"id": admin.ID, "email": admin.Email, "name": admin.Name}})
}

func (s *Server) logout(c *gin.Context) {
	exp, _ := c.Get("token_exp")
	if t, ok := exp.(time.Time); ok {
		s.auth.revoked.Revoke(c.Request.Context(), c.GetString("jti"), time.Until(t)+time.Minute)
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) me(c *gin.Context) {
	admin, err := s.db.AdminByID(c.Request.Context(), c.GetString("admin_id"))
	if err != nil {
		renderErr(c, err)
		return
	}
	exp, _ := c.Get("token_exp")
	ok(c, http.StatusOK, gin.H{"admin": gin.H{"id": admin.ID, "email": admin.Email, "name": admin.Name}, "expires_at": exp})
}

// ---------- vendor ----------

func (s *Server) getVendor(c *gin.Context) {
	snap, err := s.svc.Snapshot(c.Request.Context())
	if err != nil {
		renderErr(c, err)
		return
	}
	c.Header("ETag", `"`+strconv.Itoa(snap.Settings.Version)+`"`)
	ok(c, http.StatusOK, toVendorView(snap.Settings, ordering.WindowNow(snap)))
}

type vendorReq struct {
	VendorName        string                `json:"vendor_name"`
	PickupAddress     string                `json:"pickup_address"`
	RiderPhone        string                `json:"rider_phone"`
	PrepMinutes       int                   `json:"prep_minutes"`
	PendingTTLMinutes int                   `json:"pending_ttl_minutes"`
	LowStockThreshold int                   `json:"low_stock_threshold"`
	Windows           map[string]windowView `json:"windows"`
}

func mins(h string) int {
	var hh, mm int
	if n, _ := parseHHMM(h, &hh, &mm); n != 2 {
		return -1
	}
	return hh*60 + mm
}

func parseHHMM(s string, hh, mm *int) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, errors.New("bad")
	}
	a, err1 := strconv.Atoi(parts[0])
	b, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || a < 0 || a > 23 || b < 0 || b > 59 {
		return 0, errors.New("bad")
	}
	*hh, *mm = a, b
	return 2, nil
}

func (s *Server) putVendor(c *gin.Context) {
	var in vendorReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "malformed_json", "Body must be valid JSON.", "body-validation")
		return
	}
	var errs []fieldError
	if strings.TrimSpace(in.VendorName) == "" {
		errs = append(errs, fieldError{"vendor_name", "required"})
	}
	if strings.TrimSpace(in.PickupAddress) == "" {
		errs = append(errs, fieldError{"pickup_address", "required"})
	}
	if in.PrepMinutes < 5 || in.PrepMinutes > 240 {
		errs = append(errs, fieldError{"prep_minutes", "must be between 5 and 240"})
	}
	if in.PendingTTLMinutes < 5 || in.PendingTTLMinutes > 120 {
		errs = append(errs, fieldError{"pending_ttl_minutes", "must be between 5 and 120"})
	}
	if in.LowStockThreshold < 0 || in.LowStockThreshold > 50 {
		errs = append(errs, fieldError{"low_stock_threshold", "must be between 0 and 50"})
	}
	l, le := in.Windows["lunch"], in.Windows["evening"]
	for name, w := range map[string]windowView{"lunch": l, "evening": le} {
		o, oc, cl := mins(w.Open), mins(w.OrdersClose), mins(w.Close)
		if o < 0 || oc < 0 || cl < 0 {
			errs = append(errs, fieldError{"windows." + name, "times must be HH:MM"})
			continue
		}
		if !(o < oc && oc <= cl) {
			errs = append(errs, fieldError{"windows." + name, "open must be before orders_close, and orders_close before or at close"})
		}
	}
	if mins(l.Close) > 0 && mins(le.Open) > 0 && mins(l.Close) > mins(le.Open) {
		errs = append(errs, fieldError{"windows.evening", "evening must open after lunch closes"})
	}
	if len(errs) > 0 {
		validation(c, errs...)
		return
	}
	cur, err := s.db.Settings(c.Request.Context())
	if err != nil {
		renderErr(c, err)
		return
	}
	expected := ifMatch(c)
	if expected < 0 {
		expected = cur.Version
	}
	next := cur
	next.VendorName, next.PickupAddress, next.RiderPhone = strings.TrimSpace(in.VendorName), strings.TrimSpace(in.PickupAddress), strings.TrimSpace(in.RiderPhone)
	next.PrepMinutes, next.PendingTTLMinutes, next.LowStockThreshold = in.PrepMinutes, in.PendingTTLMinutes, in.LowStockThreshold
	next.LunchOpen, next.LunchOrdersClose, next.LunchClose = l.Open, l.OrdersClose, l.Close
	next.EveningOpen, next.EveningOrdersClose, next.EveningClose = le.Open, le.OrdersClose, le.Close
	out, err := s.db.UpdateSettings(c.Request.Context(), next, expected)
	if errors.Is(err, store.ErrVersionConflict) {
		latest, _ := s.db.Settings(c.Request.Context())
		snap, _ := s.svc.Snapshot(c.Request.Context())
		failWith(c, http.StatusConflict, apiError{Code: "version_conflict", Message: "Settings changed elsewhere. Reloaded the current values.", Severity: "conflict", Current: toVendorView(latest, ordering.WindowNow(snap))})
		return
	}
	if err != nil {
		renderErr(c, err)
		return
	}
	s.svc.Invalidate(c.Request.Context(), "settings.updated")
	snap, _ := s.svc.Snapshot(c.Request.Context())
	ok(c, http.StatusOK, toVendorView(out, ordering.WindowNow(snap)))
}

// ---------- menu ----------

func (s *Server) listMenu(c *gin.Context) {
	snap, err := s.svc.Snapshot(c.Request.Context())
	if err != nil {
		renderErr(c, err)
		return
	}
	st := ordering.WindowNow(snap)
	items := make([]itemView, 0, len(snap.Items))
	for _, it := range snap.Items {
		items = append(items, toItemView(it, st.Name, snap.Settings.LowStockThreshold))
	}
	ok(c, http.StatusOK, gin.H{"items": items, "now": nowView{WAT: st.NowWAT, Window: st.Name, OrdersOpen: st.OrdersOpen, OrdersCloseAt: st.OrdersCloseAt, NextWindow: st.Next}})
}

var slugRe = regexp.MustCompile(`^[a-z0-9-]{2,24}$`)

type itemReq struct {
	ID          *string `json:"id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	PriceNaira  *int64  `json:"price_naira"`
	Portions    *int    `json:"portions"`
	Available   *bool   `json:"available"`
	Window      *string `json:"window"`
	Sort        *int    `json:"sort"`
	PhotoURL    *string `json:"photo_url"`
}

func validateItem(in itemReq, create bool) []fieldError {
	var errs []fieldError
	if create {
		if in.ID == nil || !slugRe.MatchString(*in.ID) {
			errs = append(errs, fieldError{"id", "2 to 24 lowercase letters, digits or hyphens"})
		}
		if in.Name == nil || strings.TrimSpace(*in.Name) == "" {
			errs = append(errs, fieldError{"name", "required"})
		}
		if in.PriceNaira == nil {
			errs = append(errs, fieldError{"price_naira", "required"})
		}
		if in.Portions == nil {
			errs = append(errs, fieldError{"portions", "required"})
		}
		if in.Window == nil {
			errs = append(errs, fieldError{"window", "required"})
		}
	}
	if in.Name != nil && (strings.TrimSpace(*in.Name) == "" || len(*in.Name) > 80) {
		errs = append(errs, fieldError{"name", "1 to 80 characters"})
	}
	if in.PriceNaira != nil && (*in.PriceNaira < 1 || *in.PriceNaira > 1_000_000) {
		errs = append(errs, fieldError{"price_naira", "whole naira between 1 and 1,000,000"})
	}
	if in.Portions != nil && (*in.Portions < 0 || *in.Portions > 10_000) {
		errs = append(errs, fieldError{"portions", "between 0 and 10,000"})
	}
	if in.Window != nil && *in.Window != "lunch" && *in.Window != "evening" && *in.Window != "all" {
		errs = append(errs, fieldError{"window", "lunch, evening or all"})
	}
	if in.Description != nil && len(*in.Description) > 200 {
		errs = append(errs, fieldError{"description", "200 characters or fewer"})
	}
	return errs
}

func (s *Server) createItem(c *gin.Context) {
	var in itemReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "malformed_json", "Body must be valid JSON.", "body-validation")
		return
	}
	if errs := validateItem(in, true); len(errs) > 0 {
		validation(c, errs...)
		return
	}
	it := store.Item{ID: *in.ID, Name: strings.TrimSpace(*in.Name), PriceKobo: money.FromNaira(*in.PriceNaira), Portions: *in.Portions, Available: true, Window: *in.Window, Sort: 100}
	if in.Description != nil {
		it.Description = *in.Description
	}
	if in.Available != nil {
		it.Available = *in.Available
	}
	if in.Sort != nil {
		it.Sort = *in.Sort
	}
	it.PhotoURL = in.PhotoURL
	out, err := s.db.CreateItem(c.Request.Context(), it)
	if err != nil {
		renderErr(c, err)
		return
	}
	s.svc.Invalidate(c.Request.Context(), "menu.updated")
	snap, _ := s.svc.Snapshot(c.Request.Context())
	ok(c, http.StatusCreated, toItemView(out, ordering.WindowNow(snap).Name, snap.Settings.LowStockThreshold))
}

func (s *Server) patchItem(c *gin.Context) {
	var in itemReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "malformed_json", "Body must be valid JSON.", "body-validation")
		return
	}
	if errs := validateItem(in, false); len(errs) > 0 {
		validation(c, errs...)
		return
	}
	id := c.Param("id")
	cur, err := s.db.Item(c.Request.Context(), id)
	if err != nil {
		renderErr(c, err)
		return
	}
	expected := ifMatch(c)
	if expected < 0 {
		expected = cur.Version
	}
	p := store.ItemPatch{Name: in.Name, Description: in.Description, Portions: in.Portions, Available: in.Available, Window: in.Window, Sort: in.Sort, PhotoURL: in.PhotoURL}
	if in.PriceNaira != nil {
		k := int64(money.FromNaira(*in.PriceNaira))
		p.PriceKobo = &k
	}
	out, err := s.db.PatchItem(c.Request.Context(), id, p, expected)
	if errors.Is(err, store.ErrVersionConflict) {
		latest, _ := s.db.Item(c.Request.Context(), id)
		snap, _ := s.svc.Snapshot(c.Request.Context())
		failWith(c, http.StatusConflict, apiError{Code: "version_conflict", Message: "This item changed elsewhere. Reloaded the current values.", Severity: "conflict",
			Current: toItemView(latest, ordering.WindowNow(snap).Name, snap.Settings.LowStockThreshold)})
		return
	}
	if err != nil {
		renderErr(c, err)
		return
	}
	s.svc.Invalidate(c.Request.Context(), "menu.updated")
	snap, _ := s.svc.Snapshot(c.Request.Context())
	ok(c, http.StatusOK, toItemView(out, ordering.WindowNow(snap).Name, snap.Settings.LowStockThreshold))
}

// ---------- zones ----------

func (s *Server) listZones(c *gin.Context) {
	zones, err := s.db.Zones(c.Request.Context())
	if err != nil {
		renderErr(c, err)
		return
	}
	out := make([]zoneView, 0, len(zones))
	for _, z := range zones {
		out = append(out, toZoneView(z))
	}
	ok(c, http.StatusOK, gin.H{"zones": out})
}

type zoneReq struct {
	ID       *string `json:"id"`
	Name     *string `json:"name"`
	FeeNaira *int64  `json:"fee_naira"`
	Active   *bool   `json:"active"`
	Sort     *int    `json:"sort"`
}

func (s *Server) createZone(c *gin.Context) {
	var in zoneReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "malformed_json", "Body must be valid JSON.", "body-validation")
		return
	}
	var errs []fieldError
	if in.ID == nil || !slugRe.MatchString(*in.ID) {
		errs = append(errs, fieldError{"id", "2 to 24 lowercase letters, digits or hyphens"})
	}
	if in.Name == nil || strings.TrimSpace(*in.Name) == "" {
		errs = append(errs, fieldError{"name", "required"})
	}
	if in.FeeNaira == nil || *in.FeeNaira < 0 || *in.FeeNaira > 100_000 {
		errs = append(errs, fieldError{"fee_naira", "whole naira between 0 and 100,000"})
	}
	if len(errs) > 0 {
		validation(c, errs...)
		return
	}
	z := store.Zone{ID: *in.ID, Name: strings.TrimSpace(*in.Name), FeeKobo: money.FromNaira(*in.FeeNaira), Active: true, Sort: 100}
	if in.Active != nil {
		z.Active = *in.Active
	}
	if in.Sort != nil {
		z.Sort = *in.Sort
	}
	out, err := s.db.CreateZone(c.Request.Context(), z)
	if err != nil {
		renderErr(c, err)
		return
	}
	s.svc.Invalidate(c.Request.Context(), "menu.updated")
	ok(c, http.StatusCreated, toZoneView(out))
}

func (s *Server) patchZone(c *gin.Context) {
	var in zoneReq
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "malformed_json", "Body must be valid JSON.", "body-validation")
		return
	}
	if in.FeeNaira != nil && (*in.FeeNaira < 0 || *in.FeeNaira > 100_000) {
		validation(c, fieldError{"fee_naira", "whole naira between 0 and 100,000"})
		return
	}
	p := store.ZonePatch{Name: in.Name, Active: in.Active, Sort: in.Sort}
	if in.FeeNaira != nil {
		k := int64(money.FromNaira(*in.FeeNaira))
		p.FeeKobo = &k
	}
	out, err := s.db.PatchZone(c.Request.Context(), c.Param("id"), p)
	if err != nil {
		renderErr(c, err)
		return
	}
	s.svc.Invalidate(c.Request.Context(), "menu.updated")
	ok(c, http.StatusOK, toZoneView(out))
}

// ---------- orders ----------

func lagosDayStart() time.Time {
	n := clock.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, clock.Lagos)
}

var validStatuses = map[string]bool{store.StatusPending: true, store.StatusPaid: true, store.StatusNeedsAttention: true, store.StatusResolved: true, store.StatusCancelled: true, store.StatusExpired: true}

func (s *Server) listOrders(c *gin.Context) {
	f := store.OrdersFilter{Cursor: c.Query("cursor"), Today: c.Query("today") == "true"}
	if lim := c.Query("limit"); lim != "" {
		n, err := strconv.Atoi(lim)
		if err != nil || n < 1 || n > 100 {
			validation(c, fieldError{"limit", "between 1 and 100"})
			return
		}
		f.Limit = n
	}
	if st := c.Query("status"); st != "" {
		for _, p := range strings.Split(st, ",") {
			p = strings.TrimSpace(p)
			if !validStatuses[p] {
				validation(c, fieldError{"status", "unknown status " + p})
				return
			}
			f.Statuses = append(f.Statuses, p)
		}
	}
	page, err := s.db.ListOrders(c.Request.Context(), f, lagosDayStart())
	if err != nil {
		renderErr(c, err)
		return
	}
	snap, err := s.svc.Snapshot(c.Request.Context())
	if err != nil {
		renderErr(c, err)
		return
	}
	rows := make([]orderRow, 0, len(page.Items))
	for _, o := range page.Items {
		rows = append(rows, toOrderRow(o, snap))
	}
	okPage(c, rows, page.NextCursor, page.HasMore)
}

var orderNoRe = regexp.MustCompile(`^[A-Z]{2,5}-\d{4,}$`)

func (s *Server) getOrder(c *gin.Context) {
	no := c.Param("order_no")
	if !orderNoRe.MatchString(no) {
		fail(c, http.StatusNotFound, "not_found", "No order "+no+".", "not-found")
		return
	}
	ctx := c.Request.Context()
	o, err := s.db.OrderByNo(ctx, no)
	if errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "not_found", "No order "+no+".", "not-found")
		return
	}
	if err != nil {
		renderErr(c, err)
		return
	}
	snap, _ := s.svc.Snapshot(ctx)
	events, _ := s.db.Events(ctx, o.ID)
	checks, _ := s.db.PaymentChecks(ctx, o.ID)
	calls, _ := s.db.ToolCallsForOrder(ctx, o.OrderNo)
	ok(c, http.StatusOK, toOrderDetail(o, snap, events, checks, calls))
}

type noteReq struct {
	Note   string `json:"note"`
	Reason string `json:"reason"`
}

func (s *Server) resolveOrder(c *gin.Context) {
	var in noteReq
	_ = c.ShouldBindJSON(&in)
	if strings.TrimSpace(in.Note) == "" {
		validation(c, fieldError{"note", "required: say what you did"})
		return
	}
	o, err := s.svc.ResolveOrder(c.Request.Context(), c.Param("order_no"), strings.TrimSpace(in.Note), "admin:"+c.GetString("admin_id"))
	if err != nil {
		renderErr(c, err)
		return
	}
	snap, _ := s.svc.Snapshot(c.Request.Context())
	ok(c, http.StatusOK, toOrderRow(o, snap))
}

func (s *Server) cancelOrder(c *gin.Context) {
	var in noteReq
	_ = c.ShouldBindJSON(&in)
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "cancelled by admin"
	}
	o, err := s.svc.CancelOrder(c.Request.Context(), c.Param("order_no"), reason, "admin:"+c.GetString("admin_id"))
	if err != nil {
		renderErr(c, err)
		return
	}
	snap, _ := s.svc.Snapshot(c.Request.Context())
	ok(c, http.StatusOK, toOrderRow(o, snap))
}

// ---------- stats and activity ----------

func (s *Server) statsToday(c *gin.Context) {
	ctx := c.Request.Context()
	snap, err := s.svc.Snapshot(ctx)
	if err != nil {
		renderErr(c, err)
		return
	}
	st := ordering.WindowNow(snap)
	ds, err := s.db.StatsSince(ctx, lagosDayStart())
	if err != nil {
		renderErr(c, err)
		return
	}
	type stock struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Portions int    `json:"portions"`
		Low      bool   `json:"low"`
		Offered  bool   `json:"offered_now"`
	}
	var stocks []stock
	for _, it := range snap.Items {
		stocks = append(stocks, stock{ID: it.ID, Name: it.Name, Portions: it.Portions, Low: it.Portions <= snap.Settings.LowStockThreshold,
			Offered: it.Available && it.Portions > 0 && (it.Window == "all" || it.Window == st.Name)})
	}
	if stocks == nil {
		stocks = []stock{}
	}
	ok(c, http.StatusOK, gin.H{
		"date_wat": clock.Now().Format("2006-01-02"), "window": st.Name, "orders_open": st.OrdersOpen, "orders_close_at": st.OrdersCloseAt, "next_window": st.Next,
		"paid_count": ds.PaidCount, "paid_total_kobo": ds.PaidTotalKobo, "paid_total_display": money.Kobo(ds.PaidTotalKobo).Format(),
		"awaiting_payment_count": ds.AwaitingCount, "needs_attention_count": ds.AttentionCount, "cancelled_count": ds.CancelledCount, "expired_count": ds.ExpiredCount,
		"stock": stocks, "low_stock_threshold": snap.Settings.LowStockThreshold, "redis": s.redis.Health(ctx),
	})
}

func (s *Server) listActivity(c *gin.Context) {
	f := store.ActivityFilter{Tool: c.Query("tool"), OrderNo: c.Query("order_no"), Cursor: c.Query("cursor")}
	if lim := c.Query("limit"); lim != "" {
		n, err := strconv.Atoi(lim)
		if err != nil || n < 1 || n > 100 {
			validation(c, fieldError{"limit", "between 1 and 100"})
			return
		}
		f.Limit = n
	}
	if v := c.Query("ok"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			validation(c, fieldError{"ok", "true or false"})
			return
		}
		f.OK = &b
	}
	page, err := s.db.ListToolCalls(c.Request.Context(), f)
	if err != nil {
		renderErr(c, err)
		return
	}
	rows := make([]toolCallView, 0, len(page.Items))
	for _, t := range page.Items {
		rows = append(rows, toToolCallView(t))
	}
	okPage(c, rows, page.NextCursor, page.HasMore)
}

// ---------- reset ----------

func (s *Server) reset(c *gin.Context) {
	if err := s.db.Reset(c.Request.Context(), true); err != nil {
		renderErr(c, err)
		return
	}
	s.svc.Invalidate(c.Request.Context(), "")
	s.svc.Publish(c.Request.Context(), redisx.Event{Type: "reset"})
	s.log.Warn("demo.reset", "admin", c.GetString("admin_email"))
	ok(c, http.StatusOK, gin.H{"reset": true})
}
