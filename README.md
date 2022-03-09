# Substream-based PancakeSwap
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/streamingfast/substream-pancakeswap)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This repo holds the `exchange` substream-based "pseudo-subgraph" from PancakeSwap.

## Build and install wasm-pack
```bash
git clone https://github.com/rustwasm/wasm-pack.git $somedir
cd $somedir && cargo build --release
export PATH=$PATH:$somedir/target/release
```

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
go install -v ./cmd/sseth
```

Run the native version:

```bash
sseth native_substreams_manifest.yaml pairs 300
sseth native_substreams_manifest.yaml pairs 10000 -s 6811000
sseth native_substreams_manifest.yaml pairs 10000 -s 6821000
sseth native_substreams_manifest.yaml pairs 2000 -s 6831000
```

Run the WASM version:

```bash
sseth wasm_substreams_manifest.yaml pairs 300
sseth wasm_substreams_manifest.yaml pairs 10000 -s 6811000
sseth wasm_substreams_manifest.yaml pairs 10000 -s 6821000
sseth wasm_substreams_manifest.yaml pairs 2000 -s 6831000
```


## Current layout

For `native_substreams_manifest.yaml`:

```mermaid
graph TD;
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> block_to_pairs
  block_to_pairs -- "map:block_to_pairs" --> pairs
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> block_to_reserves
  pairs -- "store:pairs:get" --> block_to_reserves
  block_to_reserves -- "map:block_to_reserves" --> reserves
  pairs -- "store:pairs:get" --> reserves
  block_to_reserves -- "map:block_to_reserves" --> prices
  pairs -- "store:pairs:get" --> prices
  reserves -- "store:reserves:get" --> prices
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> mint_burn_swaps_extractor
  pairs -- "store:pairs:get" --> mint_burn_swaps_extractor
  prices -- "store:prices:get" --> mint_burn_swaps_extractor
  block_to_pairs -- "map:block_to_pairs" --> totals
  mint_burn_swaps_extractor -- "map:mint_burn_swaps_extractor" --> totals
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> volumes
  mint_burn_swaps_extractor -- "map:mint_burn_swaps_extractor" --> volumes
  volumes -- "store:volumes:get" --> database_output
  volumes -- "store:volumes:deltas" --> database_output
  mint_burn_swaps_extractor -- "map:mint_burn_swaps_extractor" --> database_output
```

For `wasm_substreams_manifest.yaml`:

```mermaid
graph TD;
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> pair_extractor
  pair_extractor -- "map:pair_extractor" --> pairs
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> reserves_extractor
  pairs -- "store:pairs:get" --> reserves_extractor
  reserves_extractor -- "map:reserves_extractor" --> db_out
  pairs -- "store:pairs:deltas" --> db_out
  pairs -- "store:pairs:get" --> db_out
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
