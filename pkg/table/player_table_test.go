package table

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPlayerTable_SetActive(t *testing.T) {
	p, tbl := playerAndTable()
	pt, err := p.GetPlayerTable(cbg, tbl)
	assert.NoError(t, err)
	assert.True(t, pt.Active)

	assert.NoError(t, pt.SetActive(cbg, true))
	assert.True(t, pt.Active)
	// refresh from db and ensure it's still true
	pt, _ = p.GetPlayerTable(cbg, tbl)
	assert.True(t, pt.Active)

	assert.NoError(t, pt.SetActive(cbg, false))
	assert.False(t, pt.Active)
	// refresh from db and ensure it's still false
	pt, _ = p.GetPlayerTable(cbg, tbl)
	assert.False(t, pt.Active)
	assert.True(t, pt.Updated.After(pt.Created))
}

func TestPlayerTable_SetIsTableAdmin(t *testing.T) {
	p, tbl := playerAndTable()
	pt, err := p.GetPlayerTable(cbg, tbl)
	assert.NoError(t, err)
	assert.True(t, pt.IsTableAdmin)

	assert.NoError(t, pt.SetIsTableAdmin(cbg, true))
	assert.True(t, pt.IsTableAdmin)
	// refresh from db and ensure it's still true
	pt, _ = p.GetPlayerTable(cbg, tbl)
	assert.True(t, pt.IsTableAdmin)

	assert.NoError(t, pt.SetIsTableAdmin(cbg, false))
	assert.False(t, pt.IsTableAdmin)
	// refresh from db and ensure it's still false
	pt, _ = p.GetPlayerTable(cbg, tbl)
	assert.False(t, pt.IsTableAdmin)
	assert.True(t, pt.Updated.After(pt.Created))
}

func TestPlayerTable_AdjustBalance(t *testing.T) {
	p1 := player()
	table, _ := p1.CreateTable(cbg, "my table")
	pt1, err := p1.GetPlayerTable(cbg, table)
	assert.NoError(t, err)
	assert.NotNil(t, pt1)

	err = pt1.AdjustBalance(cbg, 25, "won pot", nil)
	assert.NoError(t, err)

	err = pt1.AdjustBalance(cbg, 50, "won pot", nil)
	assert.NoError(t, err)

	pt1.Balance = -50
	err = pt1.AdjustBalance(cbg, 50, "won pot", nil)
	assert.Error(t, err)

	p2 := player()
	pt2, _ := p2.Join(cbg, table)
	err = pt2.AdjustBalance(cbg, -10, "lost post", nil)
	assert.NoError(t, err)

	pt1, _ = p1.GetPlayerTable(cbg, table)
	pt2, _ = p2.GetPlayerTable(cbg, table)
	assert.Equal(t, 75, pt1.Balance)
	assert.Equal(t, -10, pt2.Balance)
}
