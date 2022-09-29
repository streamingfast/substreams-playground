mod pb;
mod instruction;
mod option;
mod helper;
mod keyer;
use std::str::FromStr;
use bigdecimal::BigDecimal;
use num_bigint::BigInt;
use substreams::store::StoreGet;

use crate::instruction::TokenInstruction;
use {
    bs58,
    substreams::{errors::Error, log, proto, store},
    substreams_solana::pb as solpb
};
use crate::option::COption;

#[substreams::handlers::map]
fn map_mints(blk: solpb::sol::v1::Block) -> Result<pb::spl::Mints, Error> {
    log::info!("extracting mints");
    let mut mints = vec![] ;
    for trx in blk.transactions {
        if let Some(meta) = trx.meta {
            if let Some(_) = meta.err {
                continue;
            }
            if let Some(transaction) = trx.transaction {
                if let Some(msg) = transaction.message {
                    for inst in msg.instructions {
                        let program_id = &msg.account_keys[inst.program_id_index as usize];
                        if bs58::encode(program_id).into_string() != "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
                            continue;
                        }
                        let instruction = TokenInstruction::unpack(&inst.data)?;
                        match instruction {
                            TokenInstruction::InitializeMint {
                                decimals,
                                mint_authority,
                                freeze_authority,
                            } => {
                                log::info!("Instruction: InitializeMint");
                                mints.push(get_mint(
                                    msg.account_keys[inst.accounts[0] as usize].to_vec(),
                                    decimals,
                                        mint_authority,
                                    freeze_authority
                                ));
                            }
                            TokenInstruction::InitializeMint2 {
                                decimals,
                                mint_authority,
                                freeze_authority,
                            } => {
                                log::info!("Instruction: InitializeMint2");
                                mints.push(get_mint(
                                    msg.account_keys[inst.accounts[0] as usize].to_vec(),
                                    decimals,
                                    mint_authority,
                                    freeze_authority
                                ));
                            }
                            _ => {}
                        }
                    }
                }
            }
        }
    }
    return Ok(pb::spl::Mints { mints });
}

#[substreams::handlers::store]
pub fn store_mints(mints: pb::spl::Mints, output: store::StoreSet) {
    log::info!("building mints store");
    for mint in mints.mints {
        output.set(
            0,
            keyer::mint_key(&mint.address),
            &proto::encode(&mint).unwrap(),
        );
    }
}

#[substreams::handlers::map]
fn map_accounts(blk: solpb::sol::v1::Block) -> Result<pb::spl::Accounts, Error> {
    log::info!("extracting accounts");
    let mut accounts = vec![] ;
    for trx in blk.transactions {
        if let Some(meta) = trx.meta {
            if let Some(_) = meta.err {
                continue;
            }
            if let Some(transaction) = trx.transaction {
                if let Some(msg) = transaction.message {
                    for inst in msg.instructions {
                        let program_id = &msg.account_keys[inst.program_id_index as usize];
                        if bs58::encode(program_id).into_string() != "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
                            continue;
                        }
                        let instruction = TokenInstruction::unpack(&inst.data)?;
                        match instruction {
                            TokenInstruction::InitializeAccount => {
                                log::info!("Instruction: InitializeAccount");
                                accounts.push(get_account(
                                    msg.account_keys[inst.accounts[0] as usize].to_vec(),
                                msg.account_keys[inst.accounts[1] as usize].to_vec(),
                                msg.account_keys[inst.accounts[2] as usize].to_vec(),
                                ));
                            }
                            TokenInstruction::InitializeAccount2 { owner } => {
                                log::info!("Instruction: InitializeAccount2");
                                accounts.push(get_account(
                                    msg.account_keys[inst.accounts[0] as usize].to_vec(),
                                    msg.account_keys[inst.accounts[1] as usize].to_vec(),
                                    owner,
                                ));
                            }
                            TokenInstruction::InitializeAccount3 { owner } => {
                                log::info!("Instruction: InitializeAccount3");
                                accounts.push(get_account(
                                    msg.account_keys[inst.accounts[0] as usize].to_vec(),
                                    msg.account_keys[inst.accounts[1] as usize].to_vec(),
                                    owner,
                                ));
                            }

                            _ => {}
                        }
                    }
                }
            }
        }
    }
    return Ok(pb::spl::Accounts { accounts });
}

#[substreams::handlers::store]
pub fn store_accounts(accounts: pb::spl::Accounts, output: store::StoreSet) {
    log::info!("building accounts store");
    for account in accounts.accounts {
        output.set(
            0,
            keyer::account_key(&account.address),
            &proto::encode(&account).unwrap(),
        );
    }
}

