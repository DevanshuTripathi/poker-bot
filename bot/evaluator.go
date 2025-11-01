package bot

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/DevanshuTripathi/poker-bot/game"
)

func EvaluateHand(hand []game.Card, community []game.Card) float64 {
	all := append([]game.Card{}, hand...)
	all = append(all, community...)

	rank := handRank(all)
	return rank
}

func EstimateWinProbability(hand []game.Card, community []game.Card, opponents int, simulations int) float64 {
	winCount := 0
	fmt.Print(hand)
	fmt.Print(community)

	for i := 0; i < simulations; i++ {
		simCommunity := append([]game.Card{}, community...)
		if simulateHand(hand, simCommunity, opponents) {
			winCount++
		}
	}

	return float64(winCount) / float64(simulations)
}

func handRank(cards []game.Card) float64 {

	ranks := make(map[game.Rank]int)
	suits := make(map[game.Suit]int)

	for _, card := range cards {
		ranks[card.Rank]++
		suits[card.Suit]++
	}

	isFlush := false
	for _, count := range suits {
		if count >= 5 {
			isFlush = true
			break
		}
	}

	rankOrder := map[game.Rank]int{
		"2": 2, "3": 3, "4": 4, "5": 5, "6": 6,
		"7": 7, "8": 8, "9": 9, "10": 10,
		"J": 11, "Q": 12, "K": 13, "A": 14,
	}

	uniqueVals := []int{}
	for r := range ranks {
		uniqueVals = append(uniqueVals, rankOrder[r])
	}
	sort.Ints(uniqueVals)

	isStraight := false
	count := 1
	for i := 1; i < len(uniqueVals); i++ {
		if uniqueVals[i] == uniqueVals[i-1]+1 {
			count++
			if count >= 5 {
				isStraight = true
				break
			}
		} else {
			count = 1
		}
	}

	// Ace-low straight (A-2-3-4-5)
	if !isStraight {
		hasAce := false
		for r := range ranks {
			if r == "A" {
				hasAce = true
				break
			}
		}
		if hasAce && ranks["2"] > 0 && ranks["3"] > 0 && ranks["4"] > 0 && ranks["5"] > 0 {
			isStraight = true
		}
	}

	pairCount := 0
	trips := false
	quads := false
	for _, count := range ranks {
		if count == 2 {
			pairCount++
		}
		if count == 3 {
			trips = true
		}
		if count == 4 {
			quads = true
		}
	}

	if isStraight && isFlush {
		return 0.9 // straight flush
	}

	if quads {
		return 0.8
	}

	if trips {
		return 0.4
	}

	if trips && pairCount >= 1 {
		return 0.7 // full house
	}

	if isFlush {
		return 0.6
	}

	if isStraight {
		return 0.5
	}

	switch pairCount {
	case 2:
		return 0.3 // two pair
	case 1:
		return 0.2 // one pair
	default:
		return 0.1 // high card
	}
}

func simulateHand(hand []game.Card, community []game.Card, opponents int) bool {
	deck := game.NewDeck().Shuffle()

	used := append(hand, community...)
	filtered := make([]game.Card, 0)

	for _, c := range deck {
		skip := false
		for _, u := range used {
			if c.Rank == u.Rank && c.Suit == u.Suit {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, c)
		}
	}

	cardsNeeded := 5 - len(community)
	if cardsNeeded > 0 {
		community = append(community, filtered[:cardsNeeded]...)
		filtered = filtered[cardsNeeded:]
	}

	myRank := handRank(append(hand, community...))

	for i := 0; i < opponents; i++ {
		oppHand := filtered[:2]
		filtered = filtered[2:]
		oppRank := handRank(append(oppHand, community...))
		if oppRank > myRank {
			return false
		}
	}

	return true
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
