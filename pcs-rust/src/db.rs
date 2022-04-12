use std::process::exit;

use bigdecimal::BigDecimal;
use substreams::pb::substreams::{
    store_delta, table_change::Operation, DatabaseChanges, Field, StoreDelta, StoreDeltas,
    TableChange,
};
use substreams::{log, proto};

use crate::pb::eth::Block;
use crate::pcs::{Burn, Event, Events, Mint, Reserve, Reserves, Swap};
use crate::{field, field_create_string, pb, pcs, proto_decode_to_string, utils, Type};

const PANCAKE_FACTORY: &str = "0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73";

#[derive(Clone)]
enum Item {
    PairDelta(StoreDelta),
    TokenDelta(StoreDelta),
    TotalDelta(StoreDelta),
    VolumeDelta(StoreDelta),
    Reserve(Reserve),
    Event(Event),
}

pub fn process(
    block: &Block,
    pair_deltas: StoreDeltas,
    token_deltas: StoreDeltas,
    total_deltas: StoreDeltas,
    volumes_deltas: StoreDeltas,
    reserves: Reserves,
    events: Events,
    tokens_idx: u32,
) -> DatabaseChanges {
    let items = join_sort_deltas(
        pair_deltas,
        token_deltas,
        total_deltas,
        volumes_deltas,
        reserves,
        events,
    );

    let mut database_changes: DatabaseChanges = DatabaseChanges {
        table_changes: vec![],
    };

    for item in items {
        match item {
            Item::PairDelta(delta) => {
                handle_pair_delta(delta, &block, &mut database_changes, tokens_idx)
            }
            Item::TokenDelta(delta) => handle_token_delta(delta, &mut database_changes, block),
            Item::TotalDelta(delta) => handle_total_delta(delta, &mut database_changes, block),
            Item::VolumeDelta(delta) => handle_volume_delta(delta, &mut database_changes, block),
            Item::Reserve(reserve) => handle_reserves(reserve, &mut database_changes),
            Item::Event(event) => handle_events(event, &mut database_changes, block),
        }
    }

    return database_changes;
}

fn handle_pair_delta(
    delta: StoreDelta,
    block: &Block,
    changes: &mut DatabaseChanges,
    tokens_idx: u32,
) {
    if delta.operation != store_delta::Operation::Create as i32 {
        return;
    }

    let pair: pcs::Pair = proto::decode(delta.new_value).unwrap();

    let token0 = utils::get_last_token(tokens_idx, pair.token0_address.as_str());
    let token1 = utils::get_last_token(tokens_idx, pair.token1_address.as_str());

    changes.table_changes.push(TableChange {
        table: "pair".to_string(),
        pk: pair.address.clone(),
        block_num: block.number,
        ordinal: delta.ordinal,
        operation: delta.operation,
        fields: vec![
            field!("id", pair.address.clone(), ""),
            field!("name", format!("{}-{}", token0.symbol, token1.symbol), ""),
            field!("block", block.number, ""),
            field!("timestamp", block.timestamp(), ""),
        ],
    });
}

fn handle_token_delta(delta: StoreDelta, changes: &mut DatabaseChanges, block: &Block) {
    if delta.operation != store_delta::Operation::Create as i32 {
        return;
    }

    let token: pb::tokens::Token = proto::decode(delta.new_value).unwrap();

    changes.table_changes.push(TableChange {
        table: "token".to_string(),
        pk: token.address.clone(),
        block_num: block.number,
        ordinal: delta.ordinal,
        operation: Operation::Create as i32,
        fields: vec![
            field!("id", token.address, ""),
            field!("name", token.name, ""),
            field!("symbol", token.symbol, ""),
            field!("decimals", token.decimals, ""),
        ],
    });
}

