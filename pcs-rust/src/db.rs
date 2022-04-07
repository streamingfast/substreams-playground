use substreams::pb::substreams::{store_delta, StoreDelta, StoreDeltas};
use substreams::{log, proto};

use crate::pb::eth::Block;
use crate::pcs::table_change::Operation;
use crate::pcs::{
    Burn, DatabaseChanges, Event, Events, Field, Mint, Reserve, Reserves, Swap, TableChange,
};
use crate::{field, pb, pcs, utils, Type};

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
            Item::TokenDelta(delta) => handle_token_delta(delta, &mut database_changes),
            Item::TotalDelta(delta) => handle_total_delta(delta, &mut database_changes),
            Item::VolumeDelta(delta) => handle_volume_delta(delta, &mut database_changes),
            Item::Reserve(reserve) => handle_reserves(reserve, &mut database_changes),
            Item::Event(event) => handle_events(event, &mut database_changes),
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
    if delta.operation == store_delta::Operation::Create as i32 {
        let pair: pcs::Pair = proto::decode(delta.new_value).unwrap();

        let token0 = utils::get_last_token(tokens_idx, pair.token0_address.as_str());
        let token1 = utils::get_last_token(tokens_idx, pair.token1_address.as_str());

        changes.table_changes.push(TableChange {
            table: "pair".to_string(),
            pk: pair.address.clone(),
            operation: Operation::Create as i32,
            fields: vec![
                field!("id", pair.address.clone(), ""),
                field!("name", format!("{}-{}", token0.symbol, token1.symbol), ""),
                field!("block", block.number, ""),
                field!("timestamp", block.timestamp(), ""),
            ],
        });
    }
    // fixme: is there an update ?
}

fn handle_token_delta(delta: StoreDelta, changes: &mut DatabaseChanges) {
    if delta.operation == store_delta::Operation::Create as i32 {
        let token: pb::tokens::Token = proto::decode(delta.new_value).unwrap();

        changes.table_changes.push(TableChange {
            table: "token".to_string(),
            pk: token.address.clone(),
            operation: Operation::Create as i32,
            fields: vec![
                field!("id", token.address, ""),
                field!("name", token.name, ""),
                field!("symbol", token.symbol, ""),
                field!("decimals", token.decimals, ""),
            ],
        });
    }
}

fn handle_total_delta(delta: StoreDelta, changes: &mut DatabaseChanges) {
    let parts: Vec<&str> = delta.key.split(":").collect();
    let table = parts[0];

    //todo: @alex, how do we add the event to the list of events in the table of transactions in all the
    // use-cases of swaps, burns and mints
    match table {
        "pair" => {
            let pair_address = parts[1];
            let event_type = parts[2];
            match event_type {
                "swaps" => changes.table_changes.push(TableChange {
                    table: "transaction".to_string(),
                    pk: pair_address.to_string(),
                    operation: Operation::Update as i32,
                    fields: vec![field!("swaps", pair_address, "")],
                }),
                "burns" => changes.table_changes.push(TableChange {
                    table: "transaction".to_string(),
                    pk: pair_address.to_string(),
                    operation: Operation::Update as i32,
                    fields: vec![field!("burns", pair_address, "")],
                }),
                "mints" => changes.table_changes.push(TableChange {
                    table: "transaction".to_string(),
                    pk: pair_address.to_string(),
                    operation: Operation::Update as i32,
                    fields: vec![field!("mints", pair_address, "")],
                }),
                _ => {}
            }
        }
        "pancake_factory" => changes.table_changes.push(TableChange {
            table: "pancake_factory".to_string(),
            pk: PANCAKE_FACTORY.to_string(),
            operation: Operation::Update as i32,
            fields: vec![Field {
                key: "total_pairs".to_string(),
                new_value: proto::decode(delta.new_value).unwrap(),
                old_value: proto::decode(delta.old_value).unwrap(),
            }],
        }),
        _ => {}
    }
}

