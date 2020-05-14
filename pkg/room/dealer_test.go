package room

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/table"
	"testing"
)

func TestDealer_AddClient(t *testing.T) {
	d := NewDealer(&PitBoss{}, &table.Table{})
	c := NewClient(nil, nil, nil)
	c2 := NewClient(nil, nil, nil)

	d.AddClient(c)
	d.AddClient(c2)

	assert.False(t, d.RemoveClient(c))
	assert.True(t, d.RemoveClient(c2))
}
