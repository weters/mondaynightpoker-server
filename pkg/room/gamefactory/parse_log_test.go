package gamefactory

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseGameLog_EveryGameTypeParses is an exhaustiveness check: every game type
// registered in the factory map must be able to decode its own persisted log. A
// new game added without a parser fails here rather than at runtime on the first
// analytics call.
//
// The payload used per game type is the minimum well-formed shape for that game,
// which is enough to prove the parser is wired up and returns a hand tagged with
// the right game type. The parsers' real behavior is covered by the round-trip
// tests in each game package, which marshal the live log struct.
func TestParseGameLog_EveryGameTypeParses(t *testing.T) {
	// Acey Deucey persists a bare array; every other game persists an object.
	payloads := map[string]string{
		"acey-deucey": `[]`,
	}

	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			payload, ok := payloads[name]
			if !ok {
				payload = `{}`
			}

			hand, err := ParseGameLog(name, json.RawMessage(payload))
			require.NoError(t, err)
			require.NotNil(t, hand)
			assert.Equal(t, name, hand.GameType, "the dispatcher tags the hand")
		})
	}
}

// TestParseGameLog_NoLog covers games recorded but never finished. A game row is
// created when the game starts and only gets its data payload when it ends, so a
// terminated or in-progress game legitimately has nothing to parse.
func TestParseGameLog_NoLog(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage(`null`)} {
		hand, err := ParseGameLog("bourre", raw)
		assert.NoError(t, err)
		assert.Nil(t, hand)
	}
}

func TestParseGameLog_UnknownGameType(t *testing.T) {
	hand, err := ParseGameLog("go-fish", json.RawMessage(`{}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no factory with name: go-fish")
}

func TestParseGameLog_MalformedPayload(t *testing.T) {
	hand, err := ParseGameLog("guts", json.RawMessage(`{"rounds":"nope"}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode guts log")
}
