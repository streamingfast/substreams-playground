PancakeSwap Substreams
======================

Install [`substreams` and its dependencies here](https://github.com/streamingfast/substreams), compile these modules with:

```
cd ../pancakeswap
cargo build --target wasm32-unknown-unknown --release
```

At the beginning of you manifest `substreams.yaml` file you can add some import statements at the [top](https://substreams.streamingfast.io/developer-guide/creating-your-manifest). If you make some changes to an imported module and you want to
test the changes, you will have to pack the changes in a `.spkg` file. Simply run:

```bash
cd ../eth-token
substreams pack ./substreams.yaml
```

> Also don't forget to change the url to point to the local .spkg file location

and try with:

```
substreams run -e bsc-dev.streamingfast.io:443 substreams.yaml store_pairs,map_pairs,db_out,store_volumes,store_totals -s 6810706 -t 6810711
```

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
