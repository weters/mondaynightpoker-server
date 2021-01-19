package playable

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSimpleLogMessage(t *testing.T) {
	before := time.Now()
	lm := SimpleLogMessage(0, "test %d", 5)
	assert.Equal(t, "test 5", lm.Message)
	assert.Nil(t, lm.PlayerIDs)
	assert.True(t, before.Before(lm.Time))
	assert.True(t, time.Now().After(lm.Time))
	assert.Nil(t, lm.Cards)
}

func TestSimpleLogMessage_withPlayerID(t *testing.T) {
	lm := SimpleLogMessage(1, "test %d", 4)
	assert.Equal(t, "test 4", lm.Message)
	assert.Equal(t, []int64{1}, lm.PlayerIDs)
}

func TestSimpleLogMessageSlice(t *testing.T) {
	lms := SimpleLogMessageSlice(0, "test %d", 38)
	assert.Equal(t, 1, len(lms))
	assert.Equal(t, "test 38", lms[0].Message)
}

func TestAdditionalData_GetIntSlice(t *testing.T) {
	a := assert.New(t)

	ad := AdditionalData{"ints": []float64{1, 2, 3}}
	val, ok := ad.GetIntSlice("ints")
	a.True(ok)
	a.Equal(val, []int{1, 2, 3})

	var data AdditionalData
	_ = json.Unmarshal([]byte(`{"ints":[1,2,3,4]}`), &data)
	val, ok = data.GetIntSlice("ints")
	a.True(ok)
	a.Equal(val, []int{1, 2, 3, 4})

	ad = AdditionalData{"ints": []string{"1", "2"}}
	val, ok = ad.GetIntSlice("ints")
	a.False(ok)
	a.Nil(val)
}
