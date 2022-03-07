# Substream-based PancakeSwap
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/substream-pancakeswap)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This repo holds the `exchange` substream-based "pseudo-subgraph" from PancakeSwap.

## Build wasm
```bash
go generate ./...
```

## Usage

Copy some blocks locally to speed things up:

```
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/0006809* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000681* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000682* ./localblocks/
gsutil -m cp gs://dfuseio-global-blocks-us/eth-bsc-mainnet/v1/000683* ./localblocks/
```

Compile:

```bash
go install -v ./cmd/substreams-pcs
```

Run the native version:

```bash
substreams-pcs native_substreams_manifest.yaml pairs 300
substreams-pcs native_substreams_manifest.yaml pairs 10000 -s 6811000
substreams-pcs native_substreams_manifest.yaml pairs 10000 -s 6821000
substreams-pcs native_substreams_manifest.yaml pairs 2000 -s 6831000
```

Run the WASM version:

```bash
substreams-pcs wasm_substreams_manifest.yaml pairs 300
substreams-pcs wasm_substreams_manifest.yaml pairs 10000 -s 6811000
substreams-pcs wasm_substreams_manifest.yaml pairs 10000 -s 6821000
substreams-pcs wasm_substreams_manifest.yaml pairs 2000 -s 6831000
```


## Current layout

For `native_substreams_manifest.yaml`:

```mermaid
graph TD;
  sf.ethereum.types.v1.Block -- "proto:sf.ethereum.types.v1.Block" --> pair_extractor
  pair_extractor -- "stream:pair_extractor" --> pairs
  sf.ethereum.types.v1.Block -- "proto:sf.ethereum.types.v1.Block" --> reserves_extractor
  pairs -- "store:pairs" --> reserves_extractor
  reserves_extractor -- "stream:reserves_extractor" --> reserves
  pairs -- "store:pairs" --> reserves
  reserves_extractor -- "stream:reserves_extractor" --> prices
  pairs -- "store:pairs" --> prices
  reserves -- "store:reserves" --> prices
  sf.ethereum.types.v1.Block -- "proto:sf.ethereum.types.v1.Block" --> mint_burn_swaps_extractor
  pairs -- "store:pairs" --> mint_burn_swaps_extractor
  prices -- "store:prices" --> mint_burn_swaps_extractor
  pair_extractor -- "stream:pair_extractor" --> totals
  mint_burn_swaps_extractor -- "stream:mint_burn_swaps_extractor" --> totals
  sf.ethereum.types.v1.Block -- "proto:sf.ethereum.types.v1.Block" --> volumes
  mint_burn_swaps_extractor -- "stream:mint_burn_swaps_extractor" --> volumes
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
