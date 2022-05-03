mod pb;

use crate::pb::substreams::Request;
use prost::DecodeError;
use std::{env, fs};

use crate::pb::substreams::module_output::Data::MapOutput;
use crate::pb::substreams::response::Message;
use tokio_stream::StreamExt;

#[tokio::main]
async fn main() {
    let filename = env::args().nth(1).unwrap();
    let grpc_endpoint = env::args().nth(2).unwrap();

    let contents = fs::read(filename).expect("Something went wrong reading the file");
    let manifest: pb::substreams::Manifest = decode(&contents).unwrap();
    let request: pb::substreams::Request = Request {
        start_block_num: 6810706,
        start_cursor: "".to_string(),
        stop_block_num: 6810806,
        fork_steps: vec![],
        irreversibility_condition: "".to_string(),
        manifest: Some(manifest),
        output_modules: vec!["block_to_tokens".to_string()],
        initial_store_snapshot_for_modules: vec![],
    };

    let mut client = pb::substreams::stream_client::StreamClient::connect(grpc_endpoint)
        .await
        .unwrap();

    let request = tonic::Request::new(request);

    let mut stream = client.blocks(request).await.unwrap().into_inner();

    while let Some(resp) = stream.next().await {
        match resp.unwrap().message.unwrap() {
            Message::Progress(_) => {}
            Message::SnapshotData(_) => {}
            Message::SnapshotComplete(_) => {}
            Message::Data(data) => {
                for output in data.outputs {
                    for log in output.logs {
                        println!("Remote log: {}", log)
                    }

                    match output.name.as_str() {
                        "block_to_tokens" => match output.data.unwrap() {
                            MapOutput(map_output) => {
                                let tokens: pb::tokens::Tokens = decode(&map_output.value).unwrap();
                                for token in tokens.tokens {
                                    println!(
                                        "Token: name:{} address:{} symbol:{} decimals:{}",
                                        token.name, token.address, token.symbol, token.decimals
                                    )
                                }
                            }
                            _ => {}
                        },
                        _ => {}
                    }
                }
            }
        }
    }
}

pub fn decode<T: std::default::Default + prost::Message>(buf: &Vec<u8>) -> Result<T, DecodeError> {
    ::prost::Message::decode(&buf[..])
}
