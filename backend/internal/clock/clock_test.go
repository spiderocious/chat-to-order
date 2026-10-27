package clock

import (
	"testing"
	"time"
)

func at(h, m int) time.Time { return time.Date(2026, 10, 27, h, m, 0, 0, Lagos) }

var defs = []WindowDef{
	{Name: "lunch", Open: "11:00", OrdersClose: "14:30", Close: "15:00"},
	{Name: "evening", Open: "17:00", OrdersClose: "20:00", Close: "20:30"},
}

func TestWindow(t *testing.T) {
	cases := []struct {
		h, m       int
		name       string
		ordersOpen bool
		next       string
	}{
		{10, 59, "closed", false, "lunch"},
		{11, 0, "lunch", true, ""},
		{14, 29, "lunch", true, ""},
		{14, 30, "lunch", false, "evening"},
		{14, 59, "lunch", false, "evening"},
		{15, 0, "closed", false, "evening"},
		{16, 59, "closed", false, "evening"},
		{17, 0, "evening", true, ""},
		{19, 59, "evening", true, ""},
		{20, 0, "evening", false, "lunch"},
		{20, 30, "closed", false, "lunch"},
		{23, 0, "closed", false, "lunch"},
	}
	for _, c := range cases {
		st := Window(defs, at(c.h, c.m))
		if st.Name != c.name || st.OrdersOpen != c.ordersOpen {
			t.Errorf("%02d:%02d: got %s/%v want %s/%v", c.h, c.m, st.Name, st.OrdersOpen, c.name, c.ordersOpen)
		}
		if c.next != "" && (st.Next == nil || st.Next.Name != c.next) {
			t.Errorf("%02d:%02d: next got %+v want %s", c.h, c.m, st.Next, c.next)
		}
		if c.next == "" && st.Next != nil {
			t.Errorf("%02d:%02d: unexpected next %+v", c.h, c.m, st.Next)
		}
	}
}

func TestReadyBy(t *testing.T) {
	got := ReadyBy(at(12, 44), 40) // 13:24 → 13:25
	if WAT(got) != "13:25" {
		t.Fatalf("got %s", WAT(got))
	}
	got = ReadyBy(at(12, 45), 40) // 13:25 exact
	if WAT(got) != "13:25" {
		t.Fatalf("got %s", WAT(got))
	}
}

func TestHHMMValid(t *testing.T) {
	for _, bad := range []string{"", "25:00", "12:60", "noon", "1200"} {
		if HHMM(bad).Valid() {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if !HHMM("23:59").Valid() {
		t.Error("23:59 should be valid")
	}
}