fn handle_total_delta(delta: StoreDelta, changes: &mut DatabaseChanges, block: &Block) {
    let parts: Vec<&str> = delta.key.split(":").collect();
    let table = parts[0];

    match table {
        "pair" => {
            let pk = parts[1];
            let key = parts[2];

            match key {
                "total_transactions" => changes.table_changes.push(TableChange {
                    table: "pair".to_string(),
                    pk: pk.to_string(),
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: delta.operation,
                    fields: vec![field!(
                        key,
                        String::from_utf8_lossy(delta.new_value.as_slice()),
                        String::from_utf8_lossy(delta.old_value.as_slice())
                    )],
                }),
                _ => {}
            }
        }
        "token" => {
            let pk = parts[1];
            let key = parts[2];

            match key {
                "total_transactions" => changes.table_changes.push(TableChange {
                    table: "token".to_string(),
                    pk: pk.to_string(),
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: delta.operation,
                    fields: vec![field!(
                        key,
                        String::from_utf8_lossy(delta.new_value.as_slice()),
                        String::from_utf8_lossy(delta.old_value.as_slice())
                    )],
                }),
                _ => {}
            }
        }
        "global" => changes.table_changes.push(TableChange {
            table: "pancake_factory".to_string(),
            pk: PANCAKE_FACTORY.to_string(),
            block_num: block.number,
            ordinal: delta.ordinal,
            operation: Operation::Update as i32,
            fields: vec![field!(
                "total_pairs",
                proto_decode_to_string!(delta.new_value, "0"),
                proto_decode_to_string!(delta.old_value, "0")
            )],
        }),
        "global_day" => changes.table_changes.push(TableChange {
            table: "pancake_day_data".to_string(),
            pk: PANCAKE_FACTORY.to_string(),
            block_num: block.number,
            ordinal: delta.ordinal,
            operation: Operation::Update as i32,
            fields: vec![field!(
                "total_transactions",
                String::from_utf8_lossy(delta.new_value.as_slice()),
                String::from_utf8_lossy(delta.old_value.as_slice())
            )],
        }),
        _ => {}
    }
}

fn handle_volume_delta(delta: StoreDelta, changes: &mut DatabaseChanges, block: &Block) {
    let parts: Vec<&str> = delta.key.split(":").collect();
    let table = parts[0];
    let mut field: Option<Field> = None;

    //todo: fix the proto_decode_to_string stuff, because we do not proto_encode the
    // fields, we only put them in base 10
    match table {
        "pair_day" => {
            if delta.operation == Operation::Delete as i32 {
                return; // todo: add for token and global where we ignore the delete operation
            }

            // todo
            let day = parts[1];
            let pair_address = parts[2];
            let key = parts[3];

            match key {
                "usd" => {}
                _ => {}
            }

            if field.is_some() {
                let pk = format!("{}-{}", pair_address, day);
                changes.table_changes.push(TableChange {
                    table: "pair_day_data".to_string(),
                    pk,
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: delta.operation,
                    fields: vec![field.unwrap()],
                })
            }
        }
        "pair_hour" => {
            // todo
            let hour = parts[1];
            let pair_address = parts[2];
            let key = parts[3];

            match key {
                "usd" => {
                    field = Some(field!("", "", ""));
                }
                _ => {}
            }

            if field.is_some() {
                let pk = format!("{}-{}", pair_address, hour);
                changes.table_changes.push(TableChange {
                    table: "pair_hour_data".to_string(),
                    pk,
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: Operation::Update as i32,
                    fields: vec![field.unwrap()],
                })
            }
        }
        "pair" => {
            let pair_address = parts[1];
            let field_name = parts[2];

            // changes.table_changes.push(TableChange {
            //     table: "pair".to_string(),
            //     pk: swap.log_address.to_string(),
            //     block_num: block.number,
            //     ordinal: event.log_ordinal,
            //     operation: Operation::Update as i32,
            //     fields: vec![
            //         field!("volume_token_0", swap.volume_token0, ""),
            //         field!("volume_token_1", swap.volume_token1, ""),
            //         field!("volume_usd", swap.volume_usd, ""),
            //     ],
            // });
            match field_name {
                "usd" => {
                    field = Some(field!(
                        "volume_usd",
                        String::from_utf8_lossy(delta.new_value.as_slice()),
                        String::from_utf8_lossy(delta.old_value.as_slice())
                    ));
                }
                "token0" => {
                    field = Some(field!(
                        "volume_token0",
                        String::from_utf8_lossy(delta.new_value.as_slice()),
                        String::from_utf8_lossy(delta.old_value.as_slice())
                    ));
                }
                "token1" => {
                    field = Some(field!(
                        "volume_token1",
                        String::from_utf8_lossy(delta.new_value.as_slice()),
                        String::from_utf8_lossy(delta.old_value.as_slice())
                    ));
                }
                // "total_transactions" => {
                //     let total_transactions_new: String =
                //         proto_decode_to_string!(delta.new_value, "0");
                //     let total_transactions_old: String =
                //         proto_decode_to_string!(delta.old_value, "0");
                //
                //     field = Some(field!(
                //         "total_transactions",
                //         total_transactions_new,
                //         total_transactions_old
                //     ));
                // }
                _ => {}
            }

            if field.is_some() {
                changes.table_changes.push(TableChange {
                    table: "pair".to_string(),
                    pk: pair_address.to_string(),
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: Operation::Update as i32,
                    fields: vec![field.unwrap()],
                })
            }
        }
        "token" => {
            let token_address = parts[1];
            let key = parts[2];

            match key {
                "trade" => {
                    let trade_volume_new: String = proto_decode_to_string!(delta.new_value, "0.0");
                    let trade_volume_old: String = proto_decode_to_string!(delta.old_value, "0.0");
                    field = Some(field!("trade_volume", trade_volume_new, trade_volume_old));
                }
                "trade_usd" => {
                    let trade_volume_usd_new: String =
                        proto_decode_to_string!(delta.new_value, "0.0");
                    let trade_volume_usd_old: String =
                        proto_decode_to_string!(delta.old_value, "0.0");
                    field = Some(field!(
                        "trade_volume_usd",
                        trade_volume_usd_new,
                        trade_volume_usd_old
                    ));
                }
                // "total_transactions" => {
                //     let total_transactions_new: String =
                //         proto_decode_to_string!(delta.new_value, "0");
                //     let total_transactions_old: String =
                //         proto_decode_to_string!(delta.old_value, "0");
                //     field = Some(field!(
                //         "total_transactions",
                //         total_transactions_new,
                //         total_transactions_old
                //     ));
                // }
                _ => {}
            }

            if field.is_some() {
                changes.table_changes.push(TableChange {
                    table: "token".to_string(),
                    pk: token_address.to_string(),
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: Operation::Update as i32,
                    fields: vec![field.unwrap()],
                });
            }
        }
        "global" => {
            let key = parts[1];
            match key {
                "usd" => {
                    let total_volume_usd_new: String =
                        proto_decode_to_string!(delta.new_value, "0.0");
                    let total_volume_usd_old: String =
                        proto_decode_to_string!(delta.old_value, "0.0");
                    field = Some(field!(
                        "trade_volume_usd",
                        total_volume_usd_new,
                        total_volume_usd_old
                    ));
                }
                "bnb" => {
                    let total_volume_bnb_new: String =
                        proto_decode_to_string!(delta.new_value, "0.0");
                    let total_volume_bnb_old: String =
                        proto_decode_to_string!(delta.old_value, "0.0");
                    field = Some(field!(
                        "trade_volume_bnb",
                        total_volume_bnb_new,
                        total_volume_bnb_old
                    ));
                }
                // "total_transactions" => {
                //     let total_transactions_new: String =
                //         proto_decode_to_string!(delta.new_value, "0");
                //     let total_transactions_old: String =
                //         proto_decode_to_string!(delta.old_value, "0");
                //     field = Some(field!(
                //         "total_transactions",
                //         total_transactions_new,
                //         total_transactions_old
                //     ));
                // }
                _ => {}
            }
            if field.is_some() {
                changes.table_changes.push(TableChange {
                    table: "pancake_factory".to_string(),
                    pk: PANCAKE_FACTORY.to_string(),
                    block_num: block.number,
                    ordinal: delta.ordinal,
                    operation: Operation::Update as i32,
                    fields: vec![field.unwrap()],
                });
            }
        }
        _ => {}
    }
}

