use hex;
use substreams::{log, pb};

use crate::pcs::Pair;
use crate::{address_decode, address_pretty, decode_string, decode_uint32, Token};

pub fn create_rpc_calls(addr: &Vec<u8>) -> substreams::pb::eth::RpcCalls {
    let decimals = hex::decode("313ce567").unwrap();
    let name = hex::decode("06fdde03").unwrap();
    let symbol = hex::decode("95d89b41").unwrap();

    return substreams::pb::eth::RpcCalls {
        calls: vec![
            substreams::pb::eth::RpcCall {
                to_addr: Vec::from(addr.clone()),
                method_signature: decimals,
            },
            substreams::pb::eth::RpcCall {
                to_addr: Vec::from(addr.clone()),
                method_signature: name,
            },
            substreams::pb::eth::RpcCall {
                to_addr: Vec::from(addr.clone()),
                method_signature: symbol,
            },
        ],
    };
}

pub fn retry_rpc_calls(pair_token_address: &String) -> Token {
    let rpc_calls = create_rpc_calls(&address_decode(pair_token_address));

    let rpc_responses_marshalled: Vec<u8> =
        substreams::rpc::eth_call(substreams::proto::encode(&rpc_calls).unwrap());
    let rpc_responses_unmarshalled: substreams::pb::eth::RpcResponses =
        substreams::proto::decode(&rpc_responses_marshalled).unwrap();

    if rpc_responses_unmarshalled.responses[0].failed
        || rpc_responses_unmarshalled.responses[1].failed
        || rpc_responses_unmarshalled.responses[2].failed
    {
        panic!(
            "not a token because of a failure: {}",
            address_pretty(pair_token_address.as_bytes())
        )
    };

    if !(rpc_responses_unmarshalled.responses[1].raw.len() >= 96)
        || rpc_responses_unmarshalled.responses[0].raw.len() != 32
        || !(rpc_responses_unmarshalled.responses[2].raw.len() >= 96)
    {
        panic!(
            "not a token because response length: {}",
            address_pretty(pair_token_address.as_bytes())
        )
    };

    let decoded_address = address_pretty(&pair_token_address.as_bytes());
    let decoded_decimals = decode_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
    let decoded_name = decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
    let decoded_symbol = decode_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());

    Token {
        address: decoded_address,
        name: decoded_name,
        symbol: decoded_symbol,
        decimals: decoded_decimals as u64,
    }
}
