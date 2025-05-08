@echo off
setlocal enabledelayedexpansion

REM Ensure this script is run from the project root
if not exist "go.mod" (
    echo Error: This script must be run from the root of the gopostgresembedded project.
    exit /b 1
)
if not exist "rust_lib" (
    echo Error: This script must be run from the root of the gopostgresembedded project.
    exit /b 1
)

set "RUST_LIB_DIR=rust_lib"
set "OUTPUT_DIR=libs"
set "LIB_NAME=librust_pg_embedded_lib"

REM --- Build for Windows ---
set "TARGET=x86_64-pc-windows-msvc"
set "TARGET_DIR_WINDOWS=%OUTPUT_DIR%\windows_amd64"

echo Building %LIB_NAME% for %TARGET%...
rustup target add x86_64-pc-windows-msvc

cd %RUST_LIB_DIR%
rustup run stable cargo build --release --target %TARGET%
cd ..

if not exist "%TARGET_DIR_WINDOWS%" mkdir "%TARGET_DIR_WINDOWS%"

copy "%RUST_LIB_DIR%\target\%TARGET%\release\%LIB_NAME%.a" "%TARGET_DIR_WINDOWS%\"
echo Built for %TARGET% and copied to %TARGET_DIR_WINDOWS%\

echo All requested Rust libraries built successfully.
