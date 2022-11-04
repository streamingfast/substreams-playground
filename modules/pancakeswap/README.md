PancakeSwap Substreams
======================

Install [`substreams` and its dependencies here](https://substreams.streamingfast.io/getting-started/installing-the-cli), compile these modules with:

```
# We assume you are at root of project
cd modules/pancakeswap
cargo build --target=wasm32-unknown-unknown --release
```

Run with:

```
substreams run -e bsc.streamingfast.io:443 substreams.yaml store_pairs,map_pairs,db_out,store_volumes,store_totals -s 6810706 -t 6810711
```

> Right now `bsc.streamingfast.io` endpoint is not running Substreams service for a temporary period, the command below will not work, please visit https://substreams.streamingfast.io/getting-started to look for other Substreams to run to test. If you are in dire needs for BNB Substreams support, drop a message in our [StreamingFast Discord](https://discord.gg/jZwqxJAvRs)  

## Visual data flow

This is a flow that is executed for each block.  The graph is produced with `substreams graph ./substreams.yaml`.

```mermaid

graph TD;
  map_pairs[map: map_pairs]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> map_pairs
  store_pcs_tokens[store: store_pcs_tokens]
  map_pairs --> store_pcs_tokens
  ethtokens_at_pcs:store_tokens --> store_pcs_tokens
  store_pairs[store: store_pairs]
  map_pairs --> store_pairs
  map_reserves[map: map_reserves]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> map_reserves
  store_pairs --> map_reserves
  store_pcs_tokens --> map_reserves
  store_reserves[store: store_reserves]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> store_reserves
  map_reserves --> store_reserves
  store_pairs --> store_reserves
  store_prices[store: store_prices]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> store_prices
  map_reserves --> store_prices
  store_pairs --> store_prices
  store_reserves --> store_prices
  map_burn_swaps_events[map: map_burn_swaps_events]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> map_burn_swaps_events
  store_pairs --> map_burn_swaps_events
  store_reserves --> map_burn_swaps_events
  store_pcs_tokens --> map_burn_swaps_events
  store_totals[store: store_totals]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> store_totals
  map_pairs --> store_totals
  map_burn_swaps_events --> store_totals
  store_volumes[store: store_volumes]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> store_volumes
  map_burn_swaps_events --> store_volumes
  db_out[map: db_out]
  sf.substreams.v1.Clock[source: sf.substreams.v1.Clock] --> db_out
  store_pcs_tokens -- deltas --> db_out
  store_pairs -- deltas --> db_out
  store_totals -- deltas --> db_out
  store_volumes -- deltas --> db_out
  store_reserves -- deltas --> db_out
  map_burn_swaps_events --> db_out
  store_pcs_tokens --> db_out
  ethtokens_at_pcs:map_tokens[map: ethtokens_at_pcs:map_tokens]
  sf.ethereum.type.v1.Block[source: sf.ethereum.type.v1.Block] --> ethtokens_at_pcs:map_tokens
  ethtokens_at_pcs:store_tokens[store: ethtokens_at_pcs:store_tokens]
  ethtokens_at_pcs:map_tokens --> ethtokens_at_pcs:store_tokens
```
