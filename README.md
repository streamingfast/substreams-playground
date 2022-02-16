# Substream-based PancakeSwap
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/substream-pancakeswap)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This repo holds the `exchange` substream-based "pseudo-subgraph" from PancakeSwap.


## Usage

Copy some blocks locally to speed things up:

```
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/0006809* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000681* ./localblocks/
```

Run with:

```bash

go run -v ./cmd/substream-exchange
```


## Current layout

```mermaid

graph TD;
  PE["PairExtractor(Contract)"]
  PSB[PCSPairStateBuilder]
  TPSB[PCSTotalPairsStateBuilder] 
  RE[ReservesExtractor]
  B[Raw Chain Block]
  PRSB[PCSPricesStateBuilder]
  END[Consumable stream by user]
  SWAP[SwapsExtractor]
  VOL24[Volume24hStateBuilder]
  
  B -- ETH Block --> PE
  PE -- "[]PCSPair" --> PSB
  PE -- "[]PCSPair" --> TPSB 
  TPSB -- Total Pairs Store --> END
  PSB -- "Pairs Store" --> RE
  B -- ETH Block --> RE
  RE -- Reserves Updates --> PRSB
  PRSB -- Prices Store --> END
  B -- ETH Block --> SWAP
  SWAP -- "[]PCSSwap" --> VOL24
```


## Contributing

**Issues and PR in this repo related strictly to Pancake Generated.**

Report any protocol-specific issues in their
[respective repositories](https://github.com/streamingfast/streamingfast#protocols)

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.

## License

[Apache 2.0](LICENSE)
