#!/bin/bash
set -e

# Ensure this script is run from the project root for correct relative paths
if [ ! -f "go.mod" ] || [ ! -d "rust_lib" ]; then
    echo "Error: This script must be run from the root of the gopostgresembedded project."
    exit 1
fi

RUST_LIB_DIR="rust_lib"
OUTPUT_DIR="libs" # Store libs in the project root

# Ensure the Rust library name matches your Cargo.toml
# From rust_lib/Cargo.toml: name = "rust_pg_embedded_lib"
# So the library file will be librust_pg_embedded_lib.a
LIB_NAME="librust_pg_embedded_lib"

echo "Ensuring Rust targets are installed..."
rustup target add x86_64-unknown-linux-gnu
rustup target add x86_64-pc-windows-gnu
rustup target add aarch64-apple-darwin

# Clean previous builds if any
rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# --- Build for Linux x86_64 ---
TARGET_TRIPLE_LINUX="x86_64-unknown-linux-gnu"
TARGET_DIR_LINUX="${OUTPUT_DIR}/linux_amd64"
echo "Building ${LIB_NAME} for ${TARGET_TRIPLE_LINUX}..."
(cd "${RUST_LIB_DIR}" && cargo build --release --target "${TARGET_TRIPLE_LINUX}")
mkdir -p "${TARGET_DIR_LINUX}"
cp "${RUST_LIB_DIR}/target/${TARGET_TRIPLE_LINUX}/release/${LIB_NAME}.a" "${TARGET_DIR_LINUX}/"
echo "Built for ${TARGET_TRIPLE_LINUX} and copied to ${TARGET_DIR_LINUX}/"

# --- Build for Windows x86_64 (using GNU toolchain) ---
TARGET_TRIPLE_WINDOWS="x86_64-pc-windows-gnu"
TARGET_DIR_WINDOWS="${OUTPUT_DIR}/windows_amd64"
echo "Building ${LIB_NAME} for ${TARGET_TRIPLE_WINDOWS}..."
(cd "${RUST_LIB_DIR}" && cargo build --release --target "${TARGET_TRIPLE_WINDOWS}")
mkdir -p "${TARGET_DIR_WINDOWS}"
cp "${RUST_LIB_DIR}/target/${TARGET_TRIPLE_WINDOWS}/release/${LIB_NAME}.a" "${TARGET_DIR_WINDOWS}/" # .a for GNU
echo "Built for ${TARGET_TRIPLE_WINDOWS} and copied to ${TARGET_DIR_WINDOWS}/"

# --- Build for macOS ARM64 (Apple Silicon) ---
TARGET_TRIPLE_MACOS_ARM="aarch64-apple-darwin"
TARGET_DIR_MACOS_ARM="${OUTPUT_DIR}/darwin_arm64"
echo "Building ${LIB_NAME} for ${TARGET_TRIPLE_MACOS_ARM}..."
(cd "${RUST_LIB_DIR}" && cargo build --release --target "${TARGET_TRIPLE_MACOS_ARM}")
mkdir -p "${TARGET_DIR_MACOS_ARM}"
cp "${RUST_LIB_DIR}/target/${TARGET_TRIPLE_MACOS_ARM}/release/${LIB_NAME}.a" "${TARGET_DIR_MACOS_ARM}/"
echo "Built for ${TARGET_TRIPLE_MACOS_ARM} and copied to ${TARGET_DIR_MACOS_ARM}/"

echo "All Rust libraries built successfully and copied to ${OUTPUT_DIR}/" 