package bot

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"

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
	State     []float64
	Action    int
	Reward    float64
	NextState []float64
	Done      bool
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

	// 80% chance to be aggressive
	if rand.Float64() < 0.80 {
		if strength >= 0.2 || rand.Float64() < 0.1 { // Any pair or bluff
			raiseAmount := table.Pot
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
	if toCall > p.Chips/10 {
		return "Fold", 0
	} // Folds if it's a huge bet
	return "Call", toCall
}

// Normal opponent: calling station with occasional aggression
func getNormalOpponentAction(p *game.Player, table *game.Table) (string, int) {
	maxBet := table.GetHighestBet()
	toCall := maxBet - p.CurrentBet
	minRaise := game.BigBlindAmount * 2
	strength := EvaluateHand(p.Hand, table.CommunityCards)

	// 20% chance to be aggressive
	if rand.Float64() < 0.20 {
		if strength >= 0.2 || rand.Float64() < 0.1 {
			raiseAmount := table.Pot
			if toCall > 0 {
				raiseAmount = toCall * 3
			}
			if raiseAmount < minRaise {
				raiseAmount = minRaise
			}
			return "Raise", int(math.Min(float64(p.Chips), float64(raiseAmount)))
		}
	}
	// Default: "Calling Station"
	if toCall == 0 {
		return "Check", 0
	}
	if toCall > p.Chips/5 {
		return "Fold", 0
	} // Folds if > 20% of stack
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

// Old Action mapping (0=fold, 1=call/check, 2=small raise, 3=big raise)
func ActionToAmount(action int, table *game.Table, bot *game.Player) (string, int) {

	toCall, pot := getBettingContext(table, bot)
	minRaise := game.BigBlindAmount * 2

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
	case 2:
		raiseAmount := 0
		if toCall > 0 {
			raiseAmount = toCall * 3
		} else {
			raiseAmount = pot / 2
		}
		if raiseAmount < minRaise {
			raiseAmount = minRaise
		}
		return "Raise", int(math.Min(float64(bot.Chips), float64(raiseAmount)))
	case 3:
		raiseAmount := pot + toCall
		if raiseAmount == 0 {
			raiseAmount = game.BigBlindAmount * 4
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
		0.0001,   // learningRate
		0.95,     // gamma
		1.0,      // epsilon
		0.01,     // minEpsilon
		0.999990, // Epsilon *Decay Factor*
	)

	players := []*game.Player{game.NewPlayer("BOT_Learner", 1000)} // Our bot
	for i := 1; i < numPlayers; i++ {
		opponent := game.NewPlayer("Opponent", 1000)
		opponent.PlayerType = "normal" // default type
		players = append(players, opponent)
	}
	botPlayer := players[0]
	table := game.NewTable(players)

	fmt.Printf("Training DQN with Progressive Difficulty (%d players)...\n", numPlayers)
	opponentTypes := []string{"normal"} // Start with only normal opponents

	var lastState []float64
	var lastAction int
	lastChips := botPlayer.Chips

	dummyNextState := make([]float64, FeatureVectorSize)

	for e := 0; e < episodes; e++ {

		table.ResetForNewHand() // New hand
		table.DealHands()       // Deal cards
		table.PostBlinds()      // Post blinds

		lastState = nil
		lastAction = 0
		lastChips = botPlayer.Chips

		if e == 200000 {
			fmt.Println("\n--- ADDING 'MONKEY' OPPONENT ---")
			opponentTypes = append(opponentTypes, "monkey") // Add monkey opponent
		}

		// Assign opponent types
		for _, p := range players {
			if p.Name != "BOT_Learner" {
				p.PlayerType = opponentTypes[rand.Intn(len(opponentTypes))]
			}
		}

		done := false

		// Simulate the hand
		for stage := 0; stage < 4 && !done; stage++ {
			if len(table.GetActivePlayers()) <= 1 {
				break
			} // Early exit if only one player left

			startIdx := 1
			if stage == 0 {
				startIdx = 3 % len(table.Players) // First to act pre-flop is UTG
			}

			playersToAct := len(table.GetActivePlayers())
			playersActed := 0

			for playersActed < playersToAct || !table.IsBettingRoundOver() {
				if len(table.GetActivePlayers()) <= 1 {
					done = true
					break
				} // Early exit if only one player left

				playerIdx := (startIdx + playersActed) % len(table.Players)
				p := table.Players[playerIdx]

				if !p.Active {
					playersActed++
					continue
				}

				maxBet := table.GetHighestBet()

				var actionName string
				var amount int

				if p.Name == "BOT_Learner" {
					stateVec := BuildFeaturesVector(p, table) // Build feature vector
					action := dqnAgent.ChooseAction(stateVec) // Choose action

					if lastState != nil {
						// We are NOT done, so reward is 0 (or chip-change)
						// We give a small reward based on chip changes
						reward := (botPlayer.Chips - lastChips)
						dqnAgent.Learn(lastState, lastAction, reward, stateVec, false)
					}

					lastState = stateVec        // Update last state
					lastAction = action         // Update last action
					lastChips = botPlayer.Chips // Update last chips

					actionName, amount = ActionToAmount(action, table, p) // Map action to game action

				} else {
					// Opponent acts
					switch p.PlayerType {
					case "monkey":
						actionName, amount = getMonkeyOpponentAction(p, table)
					default: // "normal"
						actionName, amount = getNormalOpponentAction(p, table)
					}
				}

				// Execute action
				table.PlayerAction(p, game.Action(actionName), amount, maxBet)
				if actionName == "Raise" { // Reset players to act after a raise
					playersToAct = len(table.GetActivePlayers())
					playersActed = 0
					startIdx = (playerIdx + 1) % len(table.Players)
				} else { // Move to next player
					playersActed++
				}
				if playersActed >= len(table.Players)*3 {
					break
				} // Safety break to avoid infinite loops
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

		// Determine the winner
		winner := findWinner(table)
		if winner != nil && winner.Name == "BOT" { // Our bot won
			botPlayer.Chips += table.Pot // Award pot to bot
		}

		if lastState != nil { // Final learning step at end of hand
			finalReward := botPlayer.Chips - lastChips // Calculate final reward
			dqnAgent.Learn(lastState, lastAction, finalReward, dummyNextState, true)
		}

		dqnAgent.DecayEpsilon() // Decay epsilon after each episode

		if (e+1)%1000 == 0 {
			// This "snapshot" is just a printout. The *real* snapshot
			// (target network update) is happening inside dqn.agent.Learn()
			fmt.Printf("\n--- EPISODE %d --- \n", e+1)
			fmt.Printf("Current Epsilon: %.4f\n", dqnAgent.GetEpsilon())
		}
	}
	fmt.Println("Training completed.")
	return dqnAgent
}