fn handle_reserves(reserve: Reserve, changes: &mut DatabaseChanges) {
    //todo: handle all the pairDayData.Reserve0, Reserve1 and ReserveUsd
    // same with all the pairHourData.Reserve0, Reserve1 and ReserveUsd
    // prices?
}

fn handle_events(event: Event, changes: &mut DatabaseChanges, block: &Block) {
    match event.r#type.as_ref().unwrap() {
        Type::Swap(swap) => handle_swap_event(&swap, &event, changes, block),
        Type::Burn(burn) => handle_burn_event(&burn, &event, changes, block),
        Type::Mint(mint) => handle_mint_event(&mint, &event, changes, block),
    }
}

fn handle_swap_event(swap: &Swap, event: &Event, changes: &mut DatabaseChanges, block: &Block) {
    changes.table_changes.push(TableChange {
        table: "swap".to_string(),
        pk: swap.id.to_string(),
        block_num: block.number,
        ordinal: event.log_ordinal,
        operation: Operation::Create as i32,
        fields: vec![
            field_create_string!("id", swap.id),
            field_create_string!("transaction", event.transaction_id),
            field_create_string!("timestamp", event.timestamp),
            field_create_string!("pair", event.pair_address),
            field_create_string!("token_0", event.token0),
            field_create_string!("token_1", event.token1),
            field_create_string!("sender", swap.sender),
            field_create_string!("from", swap.from),
            field_create_string!("amount_0_in", swap.amount0_in),
            field_create_string!("amount_1_in", swap.amount1_in),
            field_create_string!("amount_0_out", swap.amount0_out),
            field_create_string!("amount_1_out", swap.amount1_out),
            field_create_string!("to", swap.to),
            field_create_string!("amount_usd", swap.amount_usd),
            field_create_string!("log_index", event.log_ordinal),
        ],
    });
    changes.table_changes.push(TableChange {
        table: "pancake_factory".to_string(),
        pk: PANCAKE_FACTORY.to_string(),
        block_num: block.number,
        ordinal: event.log_ordinal,
        operation: Operation::Update as i32,
        fields: vec![
            field!("total_volume_usd", "", ""),   //todo: handle into total
            field!("total_volume_bnb", "", ""),   //todo: handle into total
            field!("total_transactions", "", ""), //todo: handle into total
        ],
    });
}

