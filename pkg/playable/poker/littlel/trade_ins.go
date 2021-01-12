package littlel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TradeIns are a slice of allowed trade-ins
type TradeIns struct {
	tradeInMap map[int]bool
	string     string
}

// NewTradeIns returns a new TradeIns object
func NewTradeIns(count []int, initialDeal int) (*TradeIns, error) {
	if len(count) == 0 {
		return &TradeIns{
			tradeInMap: map[int]bool{0: true},
			string:     "0",
		}, nil
	}

	// get unique list
	countMap := make(map[int]bool)
	for _, c := range count {
		if c > initialDeal {
			return nil, fmt.Errorf("invalid trade-in option: %d", c)
		}

		countMap[c] = true
	}

	// this only will work as long as trade-ins are < 10 (which
	// they should be)
	array := make([]string, 0, len(countMap))
	for val := range countMap {
		array = append(array, strconv.Itoa(val))
	}
	sort.Strings(array)

	return &TradeIns{
		tradeInMap: countMap,
		string:     strings.Join(array, ", "),
	}, nil
}

// MarshalJSON encodes to JSON
func (t TradeIns) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.tradeInMap)
}

// CanTrade returns true if the count is an acceptable trade
func (t TradeIns) CanTrade(count int) bool {
	_, ok := t.tradeInMap[count]
	return ok
}

func (t TradeIns) String() string {
	return t.string
}
