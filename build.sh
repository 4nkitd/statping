#!/bin/bash
set -e

# Define output directory
BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

echo "Cleaning up old builds..."
rm -f "$BUILD_DIR"/*

VERSION=$(git describe --tags --always)
echo "Building version: $VERSION"

# Function to build and compress
build_and_archive() {
    OS=$1
    ARCH=$2
    SUFFIX=$3
    CC_CMD=$4
    
    BIN_NAME="statping"
    if [ "$OS" == "windows" ]; then
        BIN_NAME="statping.exe"
    fi

    echo "Building for $OS/$ARCH..."
    
    # Check for CGO requirements
    if [ "$OS" == "linux" ] && [ "$(go env GOOS)" != "linux" ]; then
        if [ -z "$CC_CMD" ]; then
             echo "WARNING: Cross-compiling for Linux requries a cross-compiler (CC) with CGO enabled."
             echo "Skipping Linux/$ARCH build due to missing CC configuration. Please install a cross-compiler."
             return
        fi
        export CC=$CC_CMD
    fi

    export GOOS=$OS
    export GOARCH=$ARCH
    export CGO_ENABLED=1

    OUTPUT_BIN="${BUILD_DIR}/statping-${SUFFIX}/${BIN_NAME}"
    mkdir -p "$(dirname "$OUTPUT_BIN")"

    if go build -ldflags="-s -w -X main.VERSION=$VERSION" -o "$OUTPUT_BIN" ./cmd/statping; then
        echo "Build success: $OUTPUT_BIN"
        
        # Archive
        cd "$BUILD_DIR"
        ARCHIVE_NAME="statping-${SUFFIX}.tar.gz"
        tar -czvf "$ARCHIVE_NAME" -C "statping-${SUFFIX}" "$BIN_NAME"
        echo "Created archive: $ARCHIVE_NAME"
        cd ..
        
        # Cleanup temp dir
        rm -rf "${BUILD_DIR}/statping-${SUFFIX}"
    else
        echo "Build failed for $OS/$ARCH"
        exit 1
    fi
}

# --- Builds ---

# macOS AMD64
build_and_archive "darwin" "amd64" "darwin-amd64" ""

# macOS ARM64
build_and_archive "darwin" "arm64" "darwin-arm64" ""

# Linux AMD64
# Requires: brew install protobuf
# Requires: brew install filosottile/musl-cross/musl-cross
CC_LINUX_AMD64="x86_64-linux-musl-gcc" 
build_and_archive "linux" "amd64" "linux-amd64" "$CC_LINUX_AMD64"


# Linux ARM64
# Requires: brew install messense/macos-cross-toolchains/aarch64-unknown-linux-gnu
CC_LINUX_ARM64="aarch64-unknown-linux-gnu-gcc"
build_and_archive "linux" "arm64" "linux-arm64" "$CC_LINUX_ARM64"

echo "Artifacts are in $BUILD_DIR/"

# --- Update Homebrew Formula ---
echo "Updating Homebrew formula checksums..."

get_sha256() {
    FILE="build/$1"
    if [ -f "$FILE" ]; then
        shasum -a 256 "$FILE" | awk '{print $1}'
    else
        echo "DELETE_ME" # Placeholder for missing files
    fi
}

DARWIN_AMD64_SHA=$(get_sha256 "statping-darwin-amd64.tar.gz")
DARWIN_ARM64_SHA=$(get_sha256 "statping-darwin-arm64.tar.gz")
LINUX_AMD64_SHA=$(get_sha256 "statping-linux-amd64.tar.gz")
LINUX_ARM64_SHA=$(get_sha256 "statping-linux-arm64.tar.gz")

FORMULA_FILE="Formula/statping.rb"

# Function to replace placeholder or existing SHA
update_sha() {
    PLACEHOLDER=$1
    SHA=$2
    if [ "$SHA" != "DELETE_ME" ]; then
        # Replace specific placeholder
        sed -i "" "s/$PLACEHOLDER/$SHA/" "$FORMULA_FILE"
        # Also try replacing existing sha lines if placeholders are gone (simple regex approach)
        # This part assumes structure: sha256 "OLD_SHA"
        # It's safer to rely on placeholders for this script or use more complex regex if needed.
    fi
}

update_sha "PLACEHOLDER_DARWIN_AMD64_SHA256" "$DARWIN_AMD64_SHA"
update_sha "PLACEHOLDER_DARWIN_ARM64_SHA256" "$DARWIN_ARM64_SHA"
update_sha "PLACEHOLDER_LINUX_AMD64_SHA256" "$LINUX_AMD64_SHA"
update_sha "PLACEHOLDER_LINUX_ARM64_SHA256" "$LINUX_ARM64_SHA"

echo "Formula updated."
