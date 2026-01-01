#!/bin/bash
set -e

APP_NAME="AICoder"
# Read version from build_number if exists, else default
if [ -f "build_number" ]; then
    BUILD_NUM=$(cat build_number)
    BUILD_NUM=$((BUILD_NUM + 1))
    echo $BUILD_NUM > build_number
    VERSION="2.0.0.${BUILD_NUM}"
else
    BUILD_NUM=1
    echo $BUILD_NUM > build_number
    VERSION="2.0.0.1"
fi

# Sync version to frontend
echo "Syncing version $VERSION to frontend..."
sed -i '' "s/const APP_VERSION = \".*\";/const APP_VERSION = \"$VERSION\";/" frontend/src/App.tsx
echo "export const buildNumber = \"$BUILD_NUM\";" > frontend/src/version.ts

IDENTIFIER="com.wails.AICoder"
OUTPUT_DIR="dist"
BIN_DIR="build/bin"

# Check for Go
if ! command -v go &> /dev/null; then
    echo "Error: 'go' command not found in PATH."
    echo "Please ensure Go is installed and available."
    exit 1
fi

echo "Starting build process for version $VERSION..."

# Clean previous build
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
mkdir -p "$BIN_DIR"

# Build Frontend
echo "[1/4] Building Frontend..."
cd frontend
npm install --cache ./.npm_cache
npm run build
cd ..

# Build Binaries
echo "[2/4] Compiling Go Binaries..."

# Build AMD64
echo "  - Building for amd64..."
CGO_ENABLED=1 CGO_LDFLAGS="-framework UniformTypeIdentifiers" GOOS=darwin GOARCH=amd64 go build -tags desktop,production -o "${BIN_DIR}/${APP_NAME}_amd64"

# Build ARM64
echo "  - Building for arm64..."
CGO_ENABLED=1 CGO_LDFLAGS="-framework UniformTypeIdentifiers" GOOS=darwin GOARCH=arm64 go build -tags desktop,production -o "${BIN_DIR}/${APP_NAME}_arm64"

# Generate ICNS
echo "  - Generating .icns file..."
if [ -f "build/appicon.png" ]; then
    ICONSET_DIR="build/appicon.iconset"
    mkdir -p "$ICONSET_DIR"
    
    # Generate standard sizes
    sips -z 16 16     "build/appicon.png" --out "${ICONSET_DIR}/icon_16x16.png" > /dev/null
    sips -z 32 32     "build/appicon.png" --out "${ICONSET_DIR}/icon_16x16@2x.png" > /dev/null
    sips -z 32 32     "build/appicon.png" --out "${ICONSET_DIR}/icon_32x32.png" > /dev/null
    sips -z 64 64     "build/appicon.png" --out "${ICONSET_DIR}/icon_32x32@2x.png" > /dev/null
    sips -z 128 128   "build/appicon.png" --out "${ICONSET_DIR}/icon_128x128.png" > /dev/null
    sips -z 256 256   "build/appicon.png" --out "${ICONSET_DIR}/icon_128x128@2x.png" > /dev/null
    sips -z 256 256   "build/appicon.png" --out "${ICONSET_DIR}/icon_256x256.png" > /dev/null
    sips -z 512 512   "build/appicon.png" --out "${ICONSET_DIR}/icon_256x256@2x.png" > /dev/null
    sips -z 512 512   "build/appicon.png" --out "${ICONSET_DIR}/icon_512x512.png" > /dev/null
    sips -z 1024 1024 "build/appicon.png" --out "${ICONSET_DIR}/icon_512x512@2x.png" > /dev/null
    
    iconutil -c icns "$ICONSET_DIR" -o "build/AppIcon.icns"
    rm -rf "$ICONSET_DIR"
    echo "    Generated build/AppIcon.icns"
fi

