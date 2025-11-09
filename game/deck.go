package game

import (
	"fmt"
	"math/rand"
	"time"
)

type Suit string
type Rank string

const (
	Hearts   Suit = "Hearts"
	Diamonds Suit = "Diamonds"
	Clubs    Suit = "Clubs"
	Spades   Suit = "Spades"
)

var Ranks = []Rank{
	"2", "3", "4", "5", "6", "7", "8", "9", "10",
	"J", "Q", "K", "A",
}

type Card struct {
	Suit Suit
	Rank Rank
}

type Deck []Card

func NewDeck() Deck { // Create a standard 52-card deck
	var deck Deck
	for _, suit := range []Suit{Hearts, Diamonds, Clubs, Spades} {
		for _, rank := range Ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank})
		}
	}
	return deck
}

func (d Deck) Shuffle() Deck { // Shuffle the deck
	rand.Seed(time.Now().UnixNano())
	shuffled := make(Deck, len(d))
	perm := rand.Perm(len(d))
	for i, v := range perm {
		shuffled[v] = d[i]
	}
	return shuffled
}

func (c Card) String() string { // String representation of a card
	return fmt.Sprintf("%s of %s", c.Rank, c.Suit)
}
