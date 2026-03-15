use std::env;
use std::fs;
use std::path::PathBuf;
use std::process::Command;

fn main() {
    println!("cargo:rerun-if-changed=../../engine");

    if env::var("PROFILE").as_deref() == Ok("release") {
        build_release_engine();
    }

    tauri_build::build()
}

fn build_release_engine() {
    let manifest_dir =
        PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("missing CARGO_MANIFEST_DIR"));
    let engine_dir = manifest_dir.join("../../engine");
    let bin_dir = manifest_dir.join("bin");
    let exe_suffix = env::consts::EXE_SUFFIX;
    let output = bin_dir.join(format!("welife-engine{}", exe_suffix));

    fs::create_dir_all(&bin_dir).expect("create src-tauri/bin");

    let status = Command::new("go")
        .args(["build", "-o"])
        .arg(&output)
        .arg("./cmd/welife")
        .current_dir(&engine_dir)
        .status()
        .expect("failed to start `go build` for engine");

    if !status.success() {
        panic!("go build for engine failed with status {status}");
    }
}
