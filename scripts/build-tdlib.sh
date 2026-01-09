#!/bin/bash
# Build TDLib static libraries for tlgram
#
# This script builds TDLib from source with static linking support.
# It installs to /opt/tdlib by default, which can be overridden with INSTALL_PREFIX.
#
# Usage:
#   ./scripts/build-tdlib.sh
#
# Requirements:
#   - CMake 3.0.2+
#   - C++14 compiler (GCC 4.9.2+, Clang 3.4+)
#   - OpenSSL
#   - zlib
#   - gperf

set -euo pipefail

# Configuration
TDLIB_VERSION="${TDLIB_VERSION:-v1.8.31}"
INSTALL_PREFIX="${INSTALL_PREFIX:-/opt/tdlib}"
BUILD_DIR="${BUILD_DIR:-/tmp/tdlib-build}"
JOBS="${JOBS:-$(nproc)}"

echo "=============================================="
echo "TDLib Build Script"
echo "=============================================="
echo "Version:     $TDLIB_VERSION"
echo "Install to:  $INSTALL_PREFIX"
echo "Build dir:   $BUILD_DIR"
echo "Jobs:        $JOBS"
echo "=============================================="

# Check for required tools
check_tool() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is required but not installed."
        exit 1
    fi
}

echo "Checking dependencies..."
check_tool cmake
check_tool g++
check_tool gperf

# Install dependencies if apt-get is available (Debian/Ubuntu)
if command -v apt-get &> /dev/null; then
    echo "Installing dependencies via apt-get..."
    sudo apt-get update
    sudo apt-get install -y \
        build-essential \
        cmake \
        gperf \
        libssl-dev \
        zlib1g-dev
fi

# Clean up any previous build
if [ -d "$BUILD_DIR" ]; then
    echo "Cleaning previous build..."
    rm -rf "$BUILD_DIR"
fi

# Clone TDLib
echo "Cloning TDLib $TDLIB_VERSION..."
git clone --depth 1 --branch "$TDLIB_VERSION" https://github.com/tdlib/td.git "$BUILD_DIR"
cd "$BUILD_DIR"

# Create build directory
mkdir build && cd build

# Configure CMake
echo "Configuring CMake..."
cmake -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_INSTALL_PREFIX="$INSTALL_PREFIX" \
      -DTD_ENABLE_LTO=ON \
      ..

# Build TDLib
echo "Building TDLib (this may take a while)..."
cmake --build . --target tdjson_static -j"$JOBS"

# Install
echo "Installing TDLib to $INSTALL_PREFIX..."
sudo cmake --install .

# Verify installation
echo ""
echo "=============================================="
echo "TDLib installed successfully!"
echo "=============================================="
echo "Headers:   $INSTALL_PREFIX/include"
echo "Libraries: $INSTALL_PREFIX/lib"
echo ""
echo "Add these to your environment for building:"
echo ""
echo "  export CGO_CFLAGS=\"-I$INSTALL_PREFIX/include\""
echo "  export CGO_LDFLAGS=\"-L$INSTALL_PREFIX/lib -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdapi -ltdactor -ltddb -ltdsqlite -ltdnet -ltdutils -lstdc++ -lssl -lcrypto -ldl -lz -lm -lpthread\""
echo ""
echo "Or use the Makefile which sets these automatically."
echo "=============================================="

# Cleanup
echo "Cleaning up build directory..."
rm -rf "$BUILD_DIR"

echo "Done!"