# Function to create App Bundle
create_app_bundle() {
    ARCH=$1
    BINARY_NAME="${APP_NAME}_${ARCH}"
    BUNDLE_PATH="${OUTPUT_DIR}/${APP_NAME}_${ARCH}.app"
    
    echo "  - Creating App Bundle for $ARCH..."
    mkdir -p "${BUNDLE_PATH}/Contents/MacOS"
    mkdir -p "${BUNDLE_PATH}/Contents/Resources"
    
    # Copy Binary
    cp "${BIN_DIR}/${BINARY_NAME}" "${BUNDLE_PATH}/Contents/MacOS/${APP_NAME}"
    chmod +x "${BUNDLE_PATH}/Contents/MacOS/${APP_NAME}"
    
    # Create Info.plist (Clean generation)
    cat > "${BUNDLE_PATH}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${IDENTIFIER}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleGetInfoString</key>
    <string>AICoder</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13.0</string>
    <key>NSHighResolutionCapable</key>
    <string>true</string>
    <key>NSHumanReadableCopyright</key>
    <string>Copyright 2025</string>
</dict>
</plist>
EOF
        
    # Copy Icon
    if [ -f "build/AppIcon.icns" ]; then
        cp "build/AppIcon.icns" "${BUNDLE_PATH}/Contents/Resources/AppIcon.icns"
    elif [ -f "build/appicon.png" ]; then
        cp "build/appicon.png" "${BUNDLE_PATH}/Contents/Resources/AppIcon.png"
    fi
    
    touch "${BUNDLE_PATH}"
}

echo "[3/4] Creating App Bundles..."
create_app_bundle amd64
create_app_bundle arm64

# Create Universal Binary
echo "  - Creating Universal Binary..."
UNIVERSAL_BUNDLE="${OUTPUT_DIR}/${APP_NAME}.app"
mkdir -p "${UNIVERSAL_BUNDLE}/Contents/MacOS"
mkdir -p "${UNIVERSAL_BUNDLE}/Contents/Resources"
lipo -create "${BIN_DIR}/${APP_NAME}_amd64" "${BIN_DIR}/${APP_NAME}_arm64" -output "${UNIVERSAL_BUNDLE}/Contents/MacOS/${APP_NAME}"
cp "${OUTPUT_DIR}/${APP_NAME}_arm64.app/Contents/Info.plist" "${UNIVERSAL_BUNDLE}/Contents/Info.plist"
cp -R "${OUTPUT_DIR}/${APP_NAME}_arm64.app/Contents/Resources/" "${UNIVERSAL_BUNDLE}/Contents/Resources/"
touch "${UNIVERSAL_BUNDLE}"

# Function to create PKG
create_pkg() {
    ARCH=$1
    if [ "$ARCH" == "universal" ]; then
        BUNDLE_PATH="${OUTPUT_DIR}/${APP_NAME}.app"
        PKG_NAME="${APP_NAME}-Universal.pkg"
    else
        BUNDLE_PATH="${OUTPUT_DIR}/${APP_NAME}_${ARCH}.app"
        PKG_NAME="${APP_NAME}-${ARCH}.pkg"
    fi
    
    # Temporary root for pkgbuild
    TEMP_ROOT="build/pkg_root_${ARCH}"
    rm -rf "$TEMP_ROOT"
    mkdir -p "$TEMP_ROOT/Applications"
    cp -R "$BUNDLE_PATH" "$TEMP_ROOT/Applications/"
    
    SCRIPTS_DIR="build/scripts_x64"
    if [ "$ARCH" == "arm64" ] || [ "$ARCH" == "universal" ]; then
        SCRIPTS_DIR="build/scripts_arm64"
    fi
    
    echo "  - Creating PKG for $ARCH using scripts from $SCRIPTS_DIR..."
    
    # Ensure scripts are executable
    chmod +x "$SCRIPTS_DIR/preinstall"
    chmod +x "$SCRIPTS_DIR/postinstall"
    
    pkgbuild --root "$TEMP_ROOT" \
             --identifier "$IDENTIFIER" \
             --version "$VERSION" \
             --install-location "/" \
             --scripts "$SCRIPTS_DIR" \
             "${OUTPUT_DIR}/${PKG_NAME}"
    
    rm -rf "$TEMP_ROOT"
}

echo "[4/4] Creating Packages..."
create_pkg amd64
create_pkg arm64
create_pkg universal

echo "Build Complete!"
echo "App Bundles and Packages are in $OUTPUT_DIR"
