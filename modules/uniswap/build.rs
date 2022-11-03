use anyhow::{Ok, Result};
use substreams_ethereum::Abigen;

fn main() -> Result<(), anyhow::Error> {
    Abigen::new("Factory", "abis/factory.json")?
        .generate()?
        .write_to_file("src/abi/factory.rs")?;

    Ok(())
}
