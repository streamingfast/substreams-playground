use hex;
use substreams::{log_debug};
use substreams_ethereum::pb::eth;

use crate::{address_decode, address_pretty, decode_string, decode_uint32, Token};

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

pub fn retry_rpc_calls(pair_token_address: &String) -> Option<Token> {
    let rpc_calls = create_rpc_calls(&address_decode(pair_token_address));

    let rpc_responses_unmarshalled: eth::rpc::RpcResponses =
	substreams_ethereum::rpc::eth_call(&rpc_calls);

    if rpc_responses_unmarshalled.responses[0].failed
        || rpc_responses_unmarshalled.responses[1].failed
        || rpc_responses_unmarshalled.responses[2].failed
    {
        log_debug!("not a token because of a failure: {}", address_pretty(pair_token_address.as_bytes()));
        return None
    };

    if !(rpc_responses_unmarshalled.responses[1].raw.len() >= 96)
        || rpc_responses_unmarshalled.responses[0].raw.len() != 32
        || !(rpc_responses_unmarshalled.responses[2].raw.len() >= 96)
    {
        log_debug!("not a token because response length: {}", address_pretty(pair_token_address.as_bytes()));
        return None
    };

    let decoded_decimals = decode_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
    let decoded_name = decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
    let decoded_symbol = decode_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());

    return Some(Token {
        address: pair_token_address.to_string(),
        name: decoded_name,
        symbol: decoded_symbol,
        decimals: decoded_decimals as u64,
    })
}
