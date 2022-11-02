Solana SPL Token Transfers Substreams
=====================================

* Install [`substreams` and its dependencies here](https://substreams.streamingfast.io/getting-started/installing-the-cli)
* To use streamingfast endpoints, check [`authentication`](https://substreams.streamingfast.io/reference-and-specs/authentication)

# Running from precompiled package

* Asking the server to backprocess from the initialBlock in the manifest (31,313,760) to our startBlock (31,330,000), send the snapshot from there (`-i` flag) and streaming the data from the next 50 blocks.

```
substreams run -e mainnet.sol.streamingfast.io:443 https://github.com/streamingfast/substreams-playground/releases/download/v0.5.4/eth-token-at-pcs-v0.5.4.spkg store_mint_native_volumes -s 31330000 -t +50 -i
```

# Running from source

## Building
```
cargo build --release
```

## Protobuf Generation
```
substreams protogen substreams.yaml --exclude-paths="sf/solana,sf/substreams,google"
```

## Running the Substrams

* Asking the server to backprocess from the initialBlock in the manifest (31,313,760) to our startBlock (31,330,000), send the snapshot from there (`-i` flag) and streaming the data from the next 50 blocks.
```
substreams run -e mainnet.sol.streamingfast.io:443 substreams.yaml store_mint_native_volumes -s 31330000 -t +50 -i
```
