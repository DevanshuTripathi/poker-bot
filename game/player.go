package game

type Position string

const (
	Dealer     Position = "Dealer"
	SmallBlind Position = "SmallBlind"
	BigBlind   Position = "BigBlind"
	Normal     Position = "Normal"
)

type Action string

const (
	Fold  Action = "Fold"
	Check Action = "Check"
	Call  Action = "Call"
	Raise Action = "Raise"
	Bet   Action = "Bet"
)

type Player struct {
	Name       string
	Chips      int
	Hand       []Card
	Position   Position
	Active     bool
	CurrentBet int
	LastAction Action
}

func NewPlayer(name string, chips int) *Player {
	return &Player{
		Name:   name,
		Chips:  chips,
		Active: true,
	}
}

func (p *Player) Deal(cards []Card) {
	p.Hand = cards
}

func (p *Player) Fold() {
	p.Active = false
}

func (p *Player) PlaceBet(amount int) int {
	if amount > p.Chips {
		amount = p.Chips
	}
	p.Chips -= amount
	p.CurrentBet += amount
	return amount
}