fn handle_volume_delta(delta: StoreDelta, changes: &mut DatabaseChanges) {
    let parts: Vec<&str> = delta.key.split(":").collect();
    let table = parts[0];

    match table {
        //todo: update the totals here, from volumes in lib.rs
        // and maybe even add more keys inside it to be able to pick up as much information as possible
        "pairs" => {}
        "token" => {}
        _ => {}
    }

    // changes.table_changes.push(TableChange {
    //     table: "token".to_string(),
    //     pk: event.token0.to_string(),
    //     operation: Operation::Update as i32,
    //     fields: vec![
    //         Field {
    //             key: "trade_volume".to_string(),
    //             new_value: delta.trade_volume0.to_string(),
    //             old_value: "".to_string(), //todo: how to get the value?
    //         },
    //         Field {
    //             key: "trade_volume_usd".to_string(),
    //             new_value: swap.trade_volume_usd0.to_string(),
    //             old_value: "".to_string(), //todo: how to get the value?
    //         },
    //         Field {
    //             key: "trade_volume".to_string(),
    //             new_value: swap.trade_volume1.to_string(),
    //             old_value: "".to_string(), //todo: how to get the value?
    //         },
    //         Field {
    //             key: "trade_volume_usd".to_string(),
    //             new_value: swap.trade_volume_usd1.to_string(),
    //             old_value: "".to_string(), //todo: how to get the value?
    //         },
    //     ],
    // });
}

fn handle_reserves(reserve: Reserve, changes: &mut DatabaseChanges) {
    // todo
}

fn handle_events(event: Event, changes: &mut DatabaseChanges) {
    match event.r#type.as_ref().unwrap() {
        Type::Swap(swap) => handle_swap_event(&swap, &event, changes),
        Type::Burn(burn) => handle_burn_event(&burn, &event, changes),
        Type::Mint(mint) => handle_mint_event(&mint, &event, changes),
    }
}

fn handle_swap_event(swap: &Swap, event: &Event, changes: &mut DatabaseChanges) {
    changes.table_changes.push(TableChange {
        table: "swap".to_string(),
        pk: swap.id.to_string(),
        operation: Operation::Create as i32,
        fields: vec![
            field!("id", swap.id, ""),
            field!("transaction", event.transaction_id, ""),
            field!("timestamp", event.timestamp, ""),
            field!("pair", event.pair_address, ""),
            field!("token_0", event.token0, ""),
            field!("token_1", event.token1, ""),
            field!("sender", swap.sender, ""),
            field!("from", swap.from, ""),
            field!("amount_0_in", swap.amount0_in, ""),
            field!("amount_1_in", swap.amount1_in, ""),
            field!("amount_0_out", swap.amount0_out, ""),
            field!("amount_1_out", swap.amount1_out, ""),
            field!("to", swap.to, ""),
            field!("amount_usd", swap.amount_usd, ""),
        ],
    });
    changes.table_changes.push(TableChange {
        table: "pair".to_string(),
        pk: swap.log_address.to_string(),
        operation: Operation::Update as i32,
        fields: vec![
            field!("volume_token_0", swap.volume_token0, ""),
            field!("volume_token_1", swap.volume_token1, ""),
            field!("volume_usd", swap.volume_usd, ""),
        ],
    });
    changes.table_changes.push(TableChange {
        table: "pancake_factory".to_string(),
        pk: PANCAKE_FACTORY.to_string(),
        operation: Operation::Update as i32,
        fields: vec![
            field!("total_volume_usd", "", ""),
            field!("total_volume_bnb", "", ""),
            field!("total_transactions", "", ""),
        ],
    });
}

fn handle_burn_event(burn: &Burn, event: &Event, changes: &mut DatabaseChanges) {
    /* TODO: how can we compute the old and new value
    database_changes.table_changes.push(TableChange {
        table: "pair".to_string(),
        pk: burn.pair_address,
        operation: Operation::Update as i32,
        fields: vec![ Field {
            key: "total_supply".to_string(),
            new_value: "".to_string(),
            old_value: "".to_string()
        }]
    })
    */
    changes.table_changes.push(TableChange {
        table: "burn".to_string(),
        pk: burn.id.to_string(),
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
        ],
    });
}

fn handle_mint_event(mint: &Mint, event: &Event, changes: &mut DatabaseChanges) {
    changes.table_changes.push(TableChange {
        table: "mint".to_string(),
        pk: mint.id.to_string(),
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
