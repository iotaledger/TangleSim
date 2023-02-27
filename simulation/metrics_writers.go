package simulation

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/typeutils"
	"github.com/iotaledger/multivers-simulation/config"
	"github.com/iotaledger/multivers-simulation/multiverse"
	"github.com/iotaledger/multivers-simulation/network"
)

type csvRows [][]string

func singleRowFunc(row []string) func() csvRows {
	return func() csvRows {
		return csvRows{row}
	}
}

// SetupWriters sets up the csv writers for the simulation.
func (s *MetricsManager) SetupWriters() {
	s.DumpOnce("ad",
		[]string{"AdversaryGroupID", "Strategy", "AdversaryCount", "q", "ns since issuance"},
		func() csvRows {
			rows := make(csvRows, 0)
			for groupID, group := range s.network.AdversaryGroups {
				row := []string{
					strconv.FormatInt(int64(groupID), 10),
					network.AdversaryTypeToString(group.AdversaryType),
					strconv.FormatInt(int64(len(group.NodeIDs)), 10),
					strconv.FormatFloat(group.GroupMana/float64(config.NodesTotalWeight), 'f', 6, 64),
					strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10),
				}
				rows = append(rows, row)
			}
			return rows
		},
	)
	s.DumpOnceStreaming("nw",
		[]string{"Peer ID", "Neighbor ID", "Network Delay (ns)", "Packet Loss (%)", "Weight"},
		func() <-chan []string {
			c := make(chan []string)
			go func() {
				for _, peer := range s.network.Peers {
					for neighbor, connection := range peer.Neighbors {
						record := []string{
							strconv.FormatInt(int64(peer.ID), 10),
							strconv.FormatInt(int64(neighbor), 10),
							strconv.FormatInt(connection.NetworkDelay().Nanoseconds(), 10),
							strconv.FormatInt(int64(connection.PacketLoss()*100), 10),
							strconv.FormatInt(int64(s.network.WeightDistribution.Weight(peer.ID)), 10),
						}
						// At this moment csv writer will be flushed
						c <- record
					}
				}
				// remember to close the channel
				close(c)
			}()
			return c
		},
	)

	s.DumpOnTick("ds",
		[]string{"UndefinedColor", "Blue", "Red", "Green", "ns since start", "ns since issuance"},
		singleRowFunc([]string{
			strconv.FormatInt(s.ColorCounters.Get("opinionsWeights", multiverse.UndefinedColor), 10),
			strconv.FormatInt(s.ColorCounters.Get("opinionsWeights", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("opinionsWeights", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("opinionsWeights", multiverse.Green), 10),
			strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10),
			s.sinceDSIssuanceTimeStr(),
		}),
	)
	s.DumpOnTick("tp",
		[]string{"UndefinedColor (Tip Pool Size)", "Blue (Tip Pool Size)", "Red (Tip Pool Size)", "Green (Tip Pool Size)",
			"UndefinedColor (Processed)", "Blue (Processed)", "Red (Processed)", "Green (Processed)", "# of Issued Messages", "ns since start"},
		singleRowFunc([]string{
			strconv.FormatInt(s.ColorCounters.Get("tipPoolSizes", multiverse.UndefinedColor), 10),
			strconv.FormatInt(s.ColorCounters.Get("tipPoolSizes", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("tipPoolSizes", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("tipPoolSizes", multiverse.Green), 10),
			strconv.FormatInt(s.ColorCounters.Get("processedMessages", multiverse.UndefinedColor), 10),
			strconv.FormatInt(s.ColorCounters.Get("processedMessages", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("processedMessages", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("processedMessages", multiverse.Green), 10),
			strconv.FormatInt(s.GlobalCounters.Get("issuedMessages"), 10),
			strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10),
		}),
	)
	s.DumpOnTick("mm",
		[]string{"Number of Requested Messages", "ns since start"},
		singleRowFunc([]string{
			strconv.FormatInt(s.ColorCounters.Get("requestedMissingMessages", multiverse.UndefinedColor), 10),
			strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10),
		}),
	)
	s.DumpOnTick("cc", []string{"Blue (Confirmed)", "Red (Confirmed)", "Green (Confirmed)",
		"Blue (Adversary Confirmed)", "Red (Adversary Confirmed)", "Green (Adversary Confirmed)",
		"Blue (Confirmed Accumulated Weight)", "Red (Confirmed Accumulated Weight)", "Green (Confirmed Accumulated Weight)",
		"Blue (Confirmed Adversary Weight)", "Red (Confirmed Adversary Weight)", "Green (Confirmed Adversary Weight)",
		"Blue (Like)", "Red (Like)", "Green (Like)",
		"Blue (Like Accumulated Weight)", "Red (Like Accumulated Weight)", "Green (Like Accumulated Weight)",
		"Blue (Adversary Like Accumulated Weight)", "Red (Adversary Like Accumulated Weight)", "Green (Adversary Like Accumulated Weight)",
		"Unconfirmed Blue", "Unconfirmed Red", "Unconfirmed Green",
		"Unconfirmed Blue Accumulated Weight", "Unconfirmed Red Accumulated Weight", "Unconfirmed Green Accumulated Weight",
		"Flips (Winning color changed)", "Honest nodes Flips", "ns since start", "ns since issuance"},
		singleRowFunc([]string{
			strconv.FormatInt(s.ColorCounters.Get("confirmedNodes", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("confirmedNodes", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("confirmedNodes", multiverse.Green), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("confirmedNodes", multiverse.Blue), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("confirmedNodes", multiverse.Red), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("confirmedNodes", multiverse.Green), 10),
			strconv.FormatInt(s.ColorCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Blue), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Red), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("confirmedAccumulatedWeight", multiverse.Green), 10),
			strconv.FormatInt(s.ColorCounters.Get("opinions", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("opinions", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("opinions", multiverse.Green), 10),
			strconv.FormatInt(s.ColorCounters.Get("likeAccumulatedWeight", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("likeAccumulatedWeight", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("likeAccumulatedWeight", multiverse.Green), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("likeAccumulatedWeight", multiverse.Blue), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("likeAccumulatedWeight", multiverse.Red), 10),
			strconv.FormatInt(s.AdversaryCounters.Get("likeAccumulatedWeight", multiverse.Green), 10),
			strconv.FormatInt(s.ColorCounters.Get("colorUnconfirmed", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("colorUnconfirmed", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("colorUnconfirmed", multiverse.Green), 10),
			strconv.FormatInt(s.ColorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Blue), 10),
			strconv.FormatInt(s.ColorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Red), 10),
			strconv.FormatInt(s.ColorCounters.Get("unconfirmedAccumulatedWeight", multiverse.Green), 10),
			strconv.FormatInt(s.GlobalCounters.Get("flips"), 10),
			strconv.FormatInt(s.GlobalCounters.Get("honestFlips"), 10),
			strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10),
			s.sinceDSIssuanceTimeStr(),
		}),
	)
	s.DumpOnEvent("ww",
		[]string{"Witness Weight", "Time (ns)"},
		func() <-chan []string {
			c := make(chan []string)

			wwPeer := s.network.Peers[config.MonitoredWitnessWeightPeer]
			previousWitnessWeight := uint64(config.NodesTotalWeight)
			wwPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageWitnessWeightUpdated.Attach(
				events.NewClosure(func(message *multiverse.Message, weight uint64) {
					if previousWitnessWeight == weight {
						return
					}
					previousWitnessWeight = weight
					record := []string{
						strconv.FormatUint(weight, 10),
						strconv.FormatInt(time.Since(message.IssuanceTime).Nanoseconds(), 10),
					}
					// writing to the file
					c <- record
				}))
			return c
		},
	)
	s.DumpOnEvent("aw",
		[]string{"Message ID", "Issuance Time (unix)", "Confirmation Time (ns)", "Weight", "# of Confirmed Messages",
			"# of Issued Messages", "ns since start"},
		func() <-chan []string {
			c := make(chan []string)

			for _, id := range s.watchedPeerIDs {
				awPeer := s.network.Peers[id]
				if typeutils.IsInterfaceNil(awPeer) {
					panic(fmt.Sprintf("unknowm peer with id %d", id))
				}
				awPeer.Node.(multiverse.NodeInterface).Tangle().ApprovalManager.Events.MessageConfirmed.Attach(
					events.NewClosure(func(message *multiverse.Message, messageMetadata *multiverse.MessageMetadata, weight uint64, messageIDCounter int64) {
						// increased here and not in metrics setup to rice between counter attached there and read here
						s.PeerCounters.Add("confirmedMessageCounter", 1, awPeer.ID)
						record := []string{
							strconv.FormatInt(int64(message.ID), 10),
							strconv.FormatInt(message.IssuanceTime.Unix(), 10),
							strconv.FormatInt(int64(messageMetadata.ConfirmationTime().Sub(message.IssuanceTime)), 10),
							strconv.FormatUint(weight, 10),
							strconv.FormatInt(s.PeerCounters.Get("confirmedMessageCounter", awPeer.ID), 10),
							strconv.FormatInt(messageIDCounter, 0),
							strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10),
						}
						// writing to the file
						c <- record
					}))
			}
			return c
		},
	)
	s.DumpOnTick("all-tp", allNodesHeader(),
		func() csvRows {
			record := make([]string, config.NodesCount+1)
			i := 0
			for peerID := 0; peerID < config.NodesCount; peerID++ {
				tipCounterName := fmt.Sprint("tipPoolSizes-", peerID)
				record[i+0] = strconv.FormatInt(s.ColorCounters.Get(tipCounterName, multiverse.UndefinedColor), 10)
				i = i + 1
			}
			record[i] = strconv.FormatInt(time.Since(s.simulationStartTime).Nanoseconds(), 10)
			return csvRows{record}
		},
	)
	s.DumpOnShutdown("nd",
		[]string{"Node ID", "Adversary", "Min Confirmed Accumulated Weight", "Unconfirmation Count"},
		func() csvRows {
			records := make(csvRows, config.NodesCount)
			for i := 0; i < config.NodesCount; i++ {
				record := []string{
					strconv.FormatInt(int64(i), 10),
					strconv.FormatBool(network.IsAdversary(int(i))),
					strconv.FormatInt(s.PeerCounters.Get("minConfirmedAccumulatedWeight", network.PeerID(i)), 10),
					strconv.FormatInt(s.PeerCounters.Get("unconfirmationCount", network.PeerID(i)), 10),
				}
				records[i] = record
			}
			return records
		},
	)
}

// DumpOnTick registers a new writer to the metrics manager.
func (s *MetricsManager) DumpOnTick(key string, header []string, collectFunc func() csvRows) {
	resultsWriter := s.createWriter(key, header)
	s.writers[key] = resultsWriter
	s.collectFuncs[key] = collectFunc
	if err := resultsWriter.Write(header); err != nil {
		panic(err)
	}
}

// DumpOnce dumps the results once.
func (s *MetricsManager) DumpOnce(key string, header []string, collectFunc func() csvRows) {
	resultsWriter := s.createWriter(key, header)

	rows := collectFunc()

	for _, row := range rows {
		if err := resultsWriter.Write(row); err != nil {
			log.Fatal("error writing record to csv:", err)
		}
	}

	if err := resultsWriter.Error(); err != nil {
		log.Fatal(err)
	}
	resultsWriter.Flush()
}

// DumpOnceStreaming dumps the results row by row, use for large files.
func (s *MetricsManager) DumpOnceStreaming(key string, header []string, collectFunc func() <-chan []string) {
	resultsWriter := s.createWriter(key, header)

	for row := range collectFunc() {
		if err := resultsWriter.Write(row); err != nil {
			log.Fatal("error writing record to csv:", err)
		}
		resultsWriter.Flush()
	}

	if err := resultsWriter.Error(); err != nil {
		log.Fatal(err)
	}
}

func (s *MetricsManager) DumpOnEvent(key string, header []string, setupEventFunc func() <-chan []string) {
	resultsWriter := s.createWriter(key, header)
	mu := &sync.Mutex{}

	go func() {
		for {
			select {
			case row := <-setupEventFunc():
				mu.Lock()
				if err := resultsWriter.Write(row); err != nil {
					log.Fatal("error writing record to csv:", err)
				}
				resultsWriter.Flush()
				mu.Unlock()
				if err := resultsWriter.Write(header); err != nil {
					panic(err)
				}
			case <-s.shutdown:
				return
			}
		}
	}()
}

func (s *MetricsManager) DumpOnShutdown(key string, header []string, collectFunc func() csvRows) {
	s.onShutdownDumpers = append(s.onShutdownDumpers, func() {
		s.DumpOnce(key, header, collectFunc)
	})
}

func (s *MetricsManager) createWriter(key string, header []string) *csv.Writer {
	filename := fmt.Sprintf("%s-%s.csv", key, formatTime(s.simulationStartTime))
	file, err := os.Create(path.Join(config.ResultDir, filename))
	if err != nil {
		panic(err)
	}

	resultsWriter := csv.NewWriter(file)
	if err = resultsWriter.Write(header); err != nil {
		panic(err)
	}
	return resultsWriter
}

func (s *MetricsManager) sinceDSIssuanceTimeStr() string {
	if s.dsIssuanceTime.IsZero() {
		return "0"
	}
	return strconv.FormatInt(time.Since(s.dsIssuanceTime).Nanoseconds(), 10)
}
