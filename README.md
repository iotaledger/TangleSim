# Multiverse Simulation

The biggest breakthrough of Bitcoin was the introduction of a new voting scheme on top of a blockchain - a data structure which was invented by Stuart Haber and W. Scott Stornetta in 1991. The blocks that are issued by block producers do not just contain the transactions that are added to the ledger but also contain a reference to the previous block. Through this reference, the blocks form a chain where every block implicitly approves all of the previous blocks and represent a vote by the issuer on what he perceives to be the longest chain. The chain that received the most votes (blocks) wins.

This form of voting, where the messages that introduce new transactions also contain a vote by the issuer is called virtual voting. Instead of having to agree upfront on the participants of the network and making every node exchange votes with every other node to reach consensus, it relies on the idea of piling up approval by validators on past decisions by inheriting the votes of future messages using a data structure like the blockchain.

This did not just solve the huge message complexity of existing consensus algorithms and opened up the network to a much larger number of nodes, but by combining it with Proof of Work, it also enabled the network to become open and permissionless.

The real beauty of this voting scheme is not just its efficiency but also its flexibility. Instead of being limited to just Proof of Work, it gives you complete freedom over the way how the block producers are chosen. This kicked off a whole field of research trying to choose block producers more efficiently. Today we have PoW chains (Bitcoin), PoS chains (Cardano), VDF chains (Solana), permissioned chains (Hyperledger), semi-permissioned chains (EOS) and all kinds of other variants.

This level of freedom and flexibility is the reason why 99% of all DLTs still use a blockchain and very few have tried to use a different approach. Especially in a heavily researched and fast moving field like DLT it is crucial to have a technical foundation that can adapt to future developments and incorporate new findings without breaking the protocol.

There have been efforts by projects like Hashgraph to translate the concepts of virtual voting into the world of DAGs but they have failed to maintain the same properties as blockchain and are limited to a relatively small network of permissioned nodes. Other DAG based DLTs are very proprietary, often make undesirable tradeoffs and only work exactly the way they were designed. It is impossible to modify even small parts of their protocol and exchange them with a different one as research progresses.

IOTA was one of the first projects that tried to translate the longest chain wins consensus of blockchains into the world of DAGs maintaining all of its benefits but trying to solve blockchains drawbacks (slow confirmations, hard to shard and relying on a 2-class society where users have to pay miners to get their transactions included in the ledger state). It failed to fulfill this promise due to a badly designed and broken first version. 

This repository implements a simulator for a new consensus mechanism that gets rid of the drawbacks of the original ideas of IOTA and implements a scalable and fast consensus mechanism that similarly to blockchain does not rely on nodes querying each other for their opinion.
