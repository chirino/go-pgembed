#!/bin/bash
set -e

# Ensure this script is run from the project root
if [ ! -f "go.mod" ] || [ ! -d "rust_lib" ]; then
    echo "Error: This script must be run from the root of the gopostgresembedded project."
    exit 1
fi

RUST_LIB_DIR="rust_lib"
OUTPUT_DIR="libs"
LIB_NAME="librust_pg_embedded_lib" # Assumes 'name' in rust_lib/Cargo.toml


# --- Build natively for macOS ARM64 (Apple Silicon) ---
if [[ $(uname -s) == "Darwin" ]]; then
    TARGET="aarch64-apple-darwin"
    TARGET_DIR_MACOS_ARM="${OUTPUT_DIR}/darwin_arm64"

    echo "Building ${LIB_NAME} for ${TARGET}..."
    rustup target add aarch64-apple-darwin

    (cd "${RUST_LIB_DIR}" && rustup run stable cargo build --release --target "${TARGET}")
    mkdir -p "${TARGET_DIR_MACOS_ARM}"

    cp "${RUST_LIB_DIR}/target/${TARGET}/release/${LIB_NAME}.a" "${TARGET_DIR_MACOS_ARM}/"
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
    echo "Built for ${TARGET} and copied to ${TARGET_DIR_LINUX}/"
fi

echo "All requested Rust libraries built successfully." 