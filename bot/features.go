package bot

import "github.com/DevanshuTripathi/poker-bot/game"

const (
	// Strength Category (5 slots total)
	// Win-Prob Category (5 slots total)
	// Position Category (3 slots total)
	// Stack Category (3 slots total)
	// Stage (1 slot)
	// Call Cost (1 slot)
	// Pot Size (1 slot)
	// Active Players (1 slot)
	FeatureVectorSize = 20
)

func BuildFeaturesVector(bot *game.Player, table *game.Table) []float64 {
	vec := make([]float64, FeatureVectorSize) // Initialize feature vector with zeros

	strength := EvaluateHand(bot.Hand, table.CommunityCards) // Hand strength evaluation
	stage := len(table.CommunityCards)

	var winProb float64
	if stage == 0 {
		winProb = EstimateWinProbability(bot.Hand, table.CommunityCards, len(table.GetActivePlayers())-1, 50)
	} else {
		winProb = strength
	}

	toCall, pot := getBettingContext(table, bot)      // Get betting context
	numActivePlayers := len(table.GetActivePlayers()) // Number of active players
	posCategory := getPositionCategory(bot, stage)    // Position category
	stackCategory := getStackCategory(bot)            // Stack category
	sCategory := getStrengthCategory(strength)        // Strength category
	wCategory := getWinProbCategory(winProb)          // Win probability category

	if sCategory == "S_JUNK" {
		vec[0] = 1.0
	}
	if sCategory == "S_PAIR" {
		vec[1] = 1.0
	}
	if sCategory == "S_TWO_PAIR" {
		vec[2] = 1.0
	}
	if sCategory == "S_STRONG" {
		vec[3] = 1.0
	}
	if sCategory == "S_MONSTER" {
		vec[4] = 1.0
	}

	if wCategory == "W_LOW" {
		vec[5] = 1.0
	}
	if wCategory == "W_MED" {
		vec[6] = 1.0
	}
	if wCategory == "W_GOOD" {
		vec[7] = 1.0
	}
	if wCategory == "W_HIGH" {
		vec[8] = 1.0
	}
	if wCategory == "W_LOCK" {
		vec[9] = 1.0
	}

	if posCategory == "POS_BLIND" {
		vec[10] = 1.0
	}
	if posCategory == "POS_EARLY" {
		vec[11] = 1.0
	}
	if posCategory == "POS_LATE" {
		vec[12] = 1.0
	}

	if stackCategory == "STACK_SHORT" {
		vec[13] = 1.0
	}
	if stackCategory == "STACK_MED" {
		vec[14] = 1.0
	}
	if stackCategory == "STACK_DEEP" {
		vec[15] = 1.0
	}

	vec[16] = float64(stage) / 5.0                                    // Normalize stage (0-4)
	vec[17] = float64(toCall) / float64(bot.Chips+1)                  // Normalize call cost
	vec[18] = float64(pot) / float64(bot.Chips+toCall+1)              // Normalize pot size
	vec[19] = float64(numActivePlayers) / float64(len(table.Players)) // Normalize active players

	return vec
}
