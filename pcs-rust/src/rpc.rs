use hex;

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
