// Package reconcile is the ticker that catches what the conversation missed:
// a customer who paid and never said "paid", an order that never got a
// reference, and pending orders past their TTL.
package reconcile

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
	"github.com/spiderocious/chat-to-order/backend/internal/redisx"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

type Reconciler struct {
	db      *store.DB
	svc     *ordering.Service
	locks   redisx.Locker
	log     *slog.Logger
	lastRun atomic.Int64
	Counts  struct{ Confirmed, Attention, InitLost, Expired, Checked, Errors atomic.Int64 }
}

func New(db *store.DB, svc *ordering.Service, locks redisx.Locker, log *slog.Logger) *Reconciler {
	return &Reconciler{db: db, svc: svc, locks: locks, log: log}
}

func (r *Reconciler) LastRun() time.Time {
	if v := r.lastRun.Load(); v > 0 {
		return time.Unix(v, 0)
	}
	return time.Time{}
}

// Run blocks until ctx is done. One pass per minute, guarded by a lock so
// two replicas do not double-verify.
func (r *Reconciler) Run(ctx context.Context) {
	t := time.NewTicker(60 * time.Second)
	defer t.Stop()
	r.Pass(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.Pass(ctx)
		}
	}
}

func (r *Reconciler) Pass(ctx context.Context) {
	release, ok := r.locks.Acquire(ctx, "reconcile:tick", 55*time.Second)
	if !ok {
		return
	}
	defer release()
	r.lastRun.Store(time.Now().Unix())
	pctx, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	// 1. re-verify pending orders that have a reference
	pending, err := r.db.PendingForReconcile(pctx, 2*time.Minute, 5*time.Minute, 20)
	if err != nil {
		r.log.Error("reconcile.list", "err", err)
		r.Counts.Errors.Add(1)
	}
	for _, o := range pending {
		if o.Reference == nil {
			continue
		}
		r.Counts.Checked.Add(1)
		out, err := r.svc.ConfirmOrder(pctx, *o.Reference, "reconciler")
		_ = r.db.TouchReconciled(pctx, o.ID)
		if err != nil {
			continue // not paid yet, or Paystack down; next pass
		}
		if !out.AlreadyConfirmed {
			switch out.Status {
			case store.StatusPaid:
				r.Counts.Confirmed.Add(1)
			case store.StatusNeedsAttention:
				r.Counts.Attention.Add(1)
			}
			r.log.Info("reconcile.confirmed", "order_no", out.OrderNo, "status", out.Status)
		}
	}

	// 2. orders that never got a Paystack reference (crash between insert and initialize)
	if ids, err := r.db.CancelReferenceless(pctx, 2*time.Minute); err == nil {
		for _, id := range ids {
			_ = store.AddEvent(pctx, r.db.Pool, id, "init_lost", "reconciler", nil)
			r.Counts.InitLost.Add(1)
		}
		if len(ids) > 0 {
			r.log.Warn("reconcile.init_lost", "count", len(ids))
		}
	}

	// 3. expire
	if ids, err := r.db.ExpirePending(pctx); err == nil && len(ids) > 0 {
		for _, id := range ids {
			_ = store.AddEvent(pctx, r.db.Pool, id, "expired", "reconciler", nil)
			if o, err := r.db.OrderByID(pctx, id); err == nil {
				r.svc.Publish(pctx, redisx.Event{Type: "order.updated", OrderNo: o.OrderNo, Status: store.StatusExpired})
			}
		}
		r.Counts.Expired.Add(int64(len(ids)))
		r.log.Info("reconcile.expired", "count", len(ids))
	}

	// 4. prune once a day around 03:00 WAT (checked every pass, cheap)
	if now := time.Now(); now.Hour() == 3 && now.Minute() == 0 {
		_ = r.db.Prune(pctx, 90*24*time.Hour)
	}
}
