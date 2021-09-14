# Multiverse Simulation

The biggest breakthrough of Bitcoin was the introduction of a new voting scheme on top of a blockchain - a data structure which was invented by Stuart Haber and W. Scott Stornetta in 1991. The blocks that are issued by block producers do not just contain the transactions that are added to the ledger but also contain a reference to the previous block. Through this reference, the blocks form a chain where every block implicitly approves all of the previous blocks and represent a vote by the issuer on what he perceives to be the longest chain. The chain that received the most votes (blocks) wins.

This form of voting, where the messages that introduce new transactions also contain a vote by the issuer is called virtual voting. Instead of having to agree upfront on the participants of the network and making every node exchange votes with every other node to reach consensus, it relies on the idea of piling up approval by validators on past decisions by inheriting the votes of future messages using a data structure like the blockchain.

This did not just solve the huge message complexity of existing consensus algorithms and opened up the network to a much larger number of nodes, but by combining it with Proof of Work, it also enabled the network to become open and permissionless.

The real beauty of this voting scheme is not just its efficiency but also its flexibility. Instead of being limited to just Proof of Work, it gives you complete freedom over the way how the block producers are chosen. This kicked off a whole field of research trying to choose block producers more efficiently. Today we have PoW chains (Bitcoin), PoS chains (Cardano), VDF chains (Solana), permissioned chains (Hyperledger), semi-permissioned chains (EOS) and all kinds of other variants.

This level of freedom and flexibility is the reason why 99% of all DLTs still use a blockchain and very few have tried to use a different approach. Especially in a heavily researched and fast moving field like DLT it is crucial to have a technical foundation that can adapt to future developments and incorporate new findings without breaking the protocol.

There have been efforts by projects like Hashgraph to translate the concepts of virtual voting into the world of DAGs but they have failed to maintain the same properties as blockchain and are limited to a relatively small network of permissioned nodes. Other DAG based DLTs are very proprietary, often make undesirable tradeoffs and only work exactly the way they were designed. It is impossible to modify even small parts of their protocol and exchange them with a different one as research progresses.

IOTA was one of the first projects that tried to translate the longest chain wins consensus of blockchains into the world of DAGs maintaining all of its benefits but trying to solve blockchains drawbacks (slow confirmations, hard to shard and relying on a 2-class society where users have to pay miners to get their transactions included in the ledger state). It failed to fulfill this promise due to a badly designed and broken first version. 

This repository implements a simulator for a new consensus mechanism that gets rid of all the original drawbacks of IOTA and that similarly to blockchain does not rely on nodes querying each other for their opinion.


## What is being simulated?
 
A configurable network of *N* nodes connected to each other in a [Watts-Strogatz](https://en.wikipedia.org/wiki/Watts%E2%80%93Strogatz_model) graph, 
where nodes are assigned weights according to a [Zipf distribution](https://en.wikipedia.org/wiki/Zipf%27s_law).
Each peer in the network can send messages at a rate proportional to its weight. The messages attach to other messages in the tangle according to a configurable tip-selection algorithm.
The simulation tracks the weight of each message, the color and color weight, and tip pool size.

To see the full list of configurations one should run the simulation with `-h` flag.

## Message Weight Mechanism

When a new message is issued by node *A* it automatically receives the weight of *A*. It also propagates the weight down to its parents, 
cumulating the weight to each message in the past cone until genesis is reached. 
Each message keeps track of its weight source in an efficient bitmap to ensure the correctness of the weight propagation calculation.
This mechanism doesn't take into account different perceptions of the tangle a node has.
This calculation can be done by every node in the simulation. Once a message bypassed a threshold of the weight (above 50%) we can consider it as *confirmed* or *seen*,
depending on whether we also simulate colored perceptions in our run.

## Color Weight Mechanism

In order to take into account different conflict perceptions we assign colors to subtangles. 
This mechanism works by having a node that colors messages.
Each message propagates its color to its descendants ad infinitum. If a message has parents with different colors the node deems it invalid and drops it.
Due to this, it is worth to note, coloring the tangle will create distinct subtangles that can't be combined.
This basically means that we are creating a simulation of the UTXO dag instead of the message dag.
Each node in the simulation keeps track of the weight of the colors. Each time a message is colored, the weight of its issuer is added to the color weight.
If the weight of the issuer was previously assigned to another color, it is subtracted from the color. 
Each node in the simulation considers the color with the most weight to be the winner. 
Each color is actually a number in the actual implementation. In case of a tie the color with the maximal number will be the winner. 

## Tip selection

Each message in the simulation can choose up to a configurable *k* other message to reference. 
They will usually pick parents that weren't referenced before, known as tips. 
Tip selection plays a great deal with the way weights are distributed, and in the simulation we will implement various 
tip-selection strategies, honest and malicious. Currently, only 2 tip selection strategies are implemented. 
URTS (Uniform Random Tip Selection) and RURTS (Restrictedk URTS). URTS, as the name implies, randomly selects any tip.
RURTS won't select tips that have aged above a configurable delta. All other tips will be selected uniformly.


## Running the simulation

It is best run via a script that will plot the results per the instructions [here](https://github.com/iotaledger/multiverse-simulation/blob/aw/scripts/README.md).
But one can naively run the simulation with a `go run .` command.