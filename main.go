package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/DevanshuTripathi/poker-bot/bot"
	"github.com/DevanshuTripathi/poker-bot/game"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("♠️  Poker Hand Input CLI ♣️")
	fmt.Println("Enter player names separated by commas (excluding bot):")
	namesInput, _ := reader.ReadString('\n')
	names := strings.Split(strings.TrimSpace(namesInput), ",")

	var players []*game.Player
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		player := game.NewPlayer(name, 1000)
		players = append(players, player)
	}

	// 👇 Add bot automatically
	bot := game.NewPlayer("BOT", 1000)
	players = append(players, bot)

	if len(players) < 2 {
		fmt.Println("Need at least 2 players to play poker!")
		return
	}

	table := game.NewTable(players)
	table.PostBlinds()

	fmt.Println("\n💺 Positions:")
	for _, p := range table.Players {
		fmt.Printf("%s - %s (Chips: %d, Bet: %d)\n", p.Name, p.Position, p.Chips, p.CurrentBet)
	}
	fmt.Printf("\n💰 Pot after blinds: %d\n\n", table.Pot)

	// 👇 Ask for bot’s hole cards
	fmt.Println("🧠 Enter BOT’s hole cards (e.g. 'Ah Kd'):")
	botInput, _ := reader.ReadString('\n')
	botCards := strings.Fields(strings.TrimSpace(botInput))
	bot.Hand = parseManualCards(botCards)

	fmt.Printf("BOT’s hand: %v\n", bot.Hand)

	fmt.Println("\n--- STARTING HAND SIMULATION ---")

	runBettingRound(reader, table, "Preflop", bot)

	manualCommunityCards(reader, table, "Flop", 3)
	runBettingRound(reader, table, "Flop", bot)

	manualCommunityCards(reader, table, "Turn", 1)
	runBettingRound(reader, table, "Turn", bot)

	manualCommunityCards(reader, table, "River", 1)
	runBettingRound(reader, table, "River", bot)

	fmt.Println("\n🏁 SHOWDOWN")
	for _, p := range table.Players {
		if p.Active {
			if p.Name == "BOT" {
				fmt.Printf("%s's hand: %v\n", p.Name, p.Hand)
			} else {
				fmt.Printf("%s is still in the hand.\n", p.Name)
			}
		}
	}
	fmt.Printf("\n💰 Final pot: %d\n", table.Pot)
}

// ================== ROUND LOGIC ===================

func runBettingRound(reader *bufio.Reader, table *game.Table, stage string, ai *game.Player) {
	fmt.Printf("\n=== %s Betting ===\n", strings.ToUpper(stage))
	currentBet := game.BigBlindAmount
	allMatched := false

	for !allMatched {
		allMatched = true
		for _, p := range table.Players {
			if !p.Active {
				continue
			}
			if p.CurrentBet < currentBet {
				allMatched = false
			}

			if p.Name == "BOT" {
				fmt.Println("\n🤖 BOT’s turn — (you can later replace this with logic)")
				strength := bot.EvaluateHand(ai.Hand, table.CommunityCards)
				winProb := bot.EstimateWinProbability(ai.Hand, table.CommunityCards, len(table.Players)-1, 500)
				fmt.Printf("BOT Hand Strength: %.2f, Estimated Win Probability: %.2f%%\n", strength, winProb*100)
				fmt.Print("Enter BOT action [fold/call/check/raise]: ")
			} else {
				fmt.Printf("\n👉 %s's turn (Chips: %d, Bet: %d)\n", p.Name, p.Chips, p.CurrentBet)
				fmt.Print("Enter action [fold/call/check/raise]: ")
			}

			input, _ := reader.ReadString('\n')
			action := strings.TrimSpace(strings.ToLower(input))

			switch action {
			case "fold":
				table.PlayerAction(p, game.Fold, 0, currentBet)
			case "call":
				table.PlayerAction(p, game.Call, 0, currentBet)
			case "check":
				table.PlayerAction(p, game.Check, 0, currentBet)
			case "raise":
				fmt.Print("Enter raise amount: ")
				raiseInput, _ := reader.ReadString('\n')
				raiseAmount, _ := strconv.Atoi(strings.TrimSpace(raiseInput))
				table.PlayerAction(p, game.Raise, raiseAmount, currentBet)
				currentBet += raiseAmount
			default:
				fmt.Println("Invalid input, skipping.")
			}
		}

		allMatched = true
		for _, p := range table.Players {
			if p.Active && p.CurrentBet < currentBet {
				allMatched = false
			}
		}
	}
}

// ================== COMMUNITY CARD INPUT ===================

func manualCommunityCards(reader *bufio.Reader, table *game.Table, stage string, count int) {
	fmt.Printf("\n🃏 Enter %d %s cards (e.g. 'Ah Kd Qs'):\n", count, stage)
	input, _ := reader.ReadString('\n')
	cards := strings.Fields(strings.TrimSpace(input))
	table.CommunityCards = append(table.CommunityCards, parseManualCards(cards)...)
	fmt.Printf("%s: %v\n", stage, table.CommunityCards)
}

// ================== HELPERS ===================

func parseManualCards(inputs []string) []game.Card {
	var cards []game.Card
	for _, c := range inputs {
		if len(c) < 2 {
			continue
		}
		rank := string(c[:len(c)-1])
		suitChar := c[len(c)-1]
		suit := suitFromChar(suitChar)
		card := game.Card{Rank: game.Rank(rank), Suit: suit}
		cards = append(cards, card)
	}
	return cards
}

func suitFromChar(c byte) game.Suit {
	switch c {
	case 'h', 'H':
		return game.Hearts
	case 'd', 'D':
		return game.Diamonds
	case 'c', 'C':
		return game.Clubs
	case 's', 'S':
		return game.Spades
	default:
		return game.Hearts
	}
}
