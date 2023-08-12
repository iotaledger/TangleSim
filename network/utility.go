package network

import "math"

func ZIPFDistribution(s float64) WeightGenerator {
	return func(nodeCount int, totalWeight float64) (result []uint64) {
		rawTotalWeight := uint64(0)
		rawWeights := make([]uint64, nodeCount)
		for i := 0; i < nodeCount; i++ {
			weight := uint64(math.Pow(float64(i+1), -s) * totalWeight)
			rawWeights[i] = weight
			rawTotalWeight += weight
		}

		normalizedTotalWeight := uint64(0)
		result = make([]uint64, nodeCount)
		for i := 0; i < nodeCount; i++ {
			normalizedWeight := uint64((float64(rawWeights[i]) / float64(rawTotalWeight)) * totalWeight)

			result[i] = normalizedWeight
			normalizedTotalWeight += normalizedWeight
		}

		result[0] += uint64(totalWeight) - normalizedTotalWeight

		return
	}
}

func EqualDistribution(validatorNodeCount int, nonValidatorNodeCount int, totalWeight int) (result []uint64) {

	normalizedTotalWeight := uint64(0)
	result = make([]uint64, validatorNodeCount+nonValidatorNodeCount)
	for i := 0; i < validatorNodeCount; i++ {
		normalizedWeight := uint64((float64(totalWeight) / float64(validatorNodeCount)))

		result[i] = normalizedWeight
		normalizedTotalWeight += normalizedWeight
	}
	result[0] += uint64(totalWeight) - normalizedTotalWeight
	return result
}

// The mixed ZIPF distribution, where the first validatorNodeCount nodes
// have equal weight, and the remaining nonValidatorNodeCount nodes' weights
// follow ZIPF distribution.
// Note that the bandwidth generated here only consider nonValidation blocks,
// even if the blocks are issued from validator nodes.
// The reason is that there is additional ticker in the main function that
// only for issuing the validation blocks.
func MixedZIPFDistribution(s float64) BandwidthGenerator {
	return func(
		validatorNodeCount int,
		nonValidatorNodeCount int,
		validatorCommitteeBandwidth float64,
		nonValidatorCommitteeBandwidth float64,

	) (result []float64) {

		result = make([]float64, validatorNodeCount+nonValidatorNodeCount)
		totalBandwidthPerValidator := validatorCommitteeBandwidth / float64(validatorNodeCount)
		for i := 0; i < validatorNodeCount; i++ {
			result[i] = totalBandwidthPerValidator
		}

		rawTotalBandwidth := uint64(0)
		rawBandwidth := make([]uint64, nonValidatorNodeCount)
		for i := 0; i < nonValidatorNodeCount; i++ {
			bandwidth := uint64(math.Pow(float64(i+1), -s) * nonValidatorCommitteeBandwidth)
			rawBandwidth[i] = bandwidth
			rawTotalBandwidth += bandwidth
		}

		normalizedTotalBandwidth := float64(0)
		for i := 0; i < nonValidatorNodeCount; i++ {
			normalizedBandwidth := (float64(rawBandwidth[i]) / float64(rawTotalBandwidth)) * nonValidatorCommitteeBandwidth

			result[i+validatorNodeCount] = normalizedBandwidth
			normalizedTotalBandwidth += normalizedBandwidth
		}

		result[validatorNodeCount] += nonValidatorCommitteeBandwidth - normalizedTotalBandwidth

		return
	}
}
