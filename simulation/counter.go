package simulation

import (
	"fmt"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"go.uber.org/atomic"
	"sync"
)

// region AtomicCounters ////////////////////////////////////////////////////////////////////////////////////////////////

type AtomicCounters struct {
	counters map[string]*atomic.Int64
}

func NewAtomicCounters() *AtomicCounters {
	return &AtomicCounters{
		counters: make(map[string]*atomic.Int64),
	}
}

func (ac *AtomicCounters) CreateAtomicCounter(counterKey string, initValue int64) {
	// if key not exist create new counter
	if _, ok := ac.counters[counterKey]; !ok {
		ac.counters[counterKey] = atomic.NewInt64(initValue)
	}
}

func (ac *AtomicCounters) Get(counterKey string) int64 {
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}
	return counter.Load()
}

func (ac *AtomicCounters) Add(counterKey string, value int64) {
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying add to not initiated counter, key: %s", counterKey))
	}
	counter.Add(value)
}

func (ac *AtomicCounters) Set(counterKey string, value int64) {
	_, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying set for not initiated counter, key: %s", counterKey))
	}
	ac.counters[counterKey] = atomic.NewInt64(value)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region ColorCounters ////////////////////////////////////////////////////////////////////////////////////////////////

type ColorCounters struct {
	counts map[string]map[multiverse.Color]int64
	mu     sync.RWMutex
}

func NewColorCounters() *ColorCounters {
	return &ColorCounters{
		counts: make(map[string]map[multiverse.Color]int64),
	}
}

// CreateCounter Adds new counter with key and provided initial conditions.
func (c *ColorCounters) CreateCounter(counterKey string, colors []multiverse.Color, initValues []int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(initValues) == 0 {
		return
	}
	// if key not exist create new map
	if innerMap, ok := c.counts[counterKey]; !ok {
		innerMap = make(map[multiverse.Color]int64)
		for i, color := range colors {
			innerMap[color] = initValues[i]
		}
		c.counts[counterKey] = innerMap
	}
}

func (c *ColorCounters) Add(counterKey string, value int64, color multiverse.Color) {
	c.mu.Lock()
	defer c.mu.Unlock()
	innerMap, ok := c.counts[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying add to not initiated counter, key: %s, color: %s", counterKey, color))
	}
	innerMap[color] += value
}

func (c *ColorCounters) Set(counterKey string, value int64, color multiverse.Color) {
	c.mu.Lock()
	defer c.mu.Unlock()
	innerMap, ok := c.counts[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying set the not initiated counter value, key: %s, color: %s", counterKey, color))
	}
	innerMap[color] = value
}

// Get gets the counter value for provided key and color.
func (c *ColorCounters) Get(counterKey string, color multiverse.Color) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	innerMap, ok := c.counts[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get value for not initiated counter, key: %s, color: %s", counterKey, color))
	}
	return innerMap[color]
}

func (c *ColorCounters) GetInt(counterKey string, color multiverse.Color) int {
	v := c.Get(counterKey, color)
	return int(v)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
