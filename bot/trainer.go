package bot

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"

	"github.com/DevanshuTripathi/poker-bot/game"
)

type QAgent struct {
	QTable       map[string][]float64
	LearningRate float64
	Gamma        float64
	Epsilon      float64
	NumActions   int
} // Old Q-Learning Agent for reference

type Experience struct {
	state  string
	action int
} // Old experience struct for Q-learning

type DQExperience struct {
	State          []float64
	Action         int
	Reward         float64
	NextState      []float64
	Done           bool
	Hole           []game.Card
	CommunityCards []game.Card
	Table          *game.Table
	Player         *game.Player
} // DQN experience struct

// Old save/load functions for Q-table
func (q *QAgent) Save(filePath string) error {
	fmt.Printf("Saving Q-Table to %s... (%d states)\n", filePath, len(q.QTable))

	// Marshal the map into an indented JSON byte slice
	data, err := json.MarshalIndent(q.QTable, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Q-Table: %w", err)
	}

	// Write the byte slice to the specified file
	err = os.WriteFile(filePath, data, 0644) // 0644 are standard file permissions
	if err != nil {
		return fmt.Errorf("failed to write Q-Table to file: %w", err)
	}

	return nil
}

// Old save/load functions for Q-table
func (q *QAgent) Load(filePath string) error {
	fmt.Printf("Loading Q-Table from %s...\n", filePath)

	// Read the entire file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Q-Table file: %w", err)
	}

	// Unmarshal the JSON data into the QTable map
	// q.QTable must be non-nil, which NewQAgent() handles
	err = json.Unmarshal(data, &q.QTable)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Q-Table: %w", err)
	}

	fmt.Printf("Successfully loaded %d states.\n", len(q.QTable))
	return nil
}

// Old copy table functions for Q-table
func deepCopyQTable(src map[string][]float64) map[string][]float64 {
	dest := make(map[string][]float64)
	for state, values := range src {
		newValues := make([]float64, len(values))
		copy(newValues, values)
		dest[state] = newValues
	}
	return dest
}

// --- Poker Specific Feature Helpers ---

func getStrengthCategory(strength float64) string {
	if strength < 0.20 {
		return "S_JUNK"
	}
	if strength < 0.30 {
		return "S_PAIR"
	}
	if strength < 0.40 {
		return "S_TWO_PAIR"
	}
	if strength < 0.70 {
		return "S_STRONG"
	} // Trips, Straight, Flush
	return "S_MONSTER" // Full House, Quads, Straight Flush
}

func getWinProbCategory(winProb float64) string {
	if winProb < 0.20 {
		return "W_LOW"
	}
	if winProb < 0.40 {
		return "W_MED"
	}
	if winProb < 0.60 {
		return "W_GOOD"
	}
	if winProb < 0.85 {
		return "W_HIGH"
	}
	return "W_LOCK" // > 85%
}

func getPositionCategory(p *game.Player, stage int) string {
	// Pre-flop, position is king
	if stage == 0 {
		switch p.Position {
		case game.SmallBlind, game.BigBlind:
			return "POS_BLIND"
		case game.Dealer:
			return "POS_LATE"
		default:
			// Simplification for 5-player game
			return "POS_EARLY"
		}
	}

	// Post-flop, we'll re-use the pre-flop positions as a
	// good-enough estimate of who acts first or last.
	switch p.Position {
	case game.SmallBlind, game.BigBlind:
		return "POS_BLIND"
	case game.Dealer:
		return "POS_LATE"
	default:
		return "POS_EARLY"
	}
}

func getStackCategory(p *game.Player) string {
	// We measure stacks in "Big Blinds" (BBs)
	stackInBBs := float64(p.Chips) / float64(game.BigBlindAmount)

	// --- NEW BUCKETS designed for a 500 BB game ---

	if stackInBBs < 200 {
		// Player has lost more than half their stack (500 -> <200)
		return "STACK_SHORT"
	}
	if stackInBBs < 700 {
		// Player is in the "normal" range, around the 500 BB start
		// (i.e., between 200 BB and 700 BB)
		return "STACK_MED"
	}
	// Player has won a significant amount (> 700 BBs)
	return "STACK_DEEP"
}

