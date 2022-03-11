package simulation

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/iotaledger/multivers-simulation/multiverse"
	"go.uber.org/atomic"
)

// region AtomicUnboundedColorCounter ////////////////////////////////////////////////////////////////////////////////////////////////

type AtomicUnboundedColorCounter struct {
	counters      map[string]map[multiverse.Color]*atomic.Int64
	countersMutex sync.RWMutex
}

func NewAtomicUnboundedColorCounters() *AtomicUnboundedColorCounter {
	return &AtomicUnboundedColorCounter{
		counters: make(map[string]map[multiverse.Color]*atomic.Int64),
	}
}

func (ac *AtomicUnboundedColorCounter) CreateAtomicColorCounter(counterKey string) {
	ac.countersMutex.Lock()
	defer ac.countersMutex.Unlock()
	// if key not exist create new counter
	if _, ok := ac.counters[counterKey]; !ok {
		ac.counters[counterKey] = make(map[multiverse.Color]*atomic.Int64)
	}
}

func (ac *AtomicUnboundedColorCounter) Get(counterKey string, color multiverse.Color) int64 {
	ac.countersMutex.RLock()
	defer ac.countersMutex.RUnlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}
	_, ok = counter[color]
	if !ok {
		counter[color] = atomic.NewInt64(0)
	}
	return counter[color].Load()
}

func (ac *AtomicUnboundedColorCounter) GetHonestColorWithMaxCount(counterKey string, adversaryCounters *AtomicUnboundedColorCounter) multiverse.Color {
	ac.countersMutex.RLock()
	adversaryCounters.countersMutex.RLock()
	defer ac.countersMutex.RUnlock()
	defer adversaryCounters.countersMutex.RUnlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}
	adCounter, ok := adversaryCounters.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}

	colorWithMaxValue := multiverse.UndefinedColor
	maxCount := int64(0)
	for color, count := range counter {
		currentCount := count.Load()
		adCount := int64(0)
		if val, ok := adCounter[color]; ok {
			adCount = val.Load()
		}
		currentCount -= adCount
		if currentCount >= maxCount {
			colorWithMaxValue = color
			maxCount = currentCount
		}
	}
	return colorWithMaxValue
}

func (ac *AtomicUnboundedColorCounter) GetColorNodes(counterKey string) string {
	ac.countersMutex.RLock()
	defer ac.countersMutex.RUnlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("Network Status: %s Consensus[ ", counterKey))
	for color, count := range counter {
		currentCount := count.Load()
		// if currentCount > 100 {
		// 	panic(fmt.Sprintf("Color %s has count %d", color.String(), currentCount))
		// }
		if currentCount != 0 {
			b.WriteString(fmt.Sprintf(" %3d %s", currentCount, color.String()))
		}
	}
	b.WriteString(" ]")
	return b.String()
}

func (ac *AtomicUnboundedColorCounter) GetColorWithMaxCount(counterKey string) (multiverse.Color, int64) {
	ac.countersMutex.RLock()
	defer ac.countersMutex.RUnlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying get from not initiated counter, key: %s", counterKey))
	}
	colorWithMaxValue := multiverse.UndefinedColor
	maxCount := int64(0)
	for color, count := range counter {
		currentCount := count.Load()
		log.Debugf("GetHonestColorWithMaxCount color %s count %d", color.String(), currentCount)
		if currentCount >= maxCount {
			colorWithMaxValue = color
			maxCount = currentCount
		}
	}
	return colorWithMaxValue, maxCount
}

func (ac *AtomicUnboundedColorCounter) Add(counterKey string, color multiverse.Color, value int64) {
	ac.countersMutex.Lock()
	defer ac.countersMutex.Unlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying add to not initiated counter, key: %s", counterKey))
	}
	_, ok = counter[color]
	if !ok {
		counter[color] = atomic.NewInt64(0)
	}
	counter[color].Add(value)
}

func (ac *AtomicUnboundedColorCounter) Set(counterKey string, color multiverse.Color, value int64) {
	ac.countersMutex.Lock()
	defer ac.countersMutex.Unlock()
	counter, ok := ac.counters[counterKey]
	if !ok {
		panic(fmt.Sprintf("Trying set for not initiated counter, key: %s", counterKey))
	}
	_, ok = counter[color]
	if !ok {
		counter[color] = atomic.NewInt64(value)
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
