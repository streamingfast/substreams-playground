use hex;
use substreams::{Hex};
use substreams_ethereum::pb::eth;

use crate::{address_decode, address_pretty, Token};
use crate::eth::{read_string, read_uint32};

pub fn create_rpc_calls(addr: &Vec<u8>) -> eth::rpc::RpcCalls {
    let decimals = hex::decode("313ce567").unwrap();
    let name = hex::decode("06fdde03").unwrap();
    let symbol = hex::decode("95d89b41").unwrap();

    return eth::rpc::RpcCalls {
        calls: vec![
            eth::rpc::RpcCall {
                to_addr: Vec::from(addr.clone()),
                method_signature: decimals,
            },
            eth::rpc::RpcCall {
                to_addr: Vec::from(addr.clone()),
                method_signature: name,
            },
            eth::rpc::RpcCall {
                to_addr: Vec::from(addr.clone()),
                method_signature: symbol,
            },
        ],
    };
}

pub fn retry_rpc_calls(pair_token_address: &String) -> Result<Token, String> {
    let rpc_calls = create_rpc_calls(&address_decode(pair_token_address));

    let rpc_responses_unmarshalled: eth::rpc::RpcResponses =
	substreams_ethereum::rpc::eth_call(&rpc_calls);

    if rpc_responses_unmarshalled.responses[0].failed
        || rpc_responses_unmarshalled.responses[1].failed
        || rpc_responses_unmarshalled.responses[2].failed
    {
        return Err(format!("not a ERC20 because of a failure: {}", address_pretty(pair_token_address.as_bytes())));
    };

    let decoded_decimals = read_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
    if decoded_decimals.is_err() {
        return Err(format!("{} is not a an ERC20 token contract decimal `eth_call` failed: {}", Hex(&pair_token_address), decoded_decimals.err().unwrap()));
    }


    let decoded_name = read_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
    if decoded_name.is_err() {
        return Err(format!("{} is not a an ERC20 token contract name `eth_call` failed: {}", Hex(&pair_token_address),decoded_name.err().unwrap()));
    }


    let decoded_symbol = read_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());
    if decoded_symbol.is_err() {
        return Err(format!("{} is not a an ERC20 token contract symbol `eth_call` failed: {}", Hex(&pair_token_address),decoded_symbol.err().unwrap()));
    }

    return Ok(Token {
        address: pair_token_address.to_string(),
        name: decoded_name.unwrap(),
        symbol: decoded_symbol.unwrap(),
        decimals: decoded_decimals.unwrap() as u64,
    })
}
