// Package clock is the single source of "now" in Lagos time, and the window
// logic that depends on it. Tests replace Now.
package clock

import (
	"fmt"
	"time"
	_ "time/tzdata" // Africa/Lagos must resolve inside distroless
)

var Lagos = mustLoad("Africa/Lagos")

// Now returns the current time in Lagos. Tests override it.
var Now = func() time.Time { return time.Now().In(Lagos) }

func mustLoad(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

// HHMM is a wall-clock time like "14:30".
type HHMM string

func (h HHMM) minutes() (int, error) {
	var hh, mm int
	if _, err := fmt.Sscanf(string(h), "%d:%d", &hh, &mm); err != nil {
		return 0, fmt.Errorf("bad time %q", h)
	}
	if hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, fmt.Errorf("bad time %q", h)
	}
	return hh*60 + mm, nil
}

// Valid reports whether h parses.
func (h HHMM) Valid() bool { _, err := h.minutes(); return err == nil }

// WindowDef is one service window.
type WindowDef struct {
	Name        string
	Open        HHMM
	OrdersClose HHMM
	Close       HHMM
}

// WindowState is what the rest of the system needs to know about now.
type WindowState struct {
	Name           string // lunch | evening | closed
	OrdersOpen     bool
	OrdersCloseAt  string // "14:30" when a window is current, else ""
	KitchenCloseAt string
	Next           *NextWindow
	NowWAT         string
}

type NextWindow struct {
	Name    string `json:"name"`
	OpensAt string `json:"opens_at"`
}

// Window resolves the current window from the definitions and now.
// Orders are open from Open until OrdersClose; the window is "current" until Close.
func Window(defs []WindowDef, now time.Time) WindowState {
	now = now.In(Lagos)
	cur := now.Hour()*60 + now.Minute()
	st := WindowState{Name: "closed", NowWAT: now.Format("15:04")}

	for _, d := range defs {
		o, _ := d.Open.minutes()
		oc, _ := d.OrdersClose.minutes()
		c, _ := d.Close.minutes()
		if cur >= o && cur < c {
			st.Name = d.Name
			st.OrdersOpen = cur < oc
			st.OrdersCloseAt = string(d.OrdersClose)
			st.KitchenCloseAt = string(d.Close)
			if !st.OrdersOpen {
				st.Next = nextAfter(defs, cur)
			}
			return st
		}
	}
	st.Next = nextAfter(defs, cur)
	return st
}

// nextAfter finds the next window that opens after cur minutes today, or the
// first window tomorrow.
func nextAfter(defs []WindowDef, cur int) *NextWindow {
	var best *WindowDef
	bestMin := 1 << 30
	for i := range defs {
		o, _ := defs[i].Open.minutes()
		if o > cur && o < bestMin {
			best, bestMin = &defs[i], o
		}
	}
	if best == nil {
		// tomorrow: earliest opening
		for i := range defs {
			o, _ := defs[i].Open.minutes()
			if o < bestMin {
				best, bestMin = &defs[i], o
			}
		}
	}
	if best == nil {
		return nil
	}
	return &NextWindow{Name: best.Name, OpensAt: string(best.Open)}
}

// ReadyBy is paid_at + prep minutes, rounded up to the next 5 minutes.
func ReadyBy(paidAt time.Time, prepMinutes int) time.Time {
	t := paidAt.In(Lagos).Add(time.Duration(prepMinutes) * time.Minute)
	rem := t.Minute() % 5
	if rem != 0 || t.Second() != 0 {
		t = t.Add(time.Duration(5-rem) * time.Minute)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, Lagos)
}

// WAT formats a time as "15:04" in Lagos.
func WAT(t time.Time) string { return t.In(Lagos).Format("15:04") }
