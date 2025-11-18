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

var qAgent *bot.QAgent     // Old Q-learning agent (not used in this version)
var dqnAgent *bot.DQNAgent // DQN agent

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("♠️ DQN Poker Bot CLI ♣️")

	var numPlayers int
	for {
		fmt.Print("Enter number of players for this game (e.g., 2, 5, 6): ")
		input, _ := reader.ReadString('\n')
		num, err := strconv.Atoi(strings.TrimSpace(input))
		if err == nil && num >= 2 && num <= 9 {
			numPlayers = num
			break
		}
		fmt.Println("Invalid input. Please enter a number between 2 and 9.")
	}

	modelDir := "models"                                                                      // Directory to save/load models
	os.MkdirAll(modelDir, os.ModePerm)                                                        // makes the folder if it doesn’t exist
	modelFile := fmt.Sprintf("%s/dqn-model-%dp-validation-winProb.gob", modelDir, numPlayers) // Model file path
	fmt.Printf("Using model file: %s\n", modelFile)

	dqnAgent = bot.NewDQNAgent(
		0.0001,   // learningRate
		0.95,     // gamma
		1.0,      // epsilon
		0.01,     // minEpsilon
		0.000001, // decayRate (this is the "Deep Sleep 3.0" rate)
	) // Initialize DQN agent

	// Try loading an existing model
	if _, err := os.Stat(modelFile); err == nil {
		fmt.Println("📂 Found saved model! Loading it...")
		if err := dqnAgent.LoadModel(modelFile); err != nil {
			fmt.Println("⚠️ Error loading model:", err)
		} else {
			fmt.Println("✅ Model loaded successfully!")
		}
	} else {
		fmt.Println("🆕 No saved model found. Training from scratch...")
		dqnAgent = bot.TrainPokerBot(50000, numPlayers) // Train for 500,000 hands
		fmt.Println("Training complete!")

		fmt.Println("💾 Saving trained model...")
		if err := dqnAgent.SaveModel(modelFile); err != nil {
			fmt.Println("❌ Failed to save model:", err)
		} else {
			fmt.Println("✅ Model saved successfully!")
		}
	}

	fmt.Println("Setting Epsilon to 0 for exploitation mode.")
	dqnAgent.SetEpsilon(0.0) // Set epsilon to 0 for exploitation (no exploration)

	fmt.Printf("\n--- Starting Live %d-Player Game ---\n", numPlayers)

	var players []*game.Player
	numHumans := numPlayers - 1

	if numHumans > 0 {
		fmt.Printf("Enter %d human player name(s) separated by commas:\n", numHumans)
		namesInput, _ := reader.ReadString('\n')
		names := strings.Split(strings.TrimSpace(namesInput), ",")

		for i, name := range names {
			if i >= numHumans {
				break
			}
			name = strings.TrimSpace(name)
			if name == "" {
				name = fmt.Sprintf("Human_%d", i+1)
			}
			player := game.NewPlayer(name, 1000)
			players = append(players, player)
		}
	}

	// Add bot automatically
	bot := game.NewPlayer("BOT", 1000)
	players = append(players, bot)

	if len(players) != numPlayers {
		fmt.Printf("Warning: Game has %d players, but bot was trained for %d.\n", len(players), numPlayers)
	}

	for {
		fmt.Println("Enter the dealer position name:")
		dealerInput, _ := reader.ReadString('\n')
		dealerName := strings.TrimSpace(dealerInput)

		dealerFound := false
		for i, p := range players {
			if p.Name == dealerName {
				// Move dealer to index 0
				temp := players[:i]
				newPlayers := append(players[i:], temp...)
				players = newPlayers
				dealerFound = true
				break
			}
		}
		if !dealerFound {
			fmt.Println("Dealer not found, using existing order.")
		}

		fmt.Println("\n🚩 Players at the table:")
		for _, p := range players {
			fmt.Printf("- %s (Chips: %d)\n", p.Name, p.Chips)
		}

		table := game.NewTable(players)
		table.PostBlinds()

		fmt.Println("\n💺 Positions:")
		for _, p := range table.Players {
			// table.Pot += p.CurrentBet
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
		if len(table.GetActivePlayers()) <= 1 {
			goto EndHand
		}

		manualCommunityCards(reader, table, "Flop", 3)
		runBettingRound(reader, table, "Flop", bot)
		if len(table.GetActivePlayers()) <= 1 {
			goto EndHand
		}

		manualCommunityCards(reader, table, "Turn", 1)
		runBettingRound(reader, table, "Turn", bot)
		if len(table.GetActivePlayers()) <= 1 {
			goto EndHand
		}

		manualCommunityCards(reader, table, "River", 1)
		runBettingRound(reader, table, "River", bot)

	EndHand:
		fmt.Println("\n🏁 SHOWDOWN / HAND END 🏁")

		for _, p := range table.Players {
			if p.Active {
				if p.Name == "BOT" {
					fmt.Printf("%s's hand: %v\n", p.Name, p.Hand)
				} else {
					// Ask for human player's hand if they are still in
					fmt.Printf("Enter %s's hole cards (e.g. 'Qc Jd'):\n", p.Name)
					humanInput, _ := reader.ReadString('\n')
					humanCards := strings.Fields(strings.TrimSpace(humanInput))
					p.Hand = parseManualCards(humanCards)
					fmt.Printf("%s's hand: %v\n", p.Name, p.Hand)
				}
			}
		}
		fmt.Printf("\n💰 Final pot: %d\n", table.Pot)

		// award pot to winner or split
		fmt.Println("\nEnter winner name to award pot (or type 'split'):")
		winnerInput, _ := reader.ReadString('\n')
		winner := strings.TrimSpace(winnerInput)

		if strings.ToLower(winner) == "split" {
			// split among active players
			active := []*game.Player{}
			for _, p := range table.Players {
				if p.Active {
					active = append(active, p)
				}
			}
			if len(active) > 0 {
				share := table.Pot / len(active)
				for _, p := range active {
					p.Chips += share
				}
				fmt.Printf("Pot split: each active player receives %d\n", share)
			}
		} else {
			awarded := false
			for _, p := range table.Players {
				if p.Name == winner {
					p.Chips += table.Pot
					awarded = true
					fmt.Printf("%s awarded the pot of %d\n", p.Name, table.Pot)
					break
				}
			}
			if !awarded {
				fmt.Println("Winner not found, pot remains unawarded.")
			}
		}

		// reset for next round: clear pot, players' bets/hands/active flags
		table.Pot = 0
		table.CommunityCards = nil
		for _, p := range table.Players {
			p.Reset()
		}

		// loop continues to next round (positions will be reassigned at NewTable)
	}
}

