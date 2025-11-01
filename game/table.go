package game

import "fmt"

const (
	SmallBlindAmount = 10
	BigBlindAmount   = 20
)

type Table struct {
	Players        []*Player
	Deck           Deck
	CommunityCards []Card
	Pot            int
}

func (t *Table) AssignPositions() {
	n := len(t.Players)
	for i := range t.Players {
		t.Players[i].Position = Normal
	}
	if n >= 2 {
		t.Players[0].Position = Dealer
		t.Players[1].Position = SmallBlind
		if n > 2 {
			t.Players[2].Position = BigBlind
		} else {
			t.Players[0].Position = BigBlind // heads-up rule: dealer is big blind
		}
	}
}

func NewTable(players []*Player) *Table {
	table := &Table{
		Players: players,
		Deck:    NewDeck().Shuffle(),
	}
	table.AssignPositions()
	return table
}

func (t *Table) DealHands() {
	for _, p := range t.Players {
		if len(t.Deck) >= 2 {
			p.Deal(t.Deck[:2])
			t.Deck = t.Deck[2:]
		}
	}
}

func (t *Table) PostBlinds() {
	for _, p := range t.Players {
		switch p.Position {
		case SmallBlind:
			p.PlaceBet(SmallBlindAmount)
		case BigBlind:
			p.PlaceBet(BigBlindAmount)
		}
	}
}

func (t *Table) DealFlop() {
	t.Deck = t.Deck[1:] // Burn one card
	t.CommunityCards = append(t.CommunityCards, t.Deck[:3]...)
	t.Deck = t.Deck[3:]
}

func (t *Table) DealTurn() {
	t.Deck = t.Deck[1:] // Burn one card
	t.CommunityCards = append(t.CommunityCards, t.Deck[0])
	t.Deck = t.Deck[1:]
}

func (t *Table) DealRiver() {
	t.Deck = t.Deck[1:] // Burn one card
	t.CommunityCards = append(t.CommunityCards, t.Deck[0])
	t.Deck = t.Deck[1:]
}

func (t *Table) ShowCommunity() {
	for _, card := range t.CommunityCards {
		fmt.Println(card)
	}
}

func (t *Table) PlayerAction(p *Player, action Action, amount int, currentBet int) {
	switch action {
	case Fold:
		p.Fold()
		fmt.Printf("%s folds\n", p.Name)
	case Call:
		toCall := currentBet - p.CurrentBet
		t.Pot += p.PlaceBet(toCall)
		fmt.Printf("%s calls %d\n", p.Name, toCall)
	case Raise:
		raiseAmount := amount
		t.Pot += p.PlaceBet(raiseAmount)
		fmt.Printf("%s raises by %d\n", p.Name, raiseAmount)
	case Check:
		p.LastAction = Check
		fmt.Printf("%s checks\n", p.Name)
	default:
		fmt.Printf("%s does nothing\n", p.Name)
	}
}

// func (t *Table) GetNextToBigBlind() int {
// 	for i, p := range t.Players {
// 		if p.Position == "Big Blind" {
// 			return (i + 1) % len(t.Players)
// 		}
// 	}
// 	return 0
// }

// func (t *Table) GetSmallBlindIndex() int {
// 	for i, p := range t.Players {
// 		if p.Position == "Small Blind" {
// 			return i
// 		}
// 	}
// 	return 0
// }