fn handle_burn_event(burn: &Burn, event: &Event, changes: &mut DatabaseChanges, block: &Block) {
    changes.table_changes.push(TableChange {
        table: "burn".to_string(),
        pk: burn.id.to_string(),
        block_num: block.number,
        ordinal: event.log_ordinal,
        operation: Operation::Create as i32,
        fields: vec![
            field!("id", burn.id, ""),
            field!("transaction", event.transaction_id, ""),
            field!("pair", event.pair_address, ""),
            field!("token_0", event.token0, ""),
            field!("token_1", event.token1, ""),
            field!("liquidity", burn.liquidity, ""),
            field!("timestamp", event.timestamp, ""),
            field!("to", burn.to, ""),
            field!("sender", burn.sender, ""),
            field!("amount_0", burn.amount0, ""),
            field!("amount_1", burn.amount1, ""),
            field!("log_index", event.log_ordinal, ""),
            field!("amount_usd", burn.amount_usd, ""),
        ],
    });
}

fn handle_mint_event(mint: &Mint, event: &Event, changes: &mut DatabaseChanges, block: &Block) {
    changes.table_changes.push(TableChange {
        table: "mint".to_string(),
        pk: mint.id.to_string(),
        block_num: block.number,
        ordinal: event.log_ordinal,
        operation: Operation::Create as i32,
        fields: vec![
            field!("id", mint.id, ""),
            field!("transaction", event.transaction_id, ""),
            field!("pair", event.pair_address, ""),
            field!("token_0", event.token0, ""),
            field!("token_1", event.token1, ""),
            field!("to", mint.to, ""),
            field!("liquidity", mint.liquidity, ""),
            field!("timestamp", event.timestamp, ""),
            field!("sender", mint.sender, ""),
            field!("amount_0", mint.amount0, ""),
            field!("amount_1", mint.amount1, ""),
            field!("log_index", event.log_ordinal, ""),
            field!("amount_usd", mint.amount_usd, ""),
        ],
    });
}

fn join_sort_deltas(
    pair_deltas: StoreDeltas,
    token_deltas: StoreDeltas,
    total_deltas: StoreDeltas,
    volumes_deltas: StoreDeltas,
    reserves: Reserves,
    events: Events,
) -> Vec<Item> {
    struct SortableItem {
        ordinal: u64,
        item: Item,
    }

    let mut items: Vec<SortableItem> = Vec::new();

    for delta in pair_deltas.deltas {
        items.push(SortableItem {
            ordinal: delta.ordinal,
            item: Item::PairDelta(delta),
        })
    }

    for delta in token_deltas.deltas {
        items.push(SortableItem {
            ordinal: delta.ordinal,
            item: Item::TokenDelta(delta),
        })
    }

    for delta in total_deltas.deltas {
        items.push(SortableItem {
            ordinal: delta.ordinal,
            item: Item::TokenDelta(delta),
        })
    }

    for delta in volumes_deltas.deltas {
        items.push(SortableItem {
            ordinal: delta.ordinal,
            item: Item::VolumeDelta(delta),
        })
    }

    for reserve in reserves.reserves {
        items.push(SortableItem {
            ordinal: reserve.log_ordinal,
            item: Item::Reserve(reserve),
        })
    }

    for event in events.events {
        items.push(SortableItem {
            ordinal: event.log_ordinal,
            item: Item::Event(event),
        })
    }

    items.sort_by(|a, b| a.ordinal.cmp(&b.ordinal));
    return items.iter().map(|item| item.item.clone()).collect();
}