#[substreams::handlers::map]
fn map_transfers(blk: solpb::sol::v1::Block, mint_store: StoreGet, account_store: StoreGet) -> Result<pb::spl::TokenTransfers, Error> {
    log::info!("extracting transfers");
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
                        if bs58::encode(program_id).into_string() != "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
                            continue;
                        }

                        let mut native_amount: u64 = 0;
                        let mut from_account_addr = "".to_string();
                        let mut to_account_addr = "".to_string();
                        let mut mint_addr = "".to_string();


                        let instruction = TokenInstruction::unpack(&inst.data)?;
                        match instruction {
                            TokenInstruction::Transfer { amount } => {
                                log::info!("Instruction: Transfer");
                                native_amount = amount;
                                from_account_addr  = bs58::encode(&msg.account_keys[inst.accounts[0] as usize].to_vec()).into_string();
                                to_account_addr  = bs58::encode(&msg.account_keys[inst.accounts[1] as usize].to_vec()).into_string();
                            }
                            TokenInstruction::TransferChecked { amount, decimals: _ } => {
                                log::info!("Instruction: TransferChecked");
                                native_amount = amount;
                                from_account_addr  = bs58::encode(&msg.account_keys[inst.accounts[0] as usize].to_vec()).into_string();
                                mint_addr  = bs58::encode(&msg.account_keys[inst.accounts[1] as usize].to_vec()).into_string();
                                to_account_addr = bs58::encode(&msg.account_keys[inst.accounts[2] as usize].to_vec()).into_string();
                            },
                            _ => {}
                        }

                        if mint_addr == "" {
                            log::info!("resolving mint_addr from account: {}", from_account_addr);
                            let account_res = helper::get_account(&account_store, &from_account_addr);
                            if account_res.is_err() {
                                log::info!("skipping transfer where account is not found: {}", from_account_addr);
                                continue
                            }
                            let account = account_res.unwrap();
                            mint_addr = account.mint;
                        }

                        let mint_res = helper::get_mint(&mint_store, &mint_addr);
                        if mint_res.is_err() {
                            log::info!("skipping transfer where mint is not found: {}", mint_addr);
                            continue
                        }
                        let mint = mint_res.unwrap();
                        let normalized_value = helper::convert_token_to_decimal(&BigInt::from(native_amount), mint.decimals.into());
                        transfers.push(pb::spl::TokenTransfer{
                            transaction_id: bs58::encode(&transaction.signatures[0]).into_string(),
                            ordinal: 0,
                            from: from_account_addr,
                            to: to_account_addr,
                            mint: mint.address,
                            amount: normalized_value.to_string(),
                            native_amount,
                        })
                    }
                }
            }
        }
    }
    return Ok(pb::spl::TokenTransfers { transfers });
}


#[substreams::handlers::store]
pub fn store_mint_native_volumes(transfers: pb::spl::TokenTransfers, output: store::StoreAddBigInt) {
    log::info!("building mint volume store");
    for transfer in transfers.transfers {
        output.add(
            0,
            keyer::native_mint_volume(&transfer.mint),
            &BigInt::from(transfer.native_amount),
        );
    }
}

#[substreams::handlers::store]
pub fn store_mint_decimal_volumes(transfers: pb::spl::TokenTransfers, output: store::StoreAddBigFloat) {
    log::info!("building mint volume store");
    for transfer in transfers.transfers {
        let v = BigDecimal::from_str(&transfer.amount).unwrap();
        output.add(
            0,
            keyer::decimal_mint_volume(&transfer.mint),
            &v,
        );
    }
}

fn get_mint(mint_account: Vec<u8>, decimal: u8, mint_authority: Vec<u8>, freeze_authority_opt: COption<Vec<u8>>) -> pb::spl::Mint {
    let mut mint =  pb::spl::Mint{
        address: bs58::encode(&mint_account).into_string(),
        decimals: decimal.into(),
        mint_authority: bs58::encode(&mint_authority).into_string(),
        freeze_authority: "".to_string()
    };
    if freeze_authority_opt.is_some() {
        mint.freeze_authority = bs58::encode(&freeze_authority_opt.unwrap()).into_string();
    }
    return mint;
}

fn get_account(account: Vec<u8>,mint: Vec<u8>,owner: Vec<u8>) -> pb::spl::Account {
    return pb::spl::Account{
        address: bs58::encode(&account).into_string(),
        mint: bs58::encode(&mint).into_string(),
        owner: bs58::encode(&owner).into_string(),
    };
}
