fn main() {
    #[cfg(target_os = "linux")]
    if std::env::var_os("CARGO_FEATURE_NATIVE_VAULT_CANDIDATE").is_some() {
        build_native_vault_candidate();
    }
    tauri_build::build()
}

#[cfg(target_os = "linux")]
fn build_native_vault_candidate() {
    use std::{path::PathBuf, process::Command};

    let source = "src/credentials/native_vault_candidate.c";
    println!("cargo:rerun-if-changed={source}");
    let out = PathBuf::from(std::env::var_os("OUT_DIR").expect("OUT_DIR"));
    let flags = Command::new("pkg-config")
        .args(["--cflags", "libsecret-1", "gio-2.0"])
        .output()
        .expect("pkg-config required for native vault candidate");
    assert!(flags.status.success(), "libsecret development headers required");
    let flags = String::from_utf8(flags.stdout).expect("pkg-config flags");
    let object = out.join("native_vault_candidate.o");
    let status = Command::new("cc")
        .args(["-std=c11", "-Wall", "-Wextra", "-Werror", "-fPIC"])
        .args(flags.split_whitespace())
        .args(["-c", source, "-o"])
        .arg(&object)
        .status()
        .expect("native vault candidate compiler");
    assert!(status.success(), "native vault candidate compile failed");
    let status = Command::new("ar")
        .arg("crs")
        .arg(out.join("libghostftp_native_vault_candidate.a"))
        .arg(object)
        .status()
        .expect("native vault candidate archiver");
    assert!(status.success(), "native vault candidate archive failed");
    println!("cargo:rustc-link-search=native={}", out.display());
    println!("cargo:rustc-link-lib=static=ghostftp_native_vault_candidate");
    for library in ["secret-1", "gio-2.0", "gobject-2.0", "glib-2.0"] {
        println!("cargo:rustc-link-lib={library}");
    }
}
