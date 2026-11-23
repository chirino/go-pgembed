#!/bin/bash
set -e

# Ensure this script is run from the project root
if [ ! -f "go.mod" ] || [ ! -d "rust" ]; then
    echo "Error: This script must be run from the root of the gopostgresembedded project."
    exit 1
fi

RUST_LIB_DIR="rust"
OUTPUT_DIR="libs"
LIB_NAME="libgo_pgembed_lib" # Assumes 'name' in rust/Cargo.toml

# Rust's prebuilt standard library embeds LLVM bitcode. Go 1.26.11+ rejects
# cgo libraries that contain LTO/bitcode sections, so strip them after copying.
find_llvm_objcopy() {
    if command -v llvm-objcopy >/dev/null 2>&1; then
        command -v llvm-objcopy
    elif command -v brew >/dev/null 2>&1 && [ -x "$(brew --prefix llvm)/bin/llvm-objcopy" ]; then
        echo "$(brew --prefix llvm)/bin/llvm-objcopy"
    else
        sysroot="$(rustc --print sysroot)"
        find "${sysroot}" -name llvm-objcopy -type f 2>/dev/null | head -n 1
    fi
}

strip_bitcode() {
    local objcopy
    objcopy="$(find_llvm_objcopy)"
    if [ -z "${objcopy}" ]; then
        echo "Error: llvm-objcopy not found (install llvm or run: rustup component add llvm-tools)."
        exit 1
    fi
    "${objcopy}" "$@"
}


# --- Build natively for macOS ARM64 (Apple Silicon) ---
if [[ $(uname -s) == "Darwin" ]]; then
    TARGET="aarch64-apple-darwin"
    TARGET_DIR_MACOS_ARM="${OUTPUT_DIR}/darwin_arm64"

    echo "Building ${LIB_NAME} for ${TARGET}..."
    rustup target add aarch64-apple-darwin

    (cd "${RUST_LIB_DIR}" && rustup run stable cargo build --release --target "${TARGET}")
    mkdir -p "${TARGET_DIR_MACOS_ARM}"

    cp "${RUST_LIB_DIR}/target/${TARGET}/release/${LIB_NAME}.a" "${TARGET_DIR_MACOS_ARM}/"
    strip_bitcode --remove-section=__LLVM,__bitcode --remove-section=__LLVM,__cmdline "${TARGET_DIR_MACOS_ARM}/${LIB_NAME}.a"
    echo "Built for ${TARGET} and copied to ${TARGET_DIR_MACOS_ARM}/"
fi

# --- Build Linux  ---
if [[ $(uname -s) == "Linux" ]]; then
    TARGET="x86_64-unknown-linux-gnu"
    TARGET_DIR_LINUX="${OUTPUT_DIR}/linux_amd64"
    
    echo "Building ${LIB_NAME} for ${TARGET}..."
    rustup target add x86_64-unknown-linux-gnu

    (cd "${RUST_LIB_DIR}" && rustup run stable cargo build --release --target "${TARGET}")
    mkdir -p "${TARGET_DIR_LINUX}"

    cp "${RUST_LIB_DIR}/target/${TARGET}/release/${LIB_NAME}.a" "${TARGET_DIR_LINUX}/"
    strip_bitcode --remove-section=.llvmbc --remove-section=.llvmcmd "${TARGET_DIR_LINUX}/${LIB_NAME}.a"
    echo "Built for ${TARGET} and copied to ${TARGET_DIR_LINUX}/"
fi

echo "All requested Rust libraries built successfully." 