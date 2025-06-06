use std::{env, path::PathBuf};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());

    tonic_build::configure()
        .compile_well_known_types(true)
        .file_descriptor_set_path(out_dir.join("grpchub_descriptor.bin"))
        .compile_protos(
            &[
                "../proto/channel/v1/channel.proto",
                "../third_party/google/rpc/status.proto",
            ],
            &["../proto/", "../third_party"],
        )?;
    Ok(())
}
