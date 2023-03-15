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

// RegisterCounters creates a new counting map under provided counterKey, with provided elements and optional initial values.
func (c *MapCounters[T, V]) RegisterCounters(counterKey string, elements []T, values ...V) {
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

func (ac *AtomicCounters[T, V]) RegisterCounter(counterKey T, initValue V) {
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