func getBettingContext(table *game.Table, bot *game.Player) (int, int) {
	maxBet := table.GetHighestBet()
	toCall := maxBet - bot.CurrentBet

	if toCall < 0 {
		toCall = 0
	}

	return toCall, table.Pot
}

// Monkey opponent: very aggressive, bluffs often
func getMonkeyOpponentAction(p *game.Player, table *game.Table) (string, int) {
	maxBet := table.GetHighestBet()
	toCall := maxBet - p.CurrentBet
	minRaise := game.BigBlindAmount * 2
	strength := EvaluateHand(p.Hand, table.CommunityCards)

	stage := len(table.CommunityCards) // 0 preflop, 3 flop, 4 turn, 5 river

	if strength > 0.7 {
		// Always all-in with monsters
		return "Raise", p.Chips
	}

	if strength > 0.4 {
		// Strong hands: 90% chance to raise
		if rand.Float64() < 0.90 {
			raiseAmount := table.Pot / 2
			if toCall > 0 {
				raiseAmount = toCall * 3
			}
			return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
		}
	}

	// Post-flop pressure: monkey raises more aggressively on turn/river
	if stage > 2 {
		if rand.Float64() < 0.45 { // 45% chance to apply turn/river pressure
			raiseAmount := int(float64(table.Pot) * 0.7) // big-ish raise ~70% pot
			if raiseAmount < minRaise {
				raiseAmount = minRaise
			}
			return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
		}
	}

	// 80% chance to be aggressive
	if rand.Float64() < 0.80 {
		if strength >= 0.2 || rand.Float64() < 0.50 { // Any pair or bluff
			raiseAmount := table.Pot / 2
			if toCall > 0 {
				raiseAmount = toCall * 3
			}
			if raiseAmount < minRaise {
				raiseAmount = minRaise
			}
			return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
		}
	}
	// Fallback to "calling station"
	if toCall == 0 {
		return "Check", 0
	}
	return "Call", toCall
}

// Normal opponent: calling station with 40% aggression
func getNormalOpponentAction(p *game.Player, table *game.Table) (string, int) {
	maxBet := table.GetHighestBet()
	toCall := maxBet - p.CurrentBet
	minRaise := game.BigBlindAmount * 2
	strength := EvaluateHand(p.Hand, table.CommunityCards)
	stage := len(table.CommunityCards) // 0 preflop, 3 flop, 4 turn, 5 river

	if strength > 0.7 {
		// 70% chance to go all-in with monsters
		if rand.Float64() < 0.70 {
			return "Raise", p.Chips
		}
	}

	if strength > 0.4 {
		// Strong hands: 50% chance to raise
		if rand.Float64() < 0.50 {
			raiseAmount := table.Pot / 2
			if toCall > 0 {
				raiseAmount = toCall * 2
			}
			return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
		}
	}

	// Post-flop occasional pressure
	if stage > 2 && rand.Float64() < 0.28 { // ~28% turn/river raise
		raiseAmount := int(float64(table.Pot) * 0.5) // ~50% pot
		if raiseAmount < minRaise {
			raiseAmount = minRaise
		}
		return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
	}

	// 40% chance to be aggressive
	if rand.Float64() < 0.20 {
		if strength >= 0.2 || rand.Float64() < 0.1 {
			raiseAmount := table.Pot / 3
			if toCall > 0 {
				raiseAmount = toCall * 2
			}
			if raiseAmount < minRaise {
				raiseAmount = minRaise
			}
			return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
		}
	}

	if toCall > 0 && strength < 0.19 { // < 0.19 is "S_JUNK"
		return "Fold", 0
	}

	// Default: "Calling Station"
	if toCall == 0 {
		return "Check", 0
	}
	if toCall > p.Chips/8 && strength < 0.25 {
		return "Fold", 0
	} // Folds if it's a huge bet against a weak hand
	return "Call", toCall
}

