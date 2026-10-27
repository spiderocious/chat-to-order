package mcpserver

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestDecodeCreateOrderAliases(t *testing.T) {
	cases := []string{
		`{"customer_name":"Tobi","items":[{"item_id":"jollof","qty":2}],"fulfilment":"delivery","zone_id":"surulere","address":"22 Bode Thomas"}`,
		`{"customer":"Tobi","items":[{"id":"jollof","quantity":2}],"fulfillment":"Delivery","zone":"Surulere","delivery_address":"22 Bode Thomas"}`,
		`{"name":"Tobi","order":[{"item":"jollof","count":2}],"type":"deliver","area":"surulere","address":"22 Bode Thomas"}`,
		`{"customer_name":"Tobi","items":{"jollof":2},"address":"22 Bode Thomas","zone_id":"surulere"}`,
	}
	for _, c := range cases {
		in := decodeCreateOrder(decode(t, c))
		if in.CustomerName != "Tobi" || in.Fulfilment != "delivery" || in.ZoneID != "surulere" || in.Address != "22 Bode Thomas" {
			t.Errorf("%s → %+v", c, in)
		}
		if len(in.Items) != 1 || in.Items[0].ItemID != "jollof" || in.Items[0].Qty != 2 {
			t.Errorf("%s → items %+v", c, in.Items)
		}
	}
	in := decodeCreateOrder(decode(t, `{"customer_name":"Ada","items":["water"],"fulfilment":"pick up"}`))
	if in.Fulfilment != "pickup" || in.Items[0].Qty != 1 {
		t.Errorf("pickup/strings: %+v", in)
	}
}

func TestDecodeReference(t *testing.T) {
	for _, c := range []string{`{"reference":"IBK-0231-a1b2"}`, `{"ref":"IBK-0231-a1b2"}`, `{"order_no":"IBK-0231-a1b2"}`} {
		if got := decodeReference(decode(t, c)); got != "IBK-0231-a1b2" {
			t.Errorf("%s → %q", c, got)
		}
	}
}