func getBettingContext(table *game.Table, bot *game.Player) (int, int) {
	maxBet := table.GetHighestBet()
	toCall := maxBet - bot.CurrentBet

	if toCall < 0 {
		toCall = 0
	}

	return toCall, table.Pot
}

// ================== ROUND LOGIC ===================

func runBettingRound(reader *bufio.Reader, table *game.Table, stage string, ai *game.Player) {
	fmt.Printf("\n=== %s Betting ===\n", strings.ToUpper(stage))
	currentBet := table.GetHighestBet()

	playersToAct := make([]*game.Player, 0, len(table.Players))
	startIdx := 0

	if stage == "Preflop" {
		if len(table.Players) > 2 {
			startIdx = 3 % len(table.Players) // UTG
		} else {
			startIdx = 0 % len(table.Players) // Heads-up, SB/Dealer acts first
		}
	} else {
		if len(table.Players) > 1 {
			startIdx = 1 % len(table.Players) // Post-flop, SB acts first
		} else {
			startIdx = 0 // Only one player
		}
	}

	playersToAct = append(playersToAct, table.Players[startIdx:]...)
	playersToAct = append(playersToAct, table.Players[:startIdx]...)

	lastRaiser := playersToAct[0]

	if stage == "Preflop" {
		for _, p := range table.Players { // Find the BB
			if p.Position == game.BigBlind {
				lastRaiser = p // The BB is the "last raiser" preflop
				break
			}
		}
	}

	actedThisRound := make(map[string]bool)
	actionClosed := false

	for !actionClosed {
		if len(table.GetActivePlayers()) <= 1 {
			actionClosed = true
			break
		}

		actionClosed = true

		for _, p := range playersToAct {
			if !p.Active {
				continue
			}

			if p == lastRaiser && actedThisRound[p.Name] && p.CurrentBet == currentBet {
				actionClosed = true
				break // This player closes the action
			}

			// If this player has already acted and their bet is still good, skip them
			if actedThisRound[p.Name] && p.CurrentBet == currentBet {
				continue
			}

			// If we are here, this player *must* act.
			actionClosed = false // The round is not over

			var action string
			var amount int

			if p.Name == "BOT" {

				fmt.Println("\n🤖 BOT’s turn — ")
				stateVec := bot.BuildFeaturesVector(ai, table)                      // Build state vector for BOT
				actionIdx := dqnAgent.ChooseAction(stateVec)                        // DQN chooses action
				actionName, raiseAmount := bot.ActionToAmount(actionIdx, table, ai) // Map action index to name and amount

				fmt.Printf("BOT (DQN) chooses action: %s (raise=%d)\n", actionName, raiseAmount)
				action = actionName
				amount = raiseAmount
			} else {
				fmt.Printf("\n👉 %s's turn (Chips: %d, Bet: %d)\n", p.Name, p.Chips, p.CurrentBet)
				fmt.Print("Enter action [fold/call/check/raise]: ")
				input, _ := reader.ReadString('\n')
				action = strings.TrimSpace(strings.ToLower(input))
			}

			actedThisRound[p.Name] = true

			switch action {
			case "Fold", "fold":
				table.PlayerAction(p, game.Fold, 0, currentBet)
			case "Call", "call":
				table.PlayerAction(p, game.Call, 0, currentBet)
			case "Check", "check":
				if p.CurrentBet < currentBet {
					fmt.Println("Cannot check, must call or fold. Folding.")
					table.PlayerAction(p, game.Fold, 0, currentBet)
				} else {
					table.PlayerAction(p, game.Check, 0, currentBet)
				}
			case "Raise", "raise":
				if p.Name != "BOT" { // If human, ask for amount
					fmt.Print("Enter raise amount: ")
					raiseInput, _ := reader.ReadString('\n')
					raiseAmount, _ := strconv.Atoi(strings.TrimSpace(raiseInput))
					amount = raiseAmount
				}

				// This 'amount' is the TOTAL bet.
				// The raise *must* be at least 2x the *previous bet*.
				// Or 2x the Big Blind if no bet.
				minRaiseAmount := currentBet * 2 // Default: 2x the current bet
				if currentBet == 0 {
					minRaiseAmount = game.BigBlindAmount * 2 // 2x the Big Blind if no bet
				}

				// Ensure the 'amount' (which is the *total* bet) is legal
				if amount < minRaiseAmount && amount < p.Chips {
					amount = minRaiseAmount
					fmt.Printf("Raise too small. Setting to minimum: %d\n", amount)
				}

				table.PlayerAction(p, game.Raise, amount, currentBet)
				currentBet = p.CurrentBet // The new "bet to match"
				lastRaiser = p            // This player is now the one who action closes on

				// Reset acted flags so everyone gets to act again
				actedThisRound = make(map[string]bool)
				actedThisRound[p.Name] = true // The raiser has acted

			default:
				fmt.Println("Invalid input, checking/folding.")
				toCall, _ := getBettingContext(table, p)
				if toCall > 0 {
					table.PlayerAction(p, game.Fold, 0, currentBet)
				} else {
					table.PlayerAction(p, game.Check, 0, currentBet)
				}
			}

			if len(table.GetActivePlayers()) <= 1 {
				actionClosed = true
				break
			}
		}

		if !actionClosed {
			actionClosed = true
			activePlayers := table.GetActivePlayers()
			if len(activePlayers) <= 1 {
				break
			}
			for _, p := range activePlayers {
				if p.CurrentBet < currentBet {
					actionClosed = false // Someone still hasn't called
				}
			}
		}
	}

	table.CollectBets()
}

// ================== COMMUNITY CARD INPUT ===================

func manualCommunityCards(reader *bufio.Reader, table *game.Table, stage string, count int) {
	if len(table.GetActivePlayers()) <= 1 {
		return // Don't deal cards if hand is over
	}
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
		var rankStr string
		var suitChar byte
		// Handle "10"
		if strings.HasPrefix(c, "10") {
			rankStr = "10"
			suitChar = c[len(c)-1] // Get last char for suit
		} else {
			rankStr = string(c[0])
			suitChar = c[1]
		}

		rank := game.Rank(rankStr)
		suit := suitFromChar(suitChar)
		card := game.Card{Rank: rank, Suit: suit}
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
