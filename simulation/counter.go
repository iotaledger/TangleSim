package simulation

import (
	"fmt"
	"sync"

	"github.com/iotaledger/hive.go/core/generics/constraints"
	"github.com/iotaledger/multivers-simulation/network"

	"github.com/iotaledger/multivers-simulation/multiverse"
)

// Generic counter ////////////////////////////////////////////////////////////////////////////////////////////////////

// CounterElements defines types that can be used as elements by the generic Counters.
type CounterElements interface {
	multiverse.Color | network.PeerID | string
}

// MapCounters is a generic counter that can be used to count any type of elements.
type MapCounters[T CounterElements, V constraints.Integer] struct {
	counters     map[string]map[T]V
	counterMutex sync.RWMutex
}

// NewCounters creates a new Counters.
func NewCounters[T CounterElements, V constraints.Integer]() *MapCounters[T, V] {
	return &MapCounters[T, V]{
		counters: make(map[string]map[T]V),
	}
}

// CreateCounter creates a new counting map under provided counterKey, with provided elements and optional initial values.
func (c *MapCounters[T, V]) CreateCounter(counterKey string, elements []T, values ...V) {
	c.counterMutex.Lock()
	defer c.counterMutex.Unlock()
	if len(elements) == 0 {
		return
	}
	if _, ok := c.counters[counterKey]; !ok {
		c.counters[counterKey] = make(map[T]V)
	}
	for _, element := range elements {
		if len(values) == len(elements) {
			c.counters[counterKey][element] = values[0]
		} else if len(elements) > 1 && len(values) == 1 {
			c.counters[counterKey][element] = values[0]
		} else {
			c.counters[counterKey][element] = 0
		}
	}
}

func (c *MapCounters[T, V]) Add(counterKey string, value V, element T) {
	c.counterMutex.Lock()
	defer c.counterMutex.Unlock()
	counter, ok := c.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying add to not initiated counter, key: %s, element: %s", counterKey, element))
	}
	counter[element] += value
}

func (c *MapCounters[T, V]) Set(counterKey string, value V, element T) {
	c.counterMutex.Lock()
	defer c.counterMutex.Unlock()
	counter, ok := c.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying set for not initiated counter, key: %s, element: %s", counterKey, element))
	}
	counter[element] = value
}

func (c *MapCounters[T, V]) Get(counterKey string, element T) V {
	c.counterMutex.RLock()
	defer c.counterMutex.RUnlock()
	counter, ok := c.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s, element: %s", counterKey, element))
	}
	return counter[element]
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region AtomicCounters ////////////////////////////////////////////////////////////////////////////////////////////////

type AtomicCounters[T CounterElements, V constraints.Integer] struct {
	counters      map[T]V
	countersMutex sync.RWMutex
}

func NewAtomicCounters[T CounterElements, V constraints.Integer]() *AtomicCounters[T, V] {
	return &AtomicCounters[T, V]{
		counters: make(map[T]V),
	}
}

func (ac *AtomicCounters[T, V]) CreateCounter(counterKey T, initValue V) {
	ac.countersMutex.Lock()
	defer ac.countersMutex.Unlock()
	// if key not exist create new counter
	if _, ok := ac.counters[counterKey]; !ok {
		ac.counters[counterKey] = initValue
	}
}

func (ac *AtomicCounters[T, V]) Get(counterKey T) V {
	ac.countersMutex.RLock()
	defer ac.countersMutex.RUnlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}
	return counter
}

func (ac *AtomicCounters[T, V]) Add(counterKey T, value V) {
	ac.countersMutex.Lock()
	defer ac.countersMutex.Unlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying add to not initiated counter, key: %s", counterKey))
	}
	counter += value
}

func (ac *AtomicCounters[T, V]) Set(counterKey T, value V) {
	ac.countersMutex.Lock()
	defer ac.countersMutex.Unlock()
	_, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying set for not initiated counter, key: %s", counterKey))
	}
	ac.counters[counterKey] = value
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
