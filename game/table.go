package game

import "fmt"

const (
	SmallBlindAmount = 1 // Small blind amount in chips
	BigBlindAmount   = 2 // Big blind amount in chips
)

type Table struct {
	Players        []*Player
	Deck           Deck
	CommunityCards []Card
	Pot            int
	CurrentAction  Action
}

func (t *Table) AssignPositions() { // Assign player positions: Dealer, Small Blind, Big Blind
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

func NewTable(players []*Player) *Table { // Create a new table with players and shuffled deck
	table := &Table{
		Players: players,
		Deck:    NewDeck().Shuffle(),
	}
	table.AssignPositions()
	return table
}

func (t *Table) DealHands() { // Deal two cards to each player
	for _, p := range t.Players {
		if len(t.Deck) >= 2 {
			p.Deal(t.Deck[:2])
			t.Deck = t.Deck[2:]
		}
	}
}

func (t *Table) PostBlinds() { // Post small and big blinds
	for _, p := range t.Players {
		switch p.Position {
		case SmallBlind:
			p.PlaceBet(SmallBlindAmount)
		case BigBlind:
			p.PlaceBet(BigBlindAmount)
		}
	}
}

func (t *Table) DealFlop() { // Deal the flop (3 community cards)
	t.Deck = t.Deck[1:] // Burn one card
	t.CommunityCards = append(t.CommunityCards, t.Deck[:3]...)
	t.Deck = t.Deck[3:]
}

func (t *Table) DealTurn() { // Deal the turn (4th community card)
	t.Deck = t.Deck[1:] // Burn one card
	t.CommunityCards = append(t.CommunityCards, t.Deck[0])
	t.Deck = t.Deck[1:]
}

func (t *Table) DealRiver() { // Deal the river (5th community card)
	t.Deck = t.Deck[1:] // Burn one card
	t.CommunityCards = append(t.CommunityCards, t.Deck[0])
	t.Deck = t.Deck[1:]
}

func (t *Table) ShowCommunity() { // Print community cards
	for _, card := range t.CommunityCards {
		fmt.Println(card)
	}
}

func (t *Table) PlayerAction(p *Player, action Action, amount int, currentBet int) { // Process player action
	switch action {
	case Fold:
		p.Fold()
	case Call:
		toCall := currentBet - p.CurrentBet
		t.Pot += p.PlaceBet(toCall)
	case Raise:
		raiseAmount := amount
		t.Pot += p.PlaceBet(raiseAmount)
	case Check:
		p.LastAction = Check
	default:
	}
}

func (t *Table) GetHighestBet() int { // Get the highest current bet among active players
	maxBet := 0
	for _, p := range t.Players {
		if p.CurrentBet > maxBet {
			maxBet = p.CurrentBet
		}
	}
	return maxBet
}

func (t *Table) GetActivePlayers() []*Player { // Get list of active players
	active := []*Player{}
	for _, p := range t.Players {
		if p.Active {
			active = append(active, p)
		}
	}
	return active
}

func (t *Table) IsBettingRoundOver() bool { // Check if betting round is over
	maxBet := t.GetHighestBet()
	for _, p := range t.GetActivePlayers() {
		if p.CurrentBet < maxBet {
			// Someone is still active but hasn't matched the bet
			return false
		}
	}
	return true
}

func (t *Table) CollectBets() { // Collect bets into the pot and reset current bets
	potThisRound := 0
	for _, p := range t.Players {
		potThisRound += p.CurrentBet
		p.CurrentBet = 0
	}
	t.Pot += potThisRound
}

func (t *Table) ResetForNewHand() { // Reset table state for a new hand
	t.CommunityCards = []Card{}
	t.Pot = 0
	t.Deck = NewDeck().Shuffle()

	// Rotate players to change blinds/dealer
	t.Players = append(t.Players[1:], t.Players[0])
	t.AssignPositions()

	for _, p := range t.Players {
		p.Reset()
	}
}
