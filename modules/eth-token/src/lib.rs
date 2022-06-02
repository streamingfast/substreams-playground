mod eth;
mod pb;
mod rpc;

use substreams_ethereum::pb::eth as ethpb;
use substreams::{log, proto, store, Hex, hex};
use substreams::errors::Error;

const INITIALIZE_METHOD_HASH: [u8;4] = hex!("1459457a");

#[substreams::handlers::map]
fn map_tokens(blk: ethpb::v1::Block) -> Result<pb::tokens::Tokens, Error> {
    let mut tokens = vec![];

    for trx in blk.transaction_traces {
        for call in trx.calls {
            if call.state_reverted {
                continue
            }
            if call.call_type == ethpb::v1::CallType::Create as i32 ||
                call.call_type == ethpb::v1::CallType::Call as i32 // proxy contract creation
            {
                let call_input_len = call.input.len();
                if call.call_type == ethpb::v1::CallType::Call as i32
                    && (call_input_len < 4 || call.input[0..4] != INITIALIZE_METHOD_HASH) {
                    // this will check if a proxy contract has been called to create a ERC20 contract.
                    // if that is the case the Proxy contract will call the initialize function on the ERC20 contract
                    // this is part of the OpenZeppelin Proxy contract standard
                    continue;
                }

                if call.call_type == ethpb::v1::CallType::Create as i32 {
                    let mut code_change_len = 0;
                    for code_change in &call.code_changes {
                        code_change_len += code_change.new_code.len()
                    }

                    log::debug!(
                        "found contract creation: {}, caller {}, code change {}, input {}",
                        Hex(&call.address),
                        Hex(&call.caller),
                        code_change_len,
                        call_input_len,
                    );

                    if code_change_len <= 150 {
                        // optimization to skip none viable SC
                        log::info!("skipping too small code to be a token contract: {}",Hex(&call.address));
                        continue;
                    }
                } else {
                    log::debug!("found proxy initialization: contract {}, caller {}",Hex(&call.address),Hex(&call.caller));
                }

                if call.caller == hex!("0000000000004946c0e9f43f4dee607b0ef1fa1c") ||
                    call.caller == hex!("00000000687f5b66638856396bee28c1db0178d1") {
                    log::debug!("skipping known caller address");
                    continue;
                }

                let rpc_calls = rpc::create_rpc_calls(&call.address);
                let rpc_responses_unmarshalled: ethpb::rpc::RpcResponses = substreams_ethereum::rpc::eth_call(&rpc_calls);
                let responses = rpc_responses_unmarshalled.responses;

                if responses[0].failed || responses[1].failed || responses[2].failed {
                    let decimals_error = String::from_utf8_lossy(responses[0].raw.as_ref());
                    let name_error = String::from_utf8_lossy(responses[1].raw.as_ref());
                    let symbol_error = String::from_utf8_lossy(responses[2].raw.as_ref());

                    log::debug!(
                        "{} is not a an ERC20 token contract because of 'eth_call' failures [decimals: {}, name: {}, symbol: {}]",
                        Hex(&call.address),
                        decimals_error,
                        name_error,
                        symbol_error,
                    );
                    continue;
                };

                let decoded_decimals = eth::read_uint32(responses[0].raw.as_ref());
                if decoded_decimals.is_err() {
                    log::debug!(
                        "{} is not a an ERC20 token contract decimal `eth_call` failed: {}",
                        Hex(&call.address),
                        decoded_decimals.err().unwrap(),
                    );
                    continue;
                }

                let decoded_name = eth::read_string(responses[1].raw.as_ref());
                if decoded_name.is_err() {
                    log::debug!(
                        "{} is not a an ERC20 token contract name `eth_call` failed: {}",
                        Hex(&call.address),
                        decoded_name.err().unwrap(),
                    );
                    continue;
                }

                let decoded_symbol = eth::read_string(responses[2].raw.as_ref());
                if decoded_symbol.is_err() {
                    log::debug!(
                        "{} is not a an ERC20 token contract symbol `eth_call` failed: {}",
                        Hex(&call.address),
                        decoded_symbol.err().unwrap(),
                    );
                    continue;
                }

                let decimals = decoded_decimals.unwrap() as u64;
                let symbol = decoded_symbol.unwrap();
                let name = decoded_name.unwrap();
                log::debug!(
                    "{} is an ERC20 token contract with name {}",
                    Hex(&call.address),
                    name,
                );
                log::debug!("out");
                let token = pb::tokens::Token {
                    address: Hex(&call.address).to_string(),
                    name,
                    symbol,
                    decimals,
                };

                tokens.push(token);
            }
        }
    }

    Ok(pb::tokens::Tokens { tokens })
}

#[substreams::handlers::store]
fn store_tokens(tokens: pb::tokens::Tokens, store: store::StoreSet) {
    for token in tokens.tokens {
        let key = format!("token:{}", token.address);
        store.set(1, key, &proto::encode(&token).unwrap());
    }
}
