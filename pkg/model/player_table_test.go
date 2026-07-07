package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlayerTable_Save(t *testing.T) {
	p1, tbl := playerAndTable()
	pt1, err := p1.GetPlayerTable(cbg, tbl)
	assert.NoError(t, err)
	assert.True(t, pt1.IsTableAdmin)
	assert.Equal(t, 2000, pt1.TableStake)

	p2 := player()
	pt2, err := p2.Join(cbg, tbl)
	assert.NoError(t, err)
	assert.True(t, pt2.Active)
	assert.Equal(t, 2000, pt2.TableStake)
	assert.False(t, pt2.IsTableAdmin)
	assert.False(t, pt2.CanStart)
	assert.False(t, pt2.CanRestart)
	assert.False(t, pt2.CanTerminate)
	assert.False(t, pt2.IsBlocked)

	pt2.Active = false
	pt2.TableStake = 3000
	pt2.IsTableAdmin = true
	pt2.CanStart = true
	pt2.CanRestart = true
	pt2.CanTerminate = true
	pt2.IsBlocked = true
	assert.NoError(t, pt2.Save(cbg))

	pt2, err = p2.GetPlayerTable(cbg, tbl)
	assert.NoError(t, err)
	assert.False(t, pt2.Active)
	assert.Equal(t, 3000, pt2.TableStake)
	assert.True(t, pt2.IsTableAdmin)
	assert.True(t, pt2.CanStart)
	assert.True(t, pt2.CanRestart)
	assert.True(t, pt2.CanTerminate)
	assert.True(t, pt2.IsBlocked)
}

func TestPlayerTable_IsPlaying(t *testing.T) {
	pt := &PlayerTable{
		Active:    true,
		IsBlocked: false,
	}
	assert.True(t, pt.IsPlaying())

	pt.Active = false
	assert.False(t, pt.IsPlaying())

	pt.Active = true
	pt.IsBlocked = true
	assert.False(t, pt.IsPlaying())
}

func TestPlayerTable_accessors(t *testing.T) {
	pt := &PlayerTable{PlayerID: 1, TableStake: 2, Balance: 0}
	assert.Equal(t, int64(1), pt.GetPlayerID())
	assert.Equal(t, 2, pt.GetTableStake())

	pt.Balance = 10
	assert.Equal(t, 10, pt.GetTableStake())
}
