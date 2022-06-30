use hex;
use substreams_ethereum::pb::eth;

pub const DECIMALS: &str = "313ce567";
pub const NAME: &str = "06fdde03";
pub const SYMBOL: &str = "95d89b41";

pub fn create_rpc_calls(addr: &Vec<u8>, method_signatures: Vec<&str>) -> eth::rpc::RpcCalls {
    let mut rpc_calls = eth::rpc::RpcCalls { calls: vec![] };

    for method_signature in method_signatures {
        rpc_calls.calls.push(eth::rpc::RpcCall {
            to_addr: Vec::from(addr.clone()),
            method_signature: hex::decode(method_signature).unwrap(),
        })
    }

    return  rpc_calls
}
