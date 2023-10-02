package network

type BandwidthGenerator func(
	validatorNodeCount int,
	nonValidatorNodeCount int,
	validatorCommitteeBandwidth float64,
	nonValidatorCommitteeBandwidth float64,
) []float64

// region BandwidthDistribution //////////////////////////////////////////////////////////////////////////////////

type BandwidthDistribution struct {
	bandwidth        map[PeerID]float64
	totalBandwidth   float64
	largestBandwidth float64
}

func NewBandwidthDistribution() *BandwidthDistribution {
	return &BandwidthDistribution{
		bandwidth: make(map[PeerID]float64),
	}
}

func (c *BandwidthDistribution) SetBandwidth(peerID PeerID, bandwidth float64) {
	if existingBandwidth, exists := c.bandwidth[peerID]; exists {
		c.totalBandwidth -= existingBandwidth

		if c.largestBandwidth == existingBandwidth {
			c.rescanForLargestBandwidth()
		}
	}

	c.bandwidth[peerID] = bandwidth
	c.totalBandwidth += bandwidth

	if bandwidth > c.largestBandwidth {
		c.largestBandwidth = bandwidth
	}
}

func (c *BandwidthDistribution) Bandwidth(peerID PeerID) float64 {
	return c.bandwidth[peerID]
}

func (c *BandwidthDistribution) Bandwidths() map[PeerID]float64 {
	return c.bandwidth
}

func (c *BandwidthDistribution) TotalBandwidth() float64 {
	return c.totalBandwidth
}

func (c *BandwidthDistribution) LargestBandwidth() float64 {
	return c.largestBandwidth
}

func (c *BandwidthDistribution) rescanForLargestBandwidth() {
	c.largestBandwidth = 0
	for _, bandwidth := range c.bandwidth {
		if bandwidth > c.largestBandwidth {
			c.largestBandwidth = bandwidth
		}
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
