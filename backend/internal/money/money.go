// Package money holds amounts as integer kobo. No float ever touches a price.
package money

import (
	"fmt"
	"strconv"
	"strings"
)

// Kobo is an amount in the smallest naira unit.
type Kobo int64

// FromNaira converts a whole-naira integer (what the vendor types) to kobo.
func FromNaira(n int64) Kobo { return Kobo(n * 100) }

// MulQty multiplies a unit price by a quantity.
func (k Kobo) MulQty(qty int) Kobo { return Kobo(int64(k) * int64(qty)) }

// Format renders "₦8,700" for whole naira and "₦8,700.50" otherwise.
func (k Kobo) Format() string {
	neg := k < 0
	if neg {
		k = -k
	}
	naira := int64(k) / 100
	rem := int64(k) % 100
	s := groupThousands(strconv.FormatInt(naira, 10))
	if rem != 0 {
		s = fmt.Sprintf("%s.%02d", s, rem)
	}
	if neg {
		return "-₦" + s
	}
	return "₦" + s
}

func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	pre := len(digits) % 3
	if pre > 0 {
		b.WriteString(digits[:pre])
	}
	for i := pre; i < len(digits); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}
