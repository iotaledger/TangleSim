package fpcs

import (
	"crypto/sha256"
	"math/big"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/types"
	"github.com/iotaledger/multivers-simulation/logger"
)

func getRandomNumber(lowerBound int, upperBound int) int {
	n := rng.Intn(upperBound - lowerBound)
	return lowerBound + n
}

func getHash(color int, randomNumber int) big.Int {
	n := new(big.Int)
	bs := []byte(strconv.Itoa(color + randomNumber))
	sum := sha256.Sum256(bs)
	n.SetBytes(sum[:])
	return *n
}

// The granularity of the theta (confirmation threshold) in the rng is 0.001
const (
	granularity = 1000
)

var (
	log = logger.New("FPCS")
	rng = rand.New(rand.NewSource(99))
)

// region FPCS //////////////////////////////////////////////////////////////////////////////////////////////////////

type FPCS struct {
	FPCSTicker   *time.Ticker
	randomNumber int
	lowerBound   int
	upperBound   int
	mutex        sync.RWMutex
	shutdown     chan types.Empty
}

func NewFPCS(epochPeriod int, lowerBound int, upperBound int) *FPCS {
	fpcs := &FPCS{
		FPCSTicker:   time.NewTicker(time.Duration(epochPeriod) * time.Second),
		randomNumber: 0,
		lowerBound:   lowerBound,
		upperBound:   upperBound,
		shutdown:     make(chan types.Empty),
	}
	return fpcs
}

func (fpcs *FPCS) Run() {
	for {
		select {
		case <-fpcs.shutdown:
			break
		case <-fpcs.FPCSTicker.C:
			go fpcs.updateRandomNumber()
		}
	}
}

func (fpcs *FPCS) Shutdown() {

}

func (fpcs *FPCS) updateRandomNumber() {
	fpcs.mutex.Lock()
	defer fpcs.mutex.Unlock()
	fpcs.randomNumber = getRandomNumber(fpcs.lowerBound, fpcs.upperBound)
	log.Debugf("Generated random number: %d", fpcs.randomNumber)
}

// endregion //////////////////////////////////////////////////////////////////////////////////////////////////////
