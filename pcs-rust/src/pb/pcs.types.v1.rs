#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pairs {
    #[prost(message, repeated, tag="1")]
    pub pairs: ::std::vec::Vec<Pair>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pair {
    #[prost(string, tag="1")]
    pub address: std::string::String,
    #[prost(string, tag="2")]
    pub token0_address: std::string::String,
    #[prost(string, tag="3")]
    pub token1_address: std::string::String,
    #[prost(string, tag="4")]
    pub creation_transaction_id: std::string::String,
    #[prost(uint64, tag="5")]
    pub block_num: u64,
    #[prost(uint64, tag="6")]
    pub log_ordinal: u64,
}
//message ERC20Token {
//  string address = 1;
//  string name = 2;
//  string symbol = 3;
//  uint64 decimals = 4;
//}

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
pub struct Events {
    #[prost(message, repeated, tag="1")]
    pub events: ::std::vec::Vec<Event>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Event {
    #[prost(uint64, tag="100")]
    pub log_ordinal: u64,
    #[prost(string, tag="101")]
    pub pair_address: std::string::String,
    #[prost(string, tag="102")]
    pub token0: std::string::String,
    #[prost(string, tag="103")]
    pub token1: std::string::String,
    #[prost(string, tag="104")]
    pub transaction_id: std::string::String,
    #[prost(uint64, tag="105")]
    pub timestamp: u64,
    #[prost(oneof="event::Type", tags="1, 2, 3")]
    pub r#type: ::std::option::Option<event::Type>,
}
pub mod event {
    #[derive(Clone, PartialEq, ::prost::Oneof)]
    pub enum Type {
        #[prost(message, tag="1")]
        Swap(super::Swap),
        #[prost(message, tag="2")]
        Burn(super::Burn),
        #[prost(message, tag="3")]
        Mint(super::Mint),
    }
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Swap {
    #[prost(string, tag="1")]
    pub id: std::string::String,
    #[prost(string, tag="2")]
    pub sender: std::string::String,
    #[prost(string, tag="3")]
    pub to: std::string::String,
    #[prost(string, tag="4")]
    pub from: std::string::String,
    #[prost(string, tag="5")]
    pub amount0_in: std::string::String,
    #[prost(string, tag="6")]
    pub amount1_in: std::string::String,
    #[prost(string, tag="7")]
    pub amount0_out: std::string::String,
    #[prost(string, tag="8")]
    pub amount1_out: std::string::String,
    #[prost(string, tag="9")]
    pub amount_bnb: std::string::String,
    #[prost(string, tag="10")]
    pub amount_usd: std::string::String,
    #[prost(string, tag="11")]
    pub trade_volume0: std::string::String,
    #[prost(string, tag="12")]
    pub trade_volume1: std::string::String,
    #[prost(string, tag="13")]
    pub trade_volume_usd0: std::string::String,
    #[prost(string, tag="14")]
    pub trade_volume_usd1: std::string::String,
    // dropped for now...
    //string untracked_volume_usd0 = 15;
    //string untracked_volume_usd1 = 16;

    #[prost(string, tag="17")]
    pub volume_usd: std::string::String,
    #[prost(string, tag="18")]
    pub volume_token0: std::string::String,
    #[prost(string, tag="19")]
    pub volume_token1: std::string::String,
    #[prost(string, tag="20")]
    pub log_address: std::string::String,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Burn {
    #[prost(string, tag="1")]
    pub id: std::string::String,
    #[prost(string, tag="2")]
    pub sender: std::string::String,
    #[prost(string, tag="3")]
    pub to: std::string::String,
    #[prost(string, tag="4")]
    pub fee_to: std::string::String,
    #[prost(string, tag="5")]
    pub amount0: std::string::String,
    #[prost(string, tag="6")]
    pub amount1: std::string::String,
    #[prost(string, tag="7")]
    pub amount_usd: std::string::String,
    #[prost(string, tag="8")]
    pub liquidity: std::string::String,
    #[prost(string, tag="9")]
    pub fee_liquidity: std::string::String,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Mint {
    #[prost(string, tag="1")]
    pub id: std::string::String,
    #[prost(string, tag="2")]
    pub sender: std::string::String,
    #[prost(string, tag="3")]
    pub to: std::string::String,
    #[prost(string, tag="4")]
    pub fee_to: std::string::String,
    #[prost(string, tag="5")]
    pub amount0: std::string::String,
    #[prost(string, tag="6")]
    pub amount1: std::string::String,
    #[prost(string, tag="7")]
    pub amount_usd: std::string::String,
    #[prost(string, tag="8")]
    pub liquidity: std::string::String,
    #[prost(string, tag="9")]
    pub fee_liquidity: std::string::String,
}
