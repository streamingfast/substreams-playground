use crate::pb::eth::Block;
use crate::pcs::table_change::Operation;
use crate::pcs::{DatabaseChanges, Event, Events, Field, Reserve, Reserves, TableChange};
use crate::{pcs, utils};
use substreams::pb::substreams::{store_delta, StoreDelta, StoreDeltas};
use substreams::proto;

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
                hand_pair_delta(delta, &block, &mut database_changes, tokens_idx)
            }
            Item::TokenDelta(_delta) => {}
            Item::Reserve(_reserve) => {}
            Item::Event(_event) => {}
        }
    }

    return database_changes;
}

fn hand_pair_delta(
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
