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

// TestNameForStoredGameType_RoundTripsEveryFactory pins the contract the stored
// game_type resolution depends on: every factory's display name must map back to
// that same factory. A new game whose DisplayName does not land in its own group
// would otherwise have its logs silently parsed by the wrong parser.
func TestNameForStoredGameType_RoundTripsEveryFactory(t *testing.T) {
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			factory, err := Get(name)
			require.NoError(t, err)

			got, err := NameForStoredGameType(factory.DisplayName())
			require.NoError(t, err)
			assert.Equal(t, name, got)
		})
	}
}

// TestNameForStoredGameType_RealDisplayNames covers the game_type values actually
// written to the column, which carry the options the game was created with rather
// than the bare display name.
func TestNameForStoredGameType_RealDisplayNames(t *testing.T) {
	for stored, want := range map[string]string{
		"Bourré":                        "bourre",
		"Bourré (ante: 50)":             "bourre",
		"4-Card Little L (trade: 0, 2)": "little-l",
		"Texas Hold'em":                 "texas-hold-em",
		"Pineapple":                     "texas-hold-em",
		"Lazy Pineapple":                "texas-hold-em",
		"Acey Deucey":                   "acey-deucey",
		"Pass the Poop: Diarrhea":       "pass-the-poop",
		"Guts (2 card)":                 "guts",
		"Seven-Card Stud":               "seven-card",
		"Baseball":                      "seven-card",
		"Follow the Queen":              "seven-card",
		"Coupons and Clippings":         "seven-card",
	} {
		got, err := NameForStoredGameType(stored)
		require.NoError(t, err, "stored game type %q", stored)
		assert.Equal(t, want, got, "stored game type %q", stored)
	}
}

func TestNameForStoredGameType_Unknown(t *testing.T) {
	name, err := NameForStoredGameType("Go Fish")
	assert.Empty(t, name)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no factory for stored game type: Go Fish")
}

func TestParseStoredGameLog(t *testing.T) {
	hand, err := ParseStoredGameLog("Bourré (ante: 50)", json.RawMessage(`{"ante":50}`))
	require.NoError(t, err)
	require.NotNil(t, hand)

	assert.Equal(t, "bourre", hand.GameType)
	assert.Equal(t, 50, hand.AnteCents)
}

func TestParseGameLog_MalformedPayload(t *testing.T) {
	hand, err := ParseGameLog("guts", json.RawMessage(`{"rounds":"nope"}`))
	assert.Nil(t, hand)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not decode guts log")
}
