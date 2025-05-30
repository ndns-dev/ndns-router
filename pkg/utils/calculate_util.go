package utils

import (
	"math/rand"
	"time"
)

// Calculate handles calculation operations
type Calculate struct {
	random *rand.Rand
}

// NewCalculate creates a new Calculate instance
func NewCalculate() *Calculate {
	return &Calculate{
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RandomFloat64 returns a random float64 between 0 and 1
func (c *Calculate) RandomFloat64() float64 {
	return c.random.Float64()
}

// RandomInt returns a random integer between 0 and max (exclusive)
func (c *Calculate) RandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	return c.random.Intn(max)
}