// Find the winner of the hand
func findWinner(table *game.Table) *game.Player {
	active := table.GetActivePlayers()
	if len(active) == 1 {
		return active[0]
	}
	bestRank := -1.0
	var winner *game.Player = nil

	for _, p := range active {
		rank := EvaluateHand(p.Hand, table.CommunityCards)
		if rank > bestRank {
			bestRank = rank
			winner = p
		}
	}
	return winner
}

// Old Q-Learning Agent for reference
func NewQAgent(numActions int, lr, gamma, epsilon float64) *QAgent {
	return &QAgent{
		QTable:       make(map[string][]float64),
		LearningRate: lr,
		Gamma:        gamma,
		Epsilon:      epsilon,
		NumActions:   numActions,
	}
}

// Old function for getting state values
func (q *QAgent) GetQValues(state string) []float64 {
	if _, ok := q.QTable[state]; !ok {
		q.QTable[state] = make([]float64, q.NumActions)
	}
	return q.QTable[state]
}

// Old function for choosing action
func (q *QAgent) ChooseAction(state string) int {
	if rand.Float64() < q.Epsilon {
		return rand.Intn(q.NumActions)
	}
	qValues := q.GetQValues(state)

	// find max value
	maxVal := qValues[0]
	for i := 1; i < q.NumActions; i++ {
		if qValues[i] > maxVal {
			maxVal = qValues[i]
		}
	}

	// collect all indices that equal maxVal (tie-breaking)
	var maxIdxs []int
	eps := 1e-9
	for i, v := range qValues {
		if math.Abs(v-maxVal) <= eps {
			maxIdxs = append(maxIdxs, i)
		}
	}

	if len(maxIdxs) == 1 {
		return maxIdxs[0]
	}

	// pick randomly among the best actions to avoid deterministic ties
	return maxIdxs[rand.Intn(len(maxIdxs))]
}

// Old function for updating Q-values
func (q *QAgent) Update(state string, action int, reward float64, nextState string) {
	// ensure Q-values exist for current and next state
	qValues := q.GetQValues(state)
	nextQ := q.GetQValues(nextState)

	// validate action index
	if action < 0 || action >= len(qValues) {
		// invalid action index; skip update
		return
	}

	// safely compute max of nextQ (handle possible zero-length)
	maxNextQ := 0.0
	if len(nextQ) > 0 {
		maxNextQ = nextQ[0]
		for _, val := range nextQ {
			if val > maxNextQ {
				maxNextQ = val
			}
		}
	}

	target := reward + q.Gamma*maxNextQ
	qValues[action] += q.LearningRate * (target - qValues[action])
}

// --- Poker Specific Feature Builders ---
// This Old function converts the table + bot state into a simplified feature string (for lookup)
func BuildFeatures(bot *game.Player, table *game.Table) string {
	strength := EvaluateHand(bot.Hand, table.CommunityCards)
	stage := len(table.CommunityCards) // 0 preflop, 3 flop, 4 turn, 5 river

	var winProb float64
	if stage == 0 {
		// --- THIS IS THE FIX ---
		// Only run the slow simulation pre-flop, and just a few times
		winProb = EstimateWinProbability(bot.Hand, table.CommunityCards, len(table.GetActivePlayers())-1, 50) // 50 sims, not 300
	} else {
		// Post-flop, just use the fast hand-strength
		// We can just set winProb to 0 or use strength
		winProb = strength // Use strength as a stand-in
	}

	toCall, pot := getBettingContext(table, bot)
	numActivePlayers := len(table.GetActivePlayers())
	posCategory := getPositionCategory(bot, stage)

	sCategory := getStrengthCategory(strength)
	wCategory := getWinProbCategory(winProb)
	stackCategory := getStackCategory(bot)

	var callCategory int
	if toCall == 0 {
		callCategory = 0 // "check"
	} else if toCall <= game.BigBlindAmount*2 {
		callCategory = 1 // "small call"
	} else if toCall <= table.Pot/2 {
		callCategory = 2 // "medium call"
	} else {
		callCategory = 3 // "large call"
	}

	var potCategory int
	if pot < game.BigBlindAmount*5 {
		potCategory = 0 // small pot
	} else if pot < game.BigBlindAmount*20 {
		potCategory = 1 // medium pot
	} else {
		potCategory = 2 // large pot
	}

	return fmt.Sprintf("%s_%s_%s_%s_T%d_C%d_P%d_A%d",
		sCategory,
		wCategory,
		posCategory,
		stackCategory,
		stage,
		callCategory,
		potCategory,
		numActivePlayers,
	)
}

