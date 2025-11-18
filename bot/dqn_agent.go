package bot

import (
	"math/rand"
	"os"

	"github.com/iampaapa/dqn"
)

type DQNAgent struct {
	agent      *dqn.DQN // Underlying DQN agent
	Gamma      float64  // Discount factor
	Epsilon    float64  // Exploration rate
	MinEpsilon float64  // Minimum exploration rate
	EpsDecay   float64  // Exploration decay rate
	NumActions int      // Number of possible actions(4: check/fold, call/fold, small raise, big raise)
}

func NewDQNAgent(lr, gamma, epsilon, minEpsilon, epsilonDecay float64) *DQNAgent {
	inputSize := FeatureVectorSize
	numActions := 4
	hiddenSize := 64    // Network hidden layer size
	bufferSize := 50000 // Replay buffer size

	agent := dqn.NewDQN(
		inputSize,
		hiddenSize,
		numActions,
		bufferSize,
		gamma,
		epsilon,
		lr,
		dqn.ReLU,
	)

	return &DQNAgent{
		agent:      agent,
		Gamma:      gamma,
		Epsilon:    epsilon,
		MinEpsilon: minEpsilon,
		EpsDecay:   epsilonDecay,
		NumActions: numActions,
	}

}

func (dqn *DQNAgent) ChooseAction(state []float64) int {
	// Handles Exploration and Exploitation internally
	action := dqn.agent.EpsilonGreedyPolicy(state, dqn.NumActions)
	return action
}

// Argmax returns the index of the maximum value in a slice of float64
func Argmax(arr []float64) int {
	maxIdx := 0
	maxVal := arr[0]
	for i, val := range arr {
		if val > maxVal {
			maxIdx = i
			maxVal = val
		}
	}
	return maxIdx
}

func (dqn *DQNAgent) ChooseActionSafe(state []float64, toCall, pot, stack int) int {
	strength := state[0] // your feature vector's 1st element is hand strength

	// Epsilon random move
	if rand.Float64() < dqn.Epsilon {
		return rand.Intn(dqn.NumActions)
	}

	// Normal DQN action
	qValues := dqn.agent.QNetworkPredict(state)
	best := Argmax(qValues)

	// 1. If call cost is huge relative to pot, fold weak hands
	if toCall > 0 {
		odds := float64(toCall) / float64(toCall+pot+1)

		// If equity << required odds, fold
		if strength < 0.30 && odds > 0.30 {
			return 0 // fold
		}
	}

	// 2. If call cost is >25% of stack, fold unless strong
	if float64(toCall) > 0.25*float64(stack) && strength < 0.40 {
		return 0
	}

	// 3. If villain raises and bot has junk, fold
	if best == 1 && strength < 0.25 && toCall > 0 { // best==CALL
		return 0 // fold weak calls
	}

	return best
}

func (dqn *DQNAgent) Learn(state []float64, action int, reward float64, nextState []float64, done bool) {
	// The 'Train' method handles everything:
	// 1. Adding to the replay buffer
	// 2. Sampling a batch
	// 3. Calculating the Bellman target
	// 4. Backpropagation
	// 5. Updating the target network
	dqn.agent.Train(state, nextState, action, reward, done)
}

func (dqn *DQNAgent) DecayEpsilon() {
	if dqn.Epsilon <= dqn.MinEpsilon {
		dqn.Epsilon = dqn.MinEpsilon
	} // Ensure it doesn't go below min

	if dqn.Epsilon > dqn.MinEpsilon {
		dqn.Epsilon *= dqn.EpsDecay
	} // Decay epsilon
}

func (dqn *DQNAgent) GetEpsilon() float64 {
	return dqn.Epsilon
}

func (dqn *DQNAgent) SetEpsilon(epsilon float64) {
	dqn.Epsilon = epsilon
}

// SaveModel saves the DQN model to a gob file
func (agent *DQNAgent) SaveModel(filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return agent.agent.Save(f)
}

// LoadModel loads the DQN model from a gob file
func (agent *DQNAgent) LoadModel(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return agent.agent.Load(f)
}

// GetWeights exposes the *inner* network's GetWeights method
func (dqn *DQNAgent) GetWeights() map[string]interface{} {
	// We're assuming the 'agent' struct has a public 'qNetwork'
	// or that you added GetWeights() to the dqn.DQN struct itself.
	// Let's assume you added it to dqn.DQN, which calls its internal qNetwork.
	return dqn.agent.GetWeights()
}

// SetWeights exposes the *inner* network's SetWeights method
func (dqn *DQNAgent) SetWeights(weights map[string]interface{}) {
	dqn.agent.SetWeights(weights)
}
