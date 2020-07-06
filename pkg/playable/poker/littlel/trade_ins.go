package littlel

import (
	"strconv"
	"strings"
)

// TradeIns are a slice of allowed trade-ins
type TradeIns []int

func (t TradeIns) String() string {
	s := make([]string, len(t))
	for i, val := range t {
		s[i] = strconv.Itoa(val)
	}

	return strings.Join(s, ", ")
}
