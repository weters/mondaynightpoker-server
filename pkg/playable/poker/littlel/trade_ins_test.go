package littlel

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTradeIns_String(t *testing.T) {
	tradeIns, _ := NewTradeIns([]int{4, 4, 0, 2, 2}, 4)
	assert.Equal(t, "0, 2, 4", tradeIns.String())
}

func TestTradeIns_validation(t *testing.T) {
	a := assert.New(t)
	tradeIns, err := NewTradeIns([]int{0, 1, 2, 3, 4}, 3)
	a.Nil(tradeIns)
	a.EqualError(err, "invalid trade-in option: 4")

	tradeIns, err = NewTradeIns([]int{0, 1, 2, 3}, 3)
	a.NotNil(tradeIns)
	a.NoError(err)

	tradeIns, err = NewTradeIns([]int{0, 1, 2, 3, 4, 5}, 4)
	a.Nil(tradeIns)
	a.EqualError(err, "invalid trade-in option: 5")

	tradeIns, err = NewTradeIns([]int{0, 1, 2, 3, 4}, 4)
	a.NotNil(tradeIns)
	a.NoError(err)
}

func TestTradeIns_CanTrade(t *testing.T) {
	tradeIns, _ := NewTradeIns([]int{4, 4, 0, 2, 2}, 4)
	assert.True(t, tradeIns.CanTrade(0))
	assert.False(t, tradeIns.CanTrade(1))
	assert.True(t, tradeIns.CanTrade(2))
	assert.False(t, tradeIns.CanTrade(3))
	assert.True(t, tradeIns.CanTrade(4))
}

func TestTradeIns_MarshalJSON(t *testing.T) {
	tradeIns, _ := NewTradeIns([]int{2}, 4)
	b, err := json.Marshal(tradeIns)
	assert.NoError(t, err)
	assert.Equal(t, `{"2":true}`, string(b))
}
