package bot

import (
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

func (dqn *DQNAgent) Learn(state []float64, action int, reward int, nextState []float64, done bool) {
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
