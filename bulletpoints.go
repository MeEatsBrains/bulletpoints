// bulletpoints allows the simulation of the bullseye loot event for the mobile game last war
package bulletpoints

import (
	"math/rand"
	"time"
)

// SimuMode allows to switch the simulation rng to three possible values:
//   - normalMode for a standard simulation
//   - allFail to fail all rolls
//   - allSuccess to succeed all rolls
type SimuMode int

const (
	NormalMode SimuMode = iota
	AllFail
	AllSuccess
)

// Simu is the interface to access the bullseye loot event simulation, a valid implementation
// of this interface me be obtained by calling NewSimu
type Simu interface {
	Simulate(seed int) int
}

// simuImpl is the internal implementation of the Simu interface.
type simuImpl struct {
	mode SimuMode
	r    *rand.Rand
}

// NewSimu generates a new simulation interface
func NewSimu(mode SimuMode) Simu {
	return &simuImpl{
		mode: mode,
	}
}

// Simulate calculates the number of bullets required to reach stage 100
func (s *simuImpl) Simulate(seed int) int {
	// either specify a seed or seed with the time if 0 is given
	if seed == 0 {
		seed = int(time.Now().UnixMicro())
	}

	// generate a seeded random source
	src := rand.NewSource(int64(seed))
	s.r = rand.New(src)

	// bullets will hold the total number of required bullets
	bullets := 0

	// iterate over the 100 stages and add the amount of bullets used, stage is 0 based, i.e. it goes from 0 to 99
	for stage := range 100 {
		bullets += s.simulateStage(stage)
	}

	return bullets
}

// simulateStage returns the bullets used up in a single stage, stage is expected to be zero based so 0 is what the
// game calls level 1
func (s *simuImpl) simulateStage(stage int) int {
	// the board has initially 11 minor prices and 1 major prize with a weight of 1
	minor := 11
	major := 1

	attempts := 0
	for {
		// next attempt
		attempts++
		// return the attempts (= bullets) required to get the hit on the major prize
		if s.hitsMajor(major, minor, attempts, stage) {
			return attempts
		}

		// in case of a miss, one minor price is taken off the board
		minor--
		// after 3, 6, and 9 attempts the weight for the major prize is increased
		switch attempts {
		case 3:
			major = 5
		case 6:
			major = 10
		case 9:
			major = 30
		}
	}
}

// hitsMajor simulates if a major prize is hit.
// The parameters are major (the weight of the major price), minor (the number of minor prices on the board),
// attempt (the current attempt for this level), and stage (0 based level number)
func (s *simuImpl) hitsMajor(major, minor, attempt, stage int) bool {
	// decide randomly between the weighted major price and the minor prices, a random number must within this range
	rndRange := major + minor

	// generate a random number or set it up for the allSuccess and allFail modes
	var rndNumber int
	switch s.mode {
	case NormalMode:
		rndNumber = s.r.Intn(rndRange)
	case AllSuccess:
		rndNumber = rndRange - 1
	default:
		rndNumber = 0
	}

	// After the nth attempt you will always get the major price. When this happens varies with the stage but the
	// pattern repeats every 5 levels
	hardLimit := []int{5, 6, 8, 5, 10}
	if attempt > hardLimit[stage%5] {
		return true
	}

	// let's assume we had 5 minor prizes and a major weight of 10, the random number will be between 0 and 14.
	// 0, 1, 2, 3, 4 represent a minor hit and 5 to 14 a major hit. So, we score a major hit if the random number is
	// equal or larger than the amount of minor targets.
	return rndNumber >= minor
}
