use std::io::Result;
fn main() -> Result<()> {
    prost_build::compile_protos(&["proto/codec_eth.proto"], &["src/"])?;
    Ok(())
}

// // use prost_build;

// fn main() {
//     tonic_build::configure()
//         .build_server(false)
//         //.out_dir("src/pb")
//         .compile(
//             &["proto/bstream.proto", "proto/codec_eth.proto"],
//             &["proto/"],
//         ).unwrap();
// }
