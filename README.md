# Substreams Playground
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

This repository holds a few example _Substreams Modules_, and example _consuming clients_.

## Documentation

https://github.com/streamingfast/substreams under _Documentation_.

## Example Substreams Modules

* [PancakeSwap Substreams](./modules/pancakeswap) - Our most complete example to date. Tracking PancakeSwap on BSC Mainnet.
* [ETH Token Substreams](./modules/erc20) - Substreams tracking ERC-20 tokens. For ETH Mainnet.
* [Solana SPL Tokens](./modules/sol-spl-tokens) - First draft at solana SPL tokens extraction
* [Uniswap](./modules/uniswap) - First draft at tracking Uniswap on ETH Mainnet


## Example Consuming Clients

* In [Go](./consumers/golang)
* In [Rust](./consumers/rust)
* In [Python](./consumers/python)
* An [E2E indexer for PancakeSwap](./consumers/pancakeswap-to-graphnode) in Go.


## Contributing

**Please first refer to the general
[StreamingFast contribution guide](https://github.com/streamingfast/streamingfast/blob/master/CONTRIBUTING.md)**,
if you wish to contribute to this code base.


## License

[Apache 2.0](LICENSE)
