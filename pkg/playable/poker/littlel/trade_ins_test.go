package littlel

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTradeIns_String(t *testing.T) {
	tradeIns, _ := NewTradeIns([]int{4, 4, 0, 2, 2})
	assert.Equal(t, "0, 2, 4", tradeIns.String())
}

func TestTradeIns_CanTrade(t *testing.T) {
	tradeIns, _ := NewTradeIns([]int{4, 4, 0, 2, 2})
	assert.True(t, tradeIns.CanTrade(0))
	assert.False(t, tradeIns.CanTrade(1))
	assert.True(t, tradeIns.CanTrade(2))
	assert.False(t, tradeIns.CanTrade(3))
	assert.True(t, tradeIns.CanTrade(4))
}

func TestTradeIns_MarshalJSON(t *testing.T) {
	tradeIns, _ := NewTradeIns([]int{2})
	b, err := json.Marshal(tradeIns)
	assert.NoError(t, err)
	assert.Equal(t, `{"2":true}`, string(b))
}
