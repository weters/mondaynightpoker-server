package passthepoop

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGameAction_ID(t *testing.T) {
	a := assert.New(t)
	a.Equal("stay", ActionStay.ID())
	a.Equal("trade", ActionTrade.ID())
	a.Equal("accept-trade", ActionAccept.ID())
	a.Equal("flip-king", ActionFlipKing.ID())
	a.Equal("block-trade", ActionBlockTrade.ID())
	a.Equal("go-to-deck", ActionGoToDeck.ID())
	a.Equal("draw-from-deck", ActionDrawFromDeck.ID())

	a.PanicsWithValue("invalid action -1", func() {
		_ = GameAction(-1).ID()
	})
}

func TestGameActionFromID(t *testing.T) {
	a := assert.New(t)

	for action := ActionStay; action <= ActionDrawFromDeck; action++ {
		fromID, err := GameActionFromID(action.ID())
		a.NoError(err)
		a.Equal(action, fromID)
	}

	fromID, err := GameActionFromID("bad-action")
	a.Equal(GameAction(0), fromID)
	a.EqualError(err, "no action with identifier bad-action")
}

func TestGameAction_MarshalJSON(t *testing.T) {
	b, err := json.Marshal(ActionGoToDeck)
	assert.NoError(t, err)
	assert.Equal(t, `{"id":"go-to-deck","name":"Go to Deck"}`, string(b))
}
