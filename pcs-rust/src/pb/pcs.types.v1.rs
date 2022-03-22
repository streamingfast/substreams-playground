#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pairs {
    #[prost(message, repeated, tag="1")]
    pub pairs: ::std::vec::Vec<Pair>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pair {
    #[prost(string, tag="1")]
    pub address: std::string::String,
    #[prost(message, optional, tag="2")]
    pub erc20_token0: ::std::option::Option<Erc20Token>,
    #[prost(message, optional, tag="3")]
    pub erc20_token1: ::std::option::Option<Erc20Token>,
    #[prost(string, tag="4")]
    pub creation_transaction_id: std::string::String,
    #[prost(uint64, tag="5")]
    pub block_num: u64,
    #[prost(uint64, tag="6")]
    pub log_ordinal: u64,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Erc20Token {
    #[prost(string, tag="1")]
    pub address: std::string::String,
    #[prost(string, tag="2")]
    pub name: std::string::String,
    #[prost(string, tag="3")]
    pub symbol: std::string::String,
    #[prost(uint64, tag="4")]
    pub decimals: u64,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Reserves {
    #[prost(message, repeated, tag="1")]
    pub reserves: ::std::vec::Vec<Reserve>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Reserve {
    #[prost(uint64, tag="1")]
    pub log_ordinal: u64,
    #[prost(string, tag="2")]
    pub pair_address: std::string::String,
    #[prost(string, tag="3")]
    pub reserve0: std::string::String,
    #[prost(string, tag="4")]
    pub reserve1: std::string::String,
    #[prost(string, tag="5")]
    pub token0_price: std::string::String,
    #[prost(string, tag="6")]
    pub token1_price: std::string::String,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PcsBaseEvents {
    #[prost(message, repeated, tag="1")]
    pub pcs_base_event: ::std::vec::Vec<PcsBaseEvent>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PcsBaseEvent {
    #[prost(string, tag="1")]
    pub pair_address: std::string::String,
    #[prost(string, tag="2")]
    pub token0: std::string::String,
    #[prost(string, tag="3")]
    pub token1: std::string::String,
    #[prost(string, tag="4")]
    pub transaction_id: std::string::String,
    #[prost(uint64, tag="5")]
    pub timestamp: u64,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PcsEvent {
    #[prost(oneof="pcs_event::Event", tags="1, 2, 3, 4, 5, 6, 7")]
    pub event: ::std::option::Option<pcs_event::Event>,
}
pub mod pcs_event {
    #[derive(Clone, PartialEq, ::prost::Oneof)]
    pub enum Event {
        #[prost(message, tag="1")]
        PairCreatedEvent(super::PairCreatedEvent),
        #[prost(message, tag="2")]
        PairApprovalEvent(super::PairApprovalEvent),
        #[prost(message, tag="3")]
        PairBurnEvent(super::PairBurnEvent),
        #[prost(message, tag="4")]
        PairMintEvent(super::PairMintEvent),
        #[prost(message, tag="5")]
        PairSwapEvent(super::PairSwapEvent),
        #[prost(message, tag="6")]
        PairSyncEvent(super::PairSyncEvent),
        #[prost(message, tag="7")]
        PairTransferEvent(super::PairTransferEvent),
    }
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairCreatedEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub token0: std::vec::Vec<u8>,
    #[prost(bytes, tag="4")]
    pub token1: std::vec::Vec<u8>,
    #[prost(bytes, tag="5")]
    pub pair: std::vec::Vec<u8>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairApprovalEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub owner: std::vec::Vec<u8>,
    #[prost(bytes, tag="4")]
    pub spender: std::vec::Vec<u8>,
    /// bigInt
    #[prost(bytes, tag="5")]
    pub value: std::vec::Vec<u8>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairBurnEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub sender: std::vec::Vec<u8>,
    /// bigInt
    #[prost(bytes, tag="4")]
    pub amount0: std::vec::Vec<u8>,
    /// bigInt
    #[prost(bytes, tag="5")]
    pub amount1: std::vec::Vec<u8>,
    #[prost(bytes, tag="6")]
    pub to: std::vec::Vec<u8>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairMintEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub sender: std::vec::Vec<u8>,
    /// bigInt
    #[prost(bytes, tag="4")]
    pub amount0: std::vec::Vec<u8>,
    /// bigInt
    #[prost(bytes, tag="5")]
    pub amount1: std::vec::Vec<u8>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairSwapEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub sender: std::vec::Vec<u8>,
    #[prost(bytes, tag="4")]
    pub amount0_in: std::vec::Vec<u8>,
    #[prost(bytes, tag="5")]
    pub amount1_in: std::vec::Vec<u8>,
    #[prost(bytes, tag="6")]
    pub amount0_out: std::vec::Vec<u8>,
    #[prost(bytes, tag="7")]
    pub amount1_out: std::vec::Vec<u8>,
    #[prost(bytes, tag="8")]
    pub to: std::vec::Vec<u8>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairSyncEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub reserve0: std::vec::Vec<u8>,
    #[prost(bytes, tag="4")]
    pub reserve1: std::vec::Vec<u8>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct PairTransferEvent {
    #[prost(bytes, tag="1")]
    pub log_address: std::vec::Vec<u8>,
    #[prost(uint64, tag="2")]
    pub log_index: u64,
    #[prost(bytes, tag="3")]
    pub from: std::vec::Vec<u8>,
    #[prost(bytes, tag="4")]
    pub to: std::vec::Vec<u8>,
    #[prost(bytes, tag="5")]
    pub value: std::vec::Vec<u8>,
}
