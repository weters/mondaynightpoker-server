package potmanager

import "encoding/json"

// Pot represents an ordered lists of pots
type Pot struct {
	Amount            int
	AllInParticipants []Participant
}

type potJSON struct {
	Amount            int     `json:"amount"`
	AllInParticipants []int64 `json:"allInParticipants"`
}

// MarshalJSON provides custom marshalling
func (p Pot) MarshalJSON() ([]byte, error) {
	ids := make([]int64, len(p.AllInParticipants))
	for i, p := range p.AllInParticipants {
		ids[i] = p.ID()
	}

	return json.Marshal(potJSON{
		Amount:            p.Amount,
		AllInParticipants: ids,
	})
}

// Pots is a collection of pots
type Pots []*Pot

// Total returns the combined total of all pots
func (p Pots) Total() int {
	total := 0
	for _, pot := range p {
		total += pot.Amount
	}

	return total
}
