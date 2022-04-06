use substreams::pb::substreams::{store_delta, StoreDelta, StoreDeltas};
use substreams::proto;

use crate::pb::eth::Block;
use crate::pcs::table_change::Operation;
use crate::pcs::{
    Burn, DatabaseChanges, Event, Events, Field, Mint, Reserve, Reserves, Swap, TableChange,
};
use crate::{pb, pcs, utils, Type};

#[derive(Clone)]
enum Item {
    PairDelta(StoreDelta),
    TokenDelta(StoreDelta),
    Reserve(Reserve),
    Event(Event),
}

pub fn process(
    block: &Block,
    pair_deltas: StoreDeltas,
    token_deltas: StoreDeltas,
    reserves: Reserves,
    events: Events,
    tokens_idx: u32,
) -> DatabaseChanges {
    let items = join_sort_deltas(pair_deltas, token_deltas, reserves, events);

    let mut database_changes: DatabaseChanges = DatabaseChanges {
        table_changes: vec![],
    };

    for item in items {
        match item {
            Item::PairDelta(delta) => {
                handle_pair_delta(delta, &block, &mut database_changes, tokens_idx)
            }
            Item::TokenDelta(delta) => handle_token_delta(delta, &mut database_changes),
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
                Field {
                    key: "id".to_string(),
                    new_value: pair.address.clone(),
                    old_value: "".to_string(),
                },
                Field {
                    key: "name".to_string(),
                    new_value: format!("{}-{}", token0.symbol, token1.symbol),
                    old_value: "".to_string(),
                },
                Field {
                    key: "block".to_string(),
                    new_value: block.number.to_string(),
                    old_value: "".to_string(),
                },
                Field {
                    key: "timestamp".to_string(),
                    new_value: block.timestamp(),
                    old_value: "".to_string(),
                },
            ],
        });
    }
}

fn handle_token_delta(delta: StoreDelta, changes: &mut DatabaseChanges) {
    if delta.operation == store_delta::Operation::Create as i32 {
        let token: pb::tokens::Token = proto::decode(delta.new_value).unwrap();

        changes.table_changes.push(TableChange {
            table: "token".to_string(),
            pk: token.address.clone(),
            operation: Operation::Create as i32,
            fields: vec![
                Field {
                    key: "id".to_string(),
                    new_value: token.address.clone(),
                    old_value: "".to_string(),
                },
                Field {
                    key: "name".to_string(),
                    new_value: token.name,
                    old_value: "".to_string(),
                },
                Field {
                    key: "symbol".to_string(),
                    new_value: token.symbol,
                    old_value: "".to_string(),
                },
                Field {
                    key: "decimals".to_string(),
                    new_value: token.decimals.to_string(),
                    old_value: "".to_string(),
                },
            ],
        });
    }
}

fn handle_reserves(reserve: Reserve, changes: &mut DatabaseChanges) {
    // todo
}

fn handle_events(event: Event, changes: &mut DatabaseChanges) {
    match event.r#type.clone().unwrap() {
        Type::Swap(swap) => handle_swap_event(swap, event, changes),
        Type::Burn(burn) => handle_burn_event(burn, event, changes),
        Type::Mint(mint) => handle_mint_event(mint, event, changes),
    }
}

