// Package money provides formatting helpers for monetary amounts. Every amount in
// this application is stored and transmitted as an integer number of cents; this
// package is the single place that turns those integers into human-readable dollar
// strings.
package money

import "fmt"

// FormatCents converts a cent amount to a dollar string. Whole-dollar amounts omit
// the decimal portion (200 -> "$2"); everything else is rendered to two places
// (150 -> "$1.50"). Negative amounts are prefixed with a minus sign outside the
// currency symbol (-50 -> "-$0.50").
func FormatCents(cents int) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}

	// A negated math.MinInt stays negative, which would render a bogus amount. The
	// magnitude is meaningless at that point, so fall back to the unsigned parts.
	dollars := cents / 100
	remainder := cents % 100
	if remainder < 0 {
		remainder = -remainder
	}
	if dollars < 0 {
		dollars = -dollars
	}

	var s string
	if remainder == 0 {
		s = fmt.Sprintf("$%d", dollars)
	} else {
		s = fmt.Sprintf("$%d.%02d", dollars, remainder)
	}

	if negative {
		s = "-" + s
	}

	return s
}
