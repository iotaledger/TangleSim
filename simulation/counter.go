package simulation

import (
	"fmt"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"sync"
)

type Counters struct {
	intCounts map[string]map[multiverse.Color]int
	mutexes   map[string]sync.RWMutex
}

func NewCounters() *Counters {
	return &Counters{
		intCounts: make(map[string]map[multiverse.Color]int),
		mutexes:   make(map[string]sync.RWMutex),
	}
}

// CreateCounter Adds new counter with key and provided initial conditions. Allowed types: int, float64
func (c *Counters) CreateCounter(counterKey string, colors []multiverse.Color, mutexEnabled bool, initValues []int) {
	if len(initValues) == 0 {
		return
	}
	// if key not exist create new map
	if innerMap, ok := c.intCounts[counterKey]; !ok {
		innerMap = make(map[multiverse.Color]int)
		for i, color := range colors {
			innerMap[color] = initValues[i]
		}
		c.intCounts[counterKey] = innerMap
		log.Info("Int Counter created ", counterKey)
	}

	if mutexEnabled {
		c.mutexes[counterKey] = sync.RWMutex{}
	}
}

func (c *Counters) Add(counterKey string, value int, color multiverse.Color) {
	if mu, ok := c.mutexes[counterKey]; ok {
		mu.Lock()
		defer mu.Unlock()
	}
	innerMap, ok := c.intCounts[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying add to not initiated counter, key: %s, color: %s", counterKey, color))
	}
	innerMap[color] += value
}

// Get gets the counter value for provided key and color.
func (c *Counters) Get(counterKey string, color multiverse.Color) int {
	innerMap, ok := c.intCounts[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get value for not initiated counter, key: %s, color: %s", counterKey, color))
	}
	return innerMap[color]
}