fn handle_swap_event(swap: Swap, event: Event, changes: &mut DatabaseChanges) {
    changes.table_changes.push(TableChange {
        table: "swap".to_string(),
        pk: swap.id.clone(),
        operation: Operation::Create as i32,
        fields: vec![
            Field {
                key: "id".to_string(),
                new_value: swap.id,
                old_value: "".to_string(),
            },
            Field {
                key: "transaction".to_string(),
                new_value: event.transaction_id,
                old_value: "".to_string(),
            },
            Field {
                key: "pair".to_string(),
                new_value: event.pair_address,
                old_value: "".to_string(),
            },
            Field {
                key: "token_0".to_string(),
                new_value: event.token0,
                old_value: "".to_string(),
            },
            Field {
                key: "token_1".to_string(),
                new_value: event.token1,
                old_value: "".to_string(),
            },
            Field {
                key: "timestamp".to_string(),
                new_value: event.timestamp.to_string(),
                old_value: "".to_string(),
            },
            Field {
                key: "sender".to_string(),
                new_value: swap.sender,
                old_value: "".to_string(),
            },
            Field {
                key: "amount_0_in".to_string(),
                new_value: swap.amount0_in,
                old_value: "".to_string(),
            },
            Field {
                key: "amount_1_in".to_string(),
                new_value: swap.amount1_in,
                old_value: "".to_string(),
            },
            Field {
                key: "amount_0_out".to_string(),
                new_value: swap.amount0_out,
                old_value: "".to_string(),
            },
            Field {
                key: "amount_1_out".to_string(),
                new_value: swap.amount1_out,
                old_value: "".to_string(),
            },
            Field {
                key: "to".to_string(),
                new_value: swap.to,
                old_value: "".to_string(),
            },
            Field {
                key: "from".to_string(),
                new_value: swap.from,
                old_value: "".to_string(),
            },
        ],
    })
}

fn handle_burn_event(burn: Burn, event: Event, changes: &mut DatabaseChanges) {
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
        pk: burn.id.clone(),
        operation: Operation::Create as i32,
        fields: vec![
            Field {
                key: "id".to_string(),
                new_value: burn.id,
                old_value: "".to_string(),
            },
            Field {
                key: "transaction".to_string(),
                new_value: event.transaction_id,
                old_value: "".to_string(),
            },
            Field {
                key: "pair".to_string(),
                new_value: event.pair_address,
                old_value: "".to_string(),
            },
            Field {
                key: "token_0".to_string(),
                new_value: event.token0,
                old_value: "".to_string(),
            },
            Field {
                key: "token_1".to_string(),
                new_value: event.token1,
                old_value: "".to_string(),
            },
            Field {
                key: "liquidity".to_string(),
                new_value: burn.liquidity,
                old_value: "".to_string(),
            },
            Field {
                key: "timestamp".to_string(),
                new_value: event.timestamp.to_string(),
                old_value: "".to_string(),
            },
            Field {
                key: "to".to_string(),
                new_value: burn.to,
                old_value: "".to_string(),
            },
            Field {
                key: "sender".to_string(),
                new_value: burn.sender,
                old_value: "".to_string(),
            },
        ],
    });
}

fn handle_mint_event(mint: Mint, event: Event, changes: &mut DatabaseChanges) {
    changes.table_changes.push(TableChange {
        table: "mint".to_string(),
        pk: mint.id.clone(),
        operation: Operation::Create as i32,
        fields: vec![
            Field {
                key: "id".to_string(),
                new_value: mint.id,
                old_value: "".to_string(),
            },
            Field {
                key: "transaction".to_string(),
                new_value: event.transaction_id,
                old_value: "".to_string(),
            },
            Field {
                key: "pair".to_string(),
                new_value: event.pair_address,
                old_value: "".to_string(),
            },
            Field {
                key: "token_0".to_string(),
                new_value: event.token0,
                old_value: "".to_string(),
            },
            Field {
                key: "token_1".to_string(),
                new_value: event.token1,
                old_value: "".to_string(),
            },
            Field {
                key: "to".to_string(),
                new_value: mint.to,
                old_value: "".to_string(),
            },
            Field {
                key: "liquidity".to_string(),
                new_value: mint.liquidity,
                old_value: "".to_string(),
            },
            Field {
                key: "timestamp".to_string(),
                new_value: event.timestamp.to_string(),
                old_value: "".to_string(),
            },
        ],
    });
}

fn join_sort_deltas(
    pair_deltas: StoreDeltas,
    token_deltas: StoreDeltas,
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
