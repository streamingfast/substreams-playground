use std::io::Result;
fn main() -> Result<()> {
    prost_build::compile_protos(&["proto/codec_eth.proto", "proto/pcs.proto"], &["src/"])?;
    Ok(())
}
