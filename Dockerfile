ARG LIB_NAME=librust_pg_embedded_lib # Define library name globally for use in the final stage

# ===== Base Stage =====
# Installs common dependencies for cross-compiling the Rust library.
# Runs on the host architecture (e.g., arm64 on Apple Silicon).
FROM debian:bookworm AS base_image

ENV DEBIAN_FRONTEND=noninteractive

# Install base tools, Rust, and C cross-compilers.
# Note: Cross-compilers are needed because the 'ring' crate (a likely dependency
# when using rustls) requires a C compiler for the *target* architecture.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl wget ca-certificates gnupg build-essential pkg-config \
    # Add amd64 architecture to allow installing amd64 tools on arm64 host
    && dpkg --add-architecture amd64 && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    # C toolchain for x86_64-unknown-linux-gnu target
    crossbuild-essential-amd64 \
    # C toolchain for x86_64-pc-windows-gnu target
    mingw-w64 \
    # Clean up apt cache
    && rm -rf /var/lib/apt/lists/*

# Install Rust via rustup
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain stable
ENV PATH="/root/.cargo/bin:${PATH}"

# Install Rust standard libraries for the target architectures
RUN rustup target add x86_64-unknown-linux-gnu
RUN rustup target add x86_64-pc-windows-gnu

# Set up working directory and copy Rust library source
WORKDIR /app
COPY rust_lib ./rust_lib
RUN cd rust_lib && cargo fetch

# ===== Linux Build Stage =====
# Builds the static library for Linux x86_64.
FROM base_image AS linux_builder

# Set environment variables for Cargo to use the correct C cross-compiler/linker.
ENV CC_x86_64_unknown_linux_gnu=x86_64-linux-gnu-gcc
ENV CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER=x86_64-linux-gnu-gcc

# Build the library for Linux
RUN cd rust_lib && cargo build --release --target x86_64-unknown-linux-gnu

# ===== Windows Build Stage =====
# Builds the static library for Windows x86_64.
FROM base_image AS windows_builder

# Set environment variables for Cargo to use the correct C cross-compiler/linker (MinGW).
ENV CC_x86_64_pc_windows_gnu=x86_64-w64-mingw32-gcc
ENV CARGO_TARGET_X86_64_PC_WINDOWS_GNU_LINKER=x86_64-w64-mingw32-gcc

# Build the library for Windows
RUN cd rust_lib && cargo build --release --target x86_64-pc-windows-gnu

# ===== Final Stage: Collect Artifacts =====
# Creates a minimal image containing only the compiled static libraries.
# Use 'docker build --output type=local,dest=./libs .' to extract these files.
FROM scratch AS release_artifacts

# Re-declare ARG within this stage to make it available for COPY commands below.
ARG LIB_NAME

# Copy the compiled libraries from the respective builder stages.
COPY --from=linux_builder /app/rust_lib/target/x86_64-unknown-linux-gnu/release/${LIB_NAME}.a /linux_amd64/${LIB_NAME}.a
COPY --from=windows_builder /app/rust_lib/target/x86_64-pc-windows-gnu/release/${LIB_NAME}.a /windows_amd64/${LIB_NAME}.a
