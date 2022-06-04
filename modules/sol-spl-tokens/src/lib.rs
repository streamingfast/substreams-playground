mod pb;
use std::convert::TryInto;
use {
    bs58,
    substreams::{errors::Error, log, proto, store},
    substreams_solana::pb as solpb
};

#[substreams::handlers::map]
fn map_transfers(blk: solpb::sol::v1::Block) -> Result<pb::spl::TokenTransfers, Error> {
    log::info!("extracting SPL Token Transfers");

    let mut transfers = vec![] ;

    for trx in blk.transactions {
        if let Some(meta) = trx.meta {
            if let Some(_) = meta.err {
                continue;
            }
            if let Some(transaction) = trx.transaction {
                if let Some(msg) = transaction.message {
                    for inst in msg.instructions {
                        let program_id = &msg.account_keys[inst.program_id_index as usize];
                        if bs58::encode(program_id).into_string()
                            != "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
                        {
                            continue;
                        }

                        if inst.data[0] != 0x03 {
                            continue;
                        }

                        let amount_res: Result<[u8; 8],_> = inst.data[1..9].try_into();
                        if amount_res.is_err() {
                            return Err(Error::Unexpected("unable to extract amount from instruction data".to_string()))
                        }
                        let amount = u64::from_le_bytes(amount_res.unwrap());

                        let from = &msg.account_keys[inst.accounts[0] as usize];
                        let to = &msg.account_keys[inst.accounts[1] as usize];

                        transfers.push(pb::spl::TokenTransfer {
                            transaction_id: bs58::encode(&transaction.signatures[0]).into_string(),
                            ordinal: 0,
                            from: bs58::encode(&from).into_string(),
                            to: bs58::encode(&to).into_string(),
                            amount,
                        });
                    }
                }
            }
        }
    }
    return Ok(pb::spl::TokenTransfers { transfers });
}

#[substreams::handlers::store]
pub fn transfer_store(transfers: pb::spl::TokenTransfers, output: store::StoreSet) {

    log::info!("building transfer state");
    for transfer in transfers.transfers {
        output.set(
            1,
            format!("transfer:{}", transfer.transaction_id),
            &proto::encode(&transfer).unwrap(),
        );
    }
}
