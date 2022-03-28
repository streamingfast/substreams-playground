#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pools {
    #[prost(message, repeated, tag="1")]
    pub pools: ::std::vec::Vec<Pool>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pool {
    #[prost(string, tag="1")]
    pub token0: std::string::String,
    #[prost(string, tag="2")]
    pub token1: std::string::String,
}
