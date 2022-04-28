PancakeSwap Substreams
======================

Install [`substreams` and its dependencies here](../README.md), compile these modules with:

```
./build.sh
```

and try with:

```
substreams run -e bsc-dev.streamingfast.io:443 substreams.yaml pairs,block_to_pairs,db_out,volumes,totals -s 6810706 -t 6810711
```

## Visual data flow

This is a flow that is executed for each block.  The graph is produced automatically from the `.yaml` manifest.

```mermaid
graph TD;
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> block_to_pairs
  block_to_pairs -- "map:block_to_pairs" --> pairs
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> block_to_reserves
  pairs -- "store:pairs:get" --> block_to_reserves
  tokens -- "store:tokens:get" --> block_to_reserves
  block_to_reserves -- "map:block_to_reserves" --> reserves
  pairs -- "store:pairs:get" --> reserves
  block_to_reserves -- "map:block_to_reserves" --> prices
  pairs -- "store:pairs:get" --> prices
  reserves -- "store:reserves:get" --> prices
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> mint_burn_swaps_extractor
  pairs -- "store:pairs:get" --> mint_burn_swaps_extractor
  prices -- "store:prices:get" --> mint_burn_swaps_extractor
  tokens -- "store:tokens:get" --> mint_burn_swaps_extractor
  block_to_pairs -- "map:block_to_pairs" --> totals
  mint_burn_swaps_extractor -- "map:mint_burn_swaps_extractor" --> totals
  sf.ethereum.type.v1.block -- "source:sf.ethereum.type.v1.block" --> volumes
  mint_burn_swaps_extractor -- "map:mint_burn_swaps_extractor" --> volumes
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> block_to_tokens
  block_to_tokens -- "map:block_to_tokens" --> tokens
  sf.ethereum.type.v1.Block -- "source:sf.ethereum.type.v1.Block" --> db_out
  tokens -- "store:tokens:deltas" --> db_out
  pairs -- "store:pairs:deltas" --> db_out
  totals -- "store:totals:deltas" --> db_out
  volumes -- "store:volumes:deltas" --> db_out
  reserves_extractor -- "map:reserves_extractor" --> db_out
  mint_burn_swaps_extractor -- "map:mint_burn_swaps_extractor" --> db_out
  tokens -- "store:tokens:get" --> db_out
```
