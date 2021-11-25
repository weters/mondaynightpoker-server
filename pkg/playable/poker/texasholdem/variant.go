package texasholdem

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Variant specifies the variant of Texas Hold'em
type Variant string

// Variant constants
const (
	Standard      Variant = "standard"
	Pineapple     Variant = "pineapple"
	LazyPineapple Variant = "lazy-pineapple"
)

var validVariants = map[Variant]bool{
	Standard:      true,
	Pineapple:     true,
	LazyPineapple: true,
}

// HoleCards returns the number of hole cards for the game
func (v Variant) HoleCards() int {
	if v == Standard {
		return 2
	}

	return 3
}

func (v Variant) String() string {
	switch v {
	case Standard:
		return "Standard"
	case Pineapple:
		return "Pineapple"
	case LazyPineapple:
		return "Lazy Pineapple"
	}

	panic(fmt.Sprintf("unknown variant: %s", string(v)))
}

// MarshalJSON encodes to JSON
func (v Variant) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		HoleCards int    `json:"holeCards"`
	}{
		ID:        string(v),
		Name:      v.String(),
		HoleCards: v.HoleCards(),
	})
}

// VariantFromString returns the variant from a string
func VariantFromString(s string) (Variant, error) {
	variant := Variant(strings.ToLower(s))
	if _, ok := validVariants[variant]; ok {
		return variant, nil
	}

	return "", fmt.Errorf("invalid variant: %s", s)
}
