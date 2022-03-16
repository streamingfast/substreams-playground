use hex;

use crate::{address_pretty, decode_string, decode_uint32, pb};

pub fn create_rpc_calls(addr: Vec<u8>) -> substreams::pb::eth::RpcCalls {
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

pub fn get_token(addr: Vec<u8>) -> Option<pb::pcs::Erc20Token> {
    let rpc_calls = create_rpc_calls(addr.clone());

    let rpc_responses_marshalled: Vec<u8> =
        substreams::rpc::eth_call(substreams::proto::encode(&rpc_calls).unwrap());
    let rpc_responses_unmarshalled: substreams::pb::eth::RpcResponses =
        substreams::proto::decode(rpc_responses_marshalled).unwrap();

    if rpc_responses_unmarshalled.responses[0].failed
        || rpc_responses_unmarshalled.responses[1].failed
        || rpc_responses_unmarshalled.responses[2].failed {
        return None;
    };

    if !(rpc_responses_unmarshalled.responses[1].raw.len() >= 96)
        || rpc_responses_unmarshalled.responses[0].raw.len() != 32
        || !(rpc_responses_unmarshalled.responses[2].raw.len() >= 96) {
        return None;
    };

    let decoded_address = address_pretty(addr.as_slice());
    let decoded_decimals = decode_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
    let decoded_name = decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
    let decoded_symbol = decode_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());

    return Some(pb::pcs::Erc20Token {
        address: decoded_address.clone(),
        name: decoded_name,
        symbol: decoded_symbol,
        decimals: decoded_decimals as u64,
    });
}
