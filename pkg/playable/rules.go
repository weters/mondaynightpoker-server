package playable

// RulesProvider is an optional interface that a Playable can implement to provide
// game rules to the client. This follows the same pattern as Tickable.
type RulesProvider interface {
	// Rules returns the rules for the current game configuration
	Rules() []RuleSection
}

// RuleSection is a titled section of game rules
type RuleSection struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
