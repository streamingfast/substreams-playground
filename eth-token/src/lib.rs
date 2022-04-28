mod eth;
mod rpc;
mod pb;

use eth::{address_pretty, decode_string, decode_uint32};
use substreams::{log, proto, state};

#[no_mangle]
pub extern "C" fn block_to_tokens(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let mut tokens = pb::tokens::Tokens { tokens: vec![] };
    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let initialize_method_hash: &str = "0x1459457a";

    for trx in blk.transaction_traces {
        for call in trx.calls {
            if call.call_type == pb::eth::CallType::Create as i32
                || call.call_type == pb::eth::CallType::Call as i32 // proxy contract creation
                && !call.state_reverted
            {
                let call_input_len = call.input.len();

                if call.call_type == pb::eth::CallType::Call as i32
                    && (call_input_len < 4
                    || !address_pretty(&call.input).starts_with(initialize_method_hash))
                {
                    continue;
                }

                let contract_address = address_pretty(&call.address);
                let caller_address = address_pretty(&call.caller);

                //pancake v1 and v2
                if caller_address == "0xca143ce32fe78f1f7019d7d551a6402fc5350c73"
                    || caller_address == "0xbcfccbde45ce874adcb698cc183debcf17952812"
                {
                    continue;
                }

                if call.call_type == pb::eth::CallType::Create as i32 {
                    let mut code_change_len = 0;
                    for code_change in &call.code_changes {
                        code_change_len += code_change.new_code.len()
                    }

                    log::println(format!(
                        "found contract creation: {}, caller {}, code change {}, input {}",
                        contract_address,
                        caller_address,
                        code_change_len,
                        call_input_len,
                    ));

                    if code_change_len <= 150 {
                        // optimization to skip none viable SC
                        log::println(format!(
                            "skipping too small code to be a token contract: {}",
                            address_pretty(&call.address)
                        ));
                        continue;
                    }
                } else if call.call_type == pb::eth::CallType::Call as i32 {
                    log::println(format!(
                        "found contract that may be a proxy contract: {}",
                        caller_address
                    ))
                }

                if caller_address == "0x0000000000004946c0e9f43f4dee607b0ef1fa1c"
                    || caller_address == "0x00000000687f5b66638856396bee28c1db0178d1"
                {
                    continue;
                }

                let rpc_calls = rpc::create_rpc_calls(&call.address);

                let rpc_responses_marshalled: Vec<u8> =
                    substreams::rpc::eth_call(substreams::proto::encode(&rpc_calls).unwrap());
                let rpc_responses_unmarshalled: substreams::pb::eth::RpcResponses =
                    substreams::proto::decode(&rpc_responses_marshalled).unwrap();

                if rpc_responses_unmarshalled.responses[0].failed
                    || rpc_responses_unmarshalled.responses[1].failed
                    || rpc_responses_unmarshalled.responses[2].failed
                {
                    log::println(format!(
                        "not a token because of a failure: {}",
                        address_pretty(&call.address)
                    ));
                    continue;
                };

                if !(rpc_responses_unmarshalled.responses[1].raw.len() >= 96)
                    || rpc_responses_unmarshalled.responses[0].raw.len() != 32
                    || !(rpc_responses_unmarshalled.responses[2].raw.len() >= 96)
                {
                    log::println(format!(
                        "not a token because response length: {}",
                        address_pretty(&call.address)
                    ));
                    continue;
                };

                log::println(format!(
                    "found a token: {} {}",
                    address_pretty(&call.address),
                    decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref()),
                ));

                let decoded_address = address_pretty(&call.address);
                let decoded_decimals =
                    decode_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
                let decoded_name =
                    decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
                let decoded_symbol =
                    decode_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());

                let token = pb::tokens::Token {
                    address: decoded_address,
                    name: decoded_name,
                    symbol: decoded_symbol,
                    decimals: decoded_decimals as u64,
                };

                tokens.tokens.push(token);
            }
        }
    }

    substreams::output(tokens);
}

#[no_mangle]
pub extern "C" fn build_tokens_state(tokens_ptr: *mut u8, tokens_len: usize) {
    substreams::register_panic_hook();

    let tokens: pb::tokens::Tokens = proto::decode_ptr(tokens_ptr, tokens_len).unwrap();

    for token in tokens.tokens {
        let key = format!("token:{}", token.address);
        state::set(1, key, &proto::encode(&token).unwrap());
    }
}
