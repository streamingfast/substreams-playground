use num_bigint::BigUint;
use crate::{pb, pcs};
use crate::pcs::{PairApprovalEvent, PairBurnEvent, PairCreatedEvent, PairMintEvent, PairSwapEvent, PairSyncEvent, PairTransferEvent, PcsEvent};
use crate::pcs::pcs_event::Event;

pub fn is_pair_created_event(sig: &str) -> bool {
    /* keccak value for PairCreated(address,address,address,uint256) */
    return sig == "0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9";
}

pub fn is_pair_approval_event(sig: &str) -> bool {
    /* keccak value for Approval(address,address,uint256) */
    return sig == "8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925";
}

pub fn is_pair_burn_event(sig: &str) -> bool {
    /* keccak value for Burn(address,uint256,uint256,address) */
    return sig == "dccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496";
}

pub fn is_pair_mint_event(sig: &str) -> bool {
    /* keccak value for Mint(address,uint256,uint256) */
    return sig == "4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f";
}

pub fn is_pair_swap_event(sig: &str) -> bool {
    /* keccak value for Swap(address,uint256,uint256,uint256,uint256,address) */
    return sig == "d78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822";
}

pub fn is_pair_sync_event(sig: &str) -> bool {
    /* keccak value for Sync(uint112,uint112) */
    return sig == "1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1";
}

pub fn is_pair_transfer_event(sig: &str) -> bool {
    /* keccak value for Transfer(address,address,uint256) */
    return sig == "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef";
}

pub fn decode_event(log: pb::eth::Log) -> PcsEvent {
    let sig = hex::encode(&log.topics[0]);

    if is_pair_created_event(&sig) {
        return new_pair_created_event(log);
    }

    if is_pair_approval_event(&sig) {
        return new_pair_approval_event(log);
    }

    if is_pair_burn_event(&sig) {
        return new_pair_burn_event(log);
    }

    if is_pair_mint_event(&sig) {
        return new_pair_mint_event(log);
    }

    if is_pair_swap_event(&sig) {
        return new_pair_swap_event(log);
    }

    if is_pair_sync_event(&sig) {
        return new_pair_sync_event(log);
    }

    if is_pair_transfer_event(&sig) {
        return new_pair_transfer_event(log);
    }

    return PcsEvent {
        event: None
    };
}

pub fn process_mint(prices_store_idx: u32, pair: pcs::Pair, tr1: Option<&PairTransferEvent>, tr2: Option<&PairTransferEvent>, sync: Option<&PairSyncEvent>, mint: &PairMintEvent) {

}

pub fn process_burn(prices_store_idx: u32, pair: pcs::Pair, tr1: Option<&PairTransferEvent>, tr2: Option<&PairTransferEvent>, sync: Option<&PairSyncEvent>, burn: &PairBurnEvent) {

}

pub fn process_swap(prices_store_idx: u32, pair: pcs::Pair, sync: Option<&PairSyncEvent>, swap: Option<&PairSwapEvent>, from_addr: String) {

}

fn new_pair_created_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairCreatedEvent(PairCreatedEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            token0: Vec::from(&log.topics[1][12..]),
            token1: Vec::from(&log.topics[2][12..]),
            pair: Vec::from(&log.data[12..44])
        }))
    };
}

fn new_pair_approval_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairApprovalEvent(PairApprovalEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            owner: Vec::from(&log.topics[1][12..]),
            spender: Vec::from(&log.topics[2][12..]),
            value: Vec::from(&log.data[0..32])
        }))
    };
}

fn new_pair_burn_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairBurnEvent(PairBurnEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            sender: Vec::from(&log.topics[1][12..]),
            amount0: Vec::from(&log.data[0..32]),
            amount1: Vec::from(&log.data[32..64]),
            to: Vec::from(&log.topics[2][12..])
        }))
    };
}

fn new_pair_mint_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairMintEvent(PairMintEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            sender: Vec::from(&log.topics[1][12..]),
            amount0: Vec::from(&log.data[0..32]),
            amount1: Vec::from(&log.data[32..64])
        }))
    };
}

fn new_pair_swap_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairSwapEvent(PairSwapEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            sender: Vec::from(&log.topics[1][12..]),
            amount0_in: Vec::from(&log.data[0..32]),
            amount1_in: Vec::from(&log.data[32..64]),
            amount0_out: Vec::from(&log.data[64..96]),
            amount1_out: Vec::from(&log.data[96..128]),
            to: Vec::from(&log.topics[2][12..])
        }))
    };
}

fn new_pair_sync_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairSyncEvent(PairSyncEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            reserve0: Vec::from(&log.data[0..32]),
            reserve1: Vec::from(&log.data[32..64])
        }))
    };
}

fn new_pair_transfer_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairTransferEvent(PairTransferEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            from: Vec::from(&log.topics[1][12..]),
            to: Vec::from(&log.topics[2][12..]),
            value: Vec::from(&log.data[0..32])
        }))
    };
}