func ActionToAmount(action int, table *game.Table, bot *game.Player) (string, int) {

	toCall, pot := getBettingContext(table, bot)
	minRaise := game.BigBlindAmount * 2
	raiseAmount := 0

	switch action {
	case 0: // fold -> if nothing to call, treat as check
		if toCall <= 0 {
			return "Check", 0
		}
		return "Fold", 0
	case 1: // call or check
		if toCall <= 0 {
			return "Check", 0
		}
		return "Call", toCall
	case 2: // small raise (~0.5 - 1.0 pot)
		// prefer half-pot to 3x toCall
		if toCall > 0 {
			raiseAmount = int(math.Max(float64(minRaise), float64(toCall)*2))
		} else {
			raiseAmount = int(math.Max(float64(minRaise), float64(pot)/2))
		}
		return "Raise", int(math.Min(float64(bot.Chips), float64(raiseAmount)))

	case 3: // big raise (~pot to 1.5 pot)
		if toCall > 0 {
			raiseAmount = int(math.Max(float64(minRaise*2), float64(toCall)*4))
		} else {
			raiseAmount = int(math.Max(float64(minRaise*2), float64(pot)))
		}
		return "Raise", int(math.Min(float64(bot.Chips), float64(raiseAmount)))
	default:
		if toCall <= 0 {
			return "Check", 0
		}
		return "Call", toCall
	}
}

// Old immediate Reward shaping
func SimulateActionAndReturnReward(bot *game.Player, actionName string, amount int, table *game.Table) (float64, *game.Table) {
	reward := 0.0

	switch actionName {
	case "fold":
		reward = -0.1
	case "call":
		reward = 0.1
	case "raise":
		reward = 0.2
	case "check":
		reward = 0.05
	}

	// You can add outcome-based rewards if simulating till end of hand
	return reward, table
}

// Old function for updating Q-values at terminal state
func (q *QAgent) UpdateTerminal(state string, action int, reward float64) {
	qValues := q.GetQValues(state)

	if action < 0 || action >= len(qValues) {
		// invalid action index; skip update
		return
	}

	target := reward
	qValues[action] += q.LearningRate * (target - qValues[action])
}

