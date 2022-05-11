fn main() {
    println!("Building proto");
    let mut prost_build = prost_build::Config::new();
    prost_build.out_dir("./src/pb");
    prost_build
        .compile_protos(
            &[
                "sf_solana_type_v1.proto",
                "solana_spl.proto",
            ],
            &["./proto"],
        )
        .unwrap();
    println!("Done!");
}
