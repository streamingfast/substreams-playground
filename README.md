# Substream-based PancakeSwap
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/substream-pancakeswap)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This repo holds the `exchange` substream-based "pseudo-subgraph" from PancakeSwap.


## Usage

Copy some blocks locally to speed things up:

```
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/0006809* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000681* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000682* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000683* ./localblocks/
```

Run with:

```bash

go run -v ./cmd/substream-exchange substreams_manifest.yaml pairs 300 | tee /tmp/sub
go run -v ./cmd/substream-exchange substreams_manifest.yaml pairs 10000 -s 6811000 | tee /tmp/sub
go run -v ./cmd/substream-exchange substreams_manifest.yaml pairs 10000 -s 6821000 | tee /tmp/sub
go run -v ./cmd/substream-exchange substreams_manifest.yaml pairs 2000 -s 6831000 | tee /tmp/sub
```


## Current layout

```mermaid

graph TD;
  PE["PairExtractor(Contract)"]
  PAIRS[PCSPairStateBuilder]
  TOTAL[PCSTotalPairsStateBuilder]
  RE[ReservesExtractor]
  B[Raw Chain Block]
  PRICES[PCSPricesStateBuilder]
  EVENTS[MintBurnSwapsExtractor]
  VOL24[Volume24hStateBuilder]
  HUB[Subscription hub]

  B -- ETH Block --> PE
  PE -- "[]PCSPair" --> PAIRS
  PE -- "[]PCSPair" --> TOTAL
  TOTAL -- Total Pairs Store --> HUB
  PAIRS -- "Pairs Store" --> RE
  B -- ETH Block --> RE
  RE -- Reserves Updates --> PRICES
  PRICES -- Prices Store --> HUB
  PRICES -- Prices Store --> EVENTS
  PAIRS -- Pairs Store --> EVENTS
  B -- ETH Block --> EVENTS
  EVENTS -- "[]Swap,[]Mint,[]Burn" --> VOL24
  EVENTS -- "[]Swap,[]Mint,[]Burn" --> TOTAL
  VOL24 -- "Volume Store" --> HUB
```

## References

Debezium format example: https://nightlies.apache.org/flink/flink-docs-master/docs/connectors/table/formats/debezium/#how-to-use-debezium-format
Fluvio Smart Modules overview: https://www.fluvio.io/docs/smartmodules/overview/



## Contributing

**Issues and PR in this repo related strictly to Pancake Generated.**

Report any protocol-specific issues in their
[respective repositories](https://github.com/streamingfast/streamingfast#protocols)

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.

## License

[Apache 2.0](LICENSE)