// The main training loop
func TrainPokerBot(episodes int, numPlayers int) *DQNAgent {

	dqnAgent := NewDQNAgent(
		0.0001,  // learningRate
		0.95,    // gamma
		1.0,     // epsilon
		0.01,    // minEpsilon
		0.99990, // Epsilon *Decay Factor*
	)

	opponentAgent := NewDQNAgent(
		0.0001,  // lr
		0.95,    // gamma
		0.00,    // Epsilon
		0.00,    // minEpsilon
		0.99990, // decayRate
	)

	players := []*game.Player{game.NewPlayer("BOT_Learner", 1000)} // Our bot
	for i := 1; i < numPlayers-1; i++ {
		opponent := game.NewPlayer("Opponent", 1000)
		opponent.PlayerType = "normal" // default type
		players = append(players, opponent)
	}
	botPlayer := players[0]
	cloneOpponent := game.NewPlayer("Clone_Opponent", 1000)
	cloneOpponent.PlayerType = "clone"
	players = append(players, cloneOpponent)
	oppPlayer := players[len(players)-1]
	table := game.NewTable(players)

	fmt.Printf("Training DQN with Progressive Difficulty (%d players)...\n", numPlayers)

	// === CSV LOGGING SETUP ===
	csvFile, err := os.Create("training_log.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	// CSV header
	csvWriter.Write([]string{
		"episode",
		"epsilon",
		"wins",
		"bot_chips",
		"clone_chips",
		"bot_buyins",
		"clone_buyins",
		"flops",
		"turns",
		"rivers",
	})

	opponentTypes := []string{"normal"}

	var lastState []float64
	var lastAction int

	var cloneLastState []float64
	var cloneLastAction int

	dummyNextState := make([]float64, FeatureVectorSize)

	winCount := 0
	cloneWinCount := 0
	flopCount := 0
	turnCount := 0
	riverCount := 0

	for e := 0; e < episodes; e++ {

		// Force reset stacks every hand to prevent massive accumulations
		for _, p := range table.Players {
			if p.Chips <= 0 {
				p.Buyin++
			}
			p.Chips = 1000
			p.Active = true
		}

		table.ResetForTraining() // New hand
		table.DealHands()        // Deal cards
		table.PostBlinds()       // Post blinds

		lastState = nil
		lastAction = 0

		cloneLastState = nil
		cloneLastAction = 0

		var handHistory []*DQExperience
		var cloneHistory []*DQExperience

		// if e == 200000 {
		// 	fmt.Println("\n--- ADDING 'MONKEY' OPPONENT ---")
		// 	opponentTypes = append(opponentTypes, "monkey") // Add monkey opponent
		// }

		// Assign opponent types
		for _, p := range players {
			if p.Name != "BOT_Learner" && p.Name != "Clone_Opponent" {
				p.PlayerType = opponentTypes[rand.Intn(len(opponentTypes))]
			}
		}

		done := false
		handSawFlop := false
		handSawTurn := false
		handSawRiver := false
		botStartChips := botPlayer.Chips

		// Simulate the hand
		for stage := 0; stage < 4 && !done; stage++ {

			// Reset players' last actions
			for _, p := range table.Players {
				p.LastAction = ""
			}

			if botPlayer.Active {
				if stage == 1 {
					handSawFlop = true
				} else if stage == 2 {
					handSawTurn = true
				} else if stage == 3 {
					handSawRiver = true
				}
			}

			if len(table.GetActivePlayers()) <= 1 {
				break
			} // Early exit if only one player left

			startIdx := 1
			if stage == 0 {
				if numPlayers == 2 {
					startIdx = 0
				} else {
					startIdx = 3 % len(table.Players)
				}
			}

			// lastAggressor := -1
			// if stage == 0 {
			// 	for i, p := range table.Players {
			// 		if p.Position == game.BigBlind {
			// 			lastAggressor = i
			// 			break
			// 		}
			// 	}
			// } else {
			// 	for i := 1; i <= len(table.Players); i++ {
			// 		p := table.Players[i%len(table.Players)]
			// 		if p.Active {
			// 			lastAggressor = i % len(table.Players)
			// 			break
			// 		}
			// 	}
			// }
			// if lastAggressor == -1 {
			// 	lastAggressor = 0
			// } // Failsafe

			currentActorIdx := startIdx

			for {

				if table.IsBettingRoundOver(stage == 0) {
					break
				}

				p := table.Players[currentActorIdx]

				// allMatched := true
				// activePlayers := table.GetActivePlayers()

				// if currentActorIdx == (lastAggressor+1)%len(players) && p.CurrentBet == maxBet {
				// 	break
				// }

				// if len(activePlayers) <= 1 {
				// 	done = true
				// 	break
				// }

				// for _, ap := range activePlayers {
				// 	if ap.Chips > 0 && ap.CurrentBet < maxBet {
				// 		allMatched = false
				// 	}
				// }
				// // Special Preflop BB case
				// if stage == 0 && p.Position == game.BigBlind && maxBet == game.BigBlindAmount {
				// 	allMatched = false
				// }
				// if allMatched {
				// 	break
				// }

				if !p.Active || p.Chips == 0 { // Skip folded or all-in players
					currentActorIdx = (currentActorIdx + 1) % len(table.Players)
					continue
				}

				maxBet := table.GetHighestBet()

				var actionName string
				var amount int

				if p.Name == "BOT_Learner" {
					stateVec := BuildFeaturesVector(p, table) // Build feature vector
					action := dqnAgent.ChooseAction(stateVec) // Choose action

					if lastState != nil {
						handHistory = append(handHistory, &DQExperience{
							State: lastState, Action: lastAction, Reward: 0, NextState: stateVec, Done: false,
							Hole: append([]game.Card{}, p.Hand...), CommunityCards: append([]game.Card{}, table.CommunityCards...),
							Table: table, Player: botPlayer,
						})
					}

					lastState = stateVec // Update last state
					lastAction = action  // Update last action

					actionName, amount = ActionToAmount(action, table, p) // Map action to game action

				} else {
					// Opponent acts
					switch p.PlayerType {
					case "clone":
						stateVec := BuildFeaturesVector(p, table)
						action := opponentAgent.ChooseAction(stateVec)

						if cloneLastState != nil {
							cloneHistory = append(cloneHistory, &DQExperience{
								State: cloneLastState, Action: cloneLastAction, Reward: 0, NextState: stateVec, Done: false,
								Hole: append([]game.Card{}, p.Hand...), CommunityCards: append([]game.Card{}, table.CommunityCards...),
								Table: table, Player: oppPlayer,
							})
						}

						cloneLastState = stateVec
						cloneLastAction = action

						actionName, amount = ActionToAmount(action, table, p)
					case "monkey":
						actionName, amount = getMonkeyOpponentAction(p, table)
					default: // "normal"
						actionName, amount = getNormalOpponentAction(p, table)
					}
				}

				// Execute action
				table.PlayerAction(p, game.Action(actionName), amount, maxBet)

				currentActorIdx = (currentActorIdx + 1) % len(table.Players)
				if len(table.GetActivePlayers()) <= 1 {
					done = true
					break
				}
			}

			table.CollectBets() // Collect bets at end of round

			// Deal community cards
			if stage == 0 {
				table.DealFlop()
			} else if stage == 1 {
				table.DealTurn()
			} else if stage == 2 {
				table.DealRiver()
			}
		}

		table.CollectBets() // Final collection of bets

		finalReward := 0.0
		// cloneReward := 0.0
		// Determine the winner
		winner := findWinner(table)
		if winner != nil && winner.Name == "BOT_Learner" {
			botPlayer.Chips += table.Pot
			winCount++
		}

		if winner != nil && winner.Name == "Clone_Opponent" {
			oppPlayer.Chips += table.Pot
			cloneWinCount++
		}

		chipChange := float64(botPlayer.Chips - botStartChips)
		finalReward = chipChange / 100.0 // Scale reward

		if finalReward > 0 {
			finalReward *= 2 // Bonus for winning chips
		}

		if finalReward > 2.0 {
			finalReward = 2.0
		}
		if finalReward < -2.0 {
			finalReward = -2.0
		}

		// if winner != nil && winner.Name == "Clone_Opponent" {
		// 	oppPlayer.Chips += table.Pot
		// 	cloneReward = 1.0 // +1.0 for a win
		// } else {
		// 	cloneReward = -1.0 // -1.0 for a loss or fold
		// }

		if lastState != nil { // Final learning step at end of hand
			handHistory = append(handHistory, &DQExperience{
				State: lastState, Action: lastAction, Reward: finalReward, NextState: dummyNextState, Done: true,
				Hole: append([]game.Card{}, botPlayer.Hand...), CommunityCards: append([]game.Card{}, table.CommunityCards...),
				Table: table, Player: botPlayer,
			})
		}

		if cloneLastState != nil { // Final learning step for clone opponent
			cloneHistory = append(cloneHistory, &DQExperience{
				State: cloneLastState, Action: cloneLastAction, Reward: 0, NextState: dummyNextState, Done: true,
				Hole: append([]game.Card{}, oppPlayer.Hand...), CommunityCards: append([]game.Card{}, table.CommunityCards...),
				Table: table, Player: oppPlayer,
			})
		}

		// Now, "back-propagate" the final reward
		// We learn from the *entire hand history* in reverse

		// strength := EvaluateHand(botPlayer.Hand, table.CommunityCards)
		// if finalReward == -1 && strength < 0.3 {
		// 	finalReward = -1.5
		// }

		// G := finalReward

		// handHistory = append(handHistory, cloneHistory...)

		for i := len(handHistory) - 1; i >= 0; i-- {
			mem := handHistory[i]

			if mem.Done {
				mem.Reward = finalReward
			} else {
				mem.Reward = 0
			}
			// else {
			// 	strength := EvaluateHand(mem.Hole, mem.CommunityCards)
			// 	toCall, pot := getBettingContext(mem.Table, mem.Player)
			// 	mem.Reward = 0

			// 	if mem.Action == 1 { // CALL
			// 		if strength < 0.30 {
			// 			pen := 0.10 + math.Min(0.5, float64(toCall)/math.Max(1.0, float64(pot))) // scale by cost
			// 			mem.Reward -= pen
			// 		} else {
			// 			// small positive for reasonable calls with decent equity
			// 			mem.Reward += 0.02
			// 		}
			// 	}

			// 	if (mem.Action == 2 || mem.Action == 3) && strength < 0.35 {
			// 		mem.Reward -= 0.20
			// 	}

			// 	// Reward folding weak hands slightly (avoid stubborn over-calling)
			// 	if mem.Action == 0 && strength < 0.25 {
			// 		mem.Reward += 0.5
			// 	}
			// 	// Big penalty for folding monsters
			// 	if mem.Action == 0 && strength > 0.75 {
			// 		mem.Reward -= 0.6
			// 	}

			// }

			dqnAgent.Learn(mem.State, mem.Action, mem.Reward, mem.NextState, mem.Done)

			// G *= dqnAgent.Gamma

		}

		// cloneStrength := EvaluateHand(oppPlayer.Hand, table.CommunityCards)
		// if cloneReward == -1 && cloneStrength < 0.3 {
		// 	cloneReward = -1.5
		// }

		// for i := len(cloneHistory) - 1; i >= 0; i-- {
		// 	mem := cloneHistory[i]

		// 	if mem.Done {
		// 		mem.Reward = cloneReward
		// 	} else {
		// 		strength := EvaluateHand(mem.Hole, mem.CommunityCards)
		// 		toCall, pot := getBettingContext(mem.Table, mem.Player)
		// 		mem.Reward = 0

		// 		if mem.Action == 1 { // CALL
		// 			if strength < 0.30 {
		// 				pen := 0.10 + math.Min(0.5, float64(toCall)/math.Max(1.0, float64(pot))) // scale by cost
		// 				mem.Reward -= pen
		// 			} else {
		// 				// small positive for reasonable calls with decent equity
		// 				mem.Reward += 0.02
		// 			}
		// 		}

		// 		if (mem.Action == 2 || mem.Action == 3) && strength < 0.35 {
		// 			mem.Reward -= 0.20
		// 		}

		// 		// Reward folding weak hands slightly (avoid stubborn over-calling)
		// 		if mem.Action == 0 && strength < 0.25 {
		// 			mem.Reward += 0.5
		// 		}
		// 		// Big penalty for folding monsters
		// 		if mem.Action == 0 && strength > 0.75 {
		// 			mem.Reward -= 0.6
		// 		}

		// 	}

		// 	opponentAgent.Learn(mem.State, mem.Action, mem.Reward, mem.NextState, mem.Done)

		// 	// G *= dqnAgent.Gamma

		// }

		dqnAgent.DecayEpsilon() // Decay epsilon after each episode
		// opponentAgent.DecayEpsilon() // Decay epsilon for opponent as well

		if handSawFlop {
			flopCount++
		}
		if handSawTurn {
			turnCount++
		}
		if handSawRiver {
			riverCount++
		}

		if (e+1)%1000 == 0 {
			// This "snapshot" is just a printout. The *real* snapshot
			// (target network update) is happening inside dqn.agent.Learn()
			fmt.Printf("\n--- EPISODE %d --- \n", e+1)
			fmt.Printf("Current Epsilon: %.4f\n", dqnAgent.GetEpsilon())
			fmt.Printf("Wins so far: %d\n", winCount)
			fmt.Printf("Clone Wins so far: %d\n", cloneWinCount)
			fmt.Printf("Bot's Chips: %d, Buyins: %d\n", botPlayer.Chips, botPlayer.Buyin)
			fmt.Printf("Clone Chips: %d, Buyins: %d\n", oppPlayer.Chips, oppPlayer.Buyin)
			fmt.Printf("Flops seen: %d, Turns seen: %d, Rivers seen: %d\n", flopCount, turnCount, riverCount)

			// === WRITE TO CSV ===
			csvWriter.Write([]string{
				strconv.Itoa(e + 1),
				fmt.Sprintf("%.4f", dqnAgent.GetEpsilon()),
				strconv.Itoa(winCount),
				strconv.Itoa(cloneWinCount),
				strconv.Itoa(botPlayer.Chips),
				strconv.Itoa(oppPlayer.Chips),
				strconv.Itoa(botPlayer.Buyin),
				strconv.Itoa(oppPlayer.Buyin),
				strconv.Itoa(flopCount),
				strconv.Itoa(turnCount),
				strconv.Itoa(riverCount),
			})
			csvWriter.Flush()

			winCount = 0
			cloneWinCount = 0
			flopCount = 0
			turnCount = 0
			riverCount = 0

			fmt.Println("Reseting Chips")

			for _, p := range table.Players {
				p.Chips = 1000
				p.Active = true
				p.CurrentBet = 0
				p.Buyin = 0
			}
		}

		if (e+1)%5000 == 0 {
			fmt.Printf("\n--- BRAIN SYNC CHECK (Episode %d) ---\n", e+1)

			// 1. Get the weights
			botWeights := dqnAgent.GetWeights()

			// 2. PRINT ALL KEYS (This will tell us the real names)
			fmt.Println("Available Weight Keys:")
			var validKey string
			for k, v := range botWeights {
				fmt.Printf(" - Key: %s | Type: %T\n", k, v)
				validKey = k // Store a valid key to use for the test
			}

			// 3. Sync to Opponent
			opponentAgent.SetWeights(botWeights)
			fmt.Println("Weights synced via library.")

			// 4. Perform the DNA Test (Only if we found a key)
			if validKey != "" {
				oppWeights := opponentAgent.GetWeights()

				// We use fmt.Sprintf to safely print the value regardless of its type
				val1 := fmt.Sprintf("%v", botWeights[validKey])
				val2 := fmt.Sprintf("%v", oppWeights[validKey])

				// Truncate for readability (just first 20 chars)
				if len(val1) > 20 {
					val1 = val1[:20] + "..."
				}
				if len(val2) > 20 {
					val2 = val2[:20] + "..."
				}

				fmt.Printf("Bot [%s]:      %s\n", validKey, val1)
				fmt.Printf("Opponent [%s]: %s\n", validKey, val2)

				if val1 == val2 {
					fmt.Println("✅ SUCCESS: Brains match!")
				} else {
					fmt.Println("❌ FAILURE: Brains do not match!")
				}
			} else {
				fmt.Println("⚠️ WARNING: GetWeights() returned empty map!")
			}
		}

	}
	fmt.Println("Training completed.")
	return dqnAgent
}
