package bot

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/DevanshuTripathi/poker-bot/game"
)

func EvaluateHand(hand []game.Card, community []game.Card) float64 {
	if len(community) == 0 {
		return rankHoleCards(hand)
	} // Pre-flop evaluation by Steve Selbrede

	all := append([]game.Card{}, hand...)
	all = append(all, community...)

	rank := handRank(all)
	return rank
}

func rankHoleCards(hand []game.Card) float64 {
	rankOrder := map[game.Rank]int{
		"2": 2, "3": 3, "4": 4, "5": 5, "6": 6,
		"7": 7, "8": 8, "9": 9, "10": 10,
		"J": 11, "Q": 12, "K": 13, "A": 15,
	} // Ace is valued 15

	cardRanks := make([]int, 0)
	for _, card := range hand {
		cardRanks = append(cardRanks, rankOrder[card.Rank])
	} // Extract ranks from hand

	// Base rank calculation
	rank := 2.0*math.Max(float64(cardRanks[0]), float64(cardRanks[1])) + math.Min(float64(cardRanks[0]), float64(cardRanks[1]))

	if hand[0].Suit == hand[1].Suit {
		rank += 8.0 // suited bonus
	}

	if cardRanks[0] > 11 && cardRanks[1] > 11 {
		rank += 8.0 // both high cards bonus
	} else {
		// Connectedness bonus
		if math.Abs(float64(cardRanks[0]-cardRanks[1])) == 1 {
			rank += 8.0
		} else if math.Abs(float64(cardRanks[0]-cardRanks[1])) == 2 {
			rank += 6.0
		} else if math.Abs(float64(cardRanks[0]-cardRanks[1])) == 3 {
			rank += 4.0
		} else if math.Abs(float64(cardRanks[0]-cardRanks[1])) == 4 {
			rank += 2.0
		}
	}

	if cardRanks[0] == cardRanks[1] {
		rank += 28.0 // pair bonus
	}

	return rank / 100.0
}

func EstimateWinProbability(hand []game.Card, community []game.Card, opponents int, simulations int) float64 {
	winCount := 0

	// Monte Carlo simulations
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

	// Kickers and high cards
	straightHigh := 0
	flushHigh := 0
	quadsHigh := 0
	tripsHigh := 0
	pairHigh := 0

	rankOrder := map[game.Rank]int{
		"2": 2, "3": 3, "4": 4, "5": 5, "6": 6,
		"7": 7, "8": 8, "9": 9, "10": 10,
		"J": 11, "Q": 12, "K": 13, "A": 14,
	} // Ace is 14 for ranking

	uniqueVals := []int{}
	for r := range ranks {
		uniqueVals = append(uniqueVals, rankOrder[r])
	}
	sort.Ints(uniqueVals) // Sort for straight detection

	// Flush and Straight detection
	isFlush := false
	for suit, count := range suits {
		if count >= 5 {
			isFlush = true
			maxRank := 0
			for _, c := range cards {
				if c.Suit != suit {
					continue
				}
				if v, ok := rankOrder[c.Rank]; ok && v > maxRank {
					maxRank = v
				}
			}
			flushHigh = maxRank
			break
		}
	}

	isStraight := false
	count := 1
	for i := 1; i < len(uniqueVals); i++ {
		if uniqueVals[i] == uniqueVals[i-1]+1 {
			count++
			if count >= 5 {
				isStraight = true
				straightHigh = uniqueVals[i]
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
			straightHigh = 5
		}
	}

	pairCount := 0
	trips := false
	quads := false
	for rank, count := range ranks {
		if count == 2 {
			pairCount++
			pairHigh = int(math.Max(float64(pairHigh), float64(rankOrder[rank])))
		}
		if count == 3 {
			trips = true
			tripsHigh = int(math.Max(float64(tripsHigh), float64(rankOrder[rank])))
		}
		if count == 4 {
			quads = true
			quadsHigh = rankOrder[rank]
		}
	}

	if isStraight && isFlush {
		return 0.9 + float64(straightHigh)/1000.0 // straight flush
	}

	if quads {
		return 0.8 + float64(quadsHigh)/1000.0
	}

	// check full house first (trips + a pair)
	if trips && pairCount >= 1 {
		return 0.7 + float64(tripsHigh)/1000.0 // full house
	}

	if isFlush {
		return 0.6 + float64(flushHigh)/1000.0
	}

	if isStraight {
		return 0.5 + float64(straightHigh)/1000.0
	}

	if trips {
		return 0.4 + float64(tripsHigh)/1000.0
	}

	switch pairCount {
	case 2:
		return 0.3 + float64(pairHigh)/1000.0 // two pair
	case 1:
		return 0.2 + float64(pairHigh)/1000.0 // one pair
	default:
		return 0.1 + float64(uniqueVals[len(uniqueVals)-1])/1000.0 // high card
	}
}

func simulateHand(hand []game.Card, community []game.Card, opponents int) bool {
	deck := game.NewDeck().Shuffle()

	used := append(hand, community...)
	filtered := make([]game.Card, 0)

	// Remove used cards from deck
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

	// Deal remaining community cards if needed
	cardsNeeded := 5 - len(community)
	if cardsNeeded > 0 {
		community = append(community, filtered[:cardsNeeded]...)
		filtered = filtered[cardsNeeded:]
	}

	// Evaluate hands
	myRank := handRank(append(hand, community...))

	// Evaluate opponents' hands
	for i := 0; i < opponents; i++ {
		oppHand := filtered[:2]
		filtered = filtered[2:]
		oppRank := handRank(append(oppHand, community...))
		if oppRank > myRank { // opponent wins
			return false
		}
	}

	return true // win
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
