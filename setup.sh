#!/bin/bash

# Exit on any error
set -e

# Optional arg for platform (e.g., darwin/arm64, windows/amd64, linux/amd64)
PLATFORM="${1:-darwin/arm64}"

# Clean previous artifacts
echo "Cleaning up previous build artifacts..."
rm -rf dist build/bin GrooveSync.app

# Ensure Go is installed (version 1.22+ recommended for latest Wails)
echo "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing latest Go..."
    if [[ "$(uname)" == "Darwin" ]]; then
        brew install go || (echo "Install Homebrew first: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""; exit 1)
    elif [[ "$(uname)" == "Linux" ]]; then
        sudo apt update && sudo apt install -y golang-go
    else
        echo "Unsupported OS. Install Go manually from https://go.dev/dl/"
        exit 1
    fi
fi
export PATH=$PATH:$(go env GOPATH)/bin:$HOME/go/bin

# Ensure Wails is installed (latest v2)
echo "Checking for Wails installation..."
if ! command -v wails &> /dev/null; then
    echo "Wails not installed. Installing latest..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
fi

# Ensure Node/npm
echo "Checking for Node.js and npm..."
if ! command -v npm &> /dev/null; then
    echo "Installing Node.js and npm..."
    if [[ "$(uname)" == "Darwin" ]]; then
        brew install node
    elif [[ "$(uname)" == "Linux" ]]; then
        sudo apt update && sudo apt install -y nodejs npm
    else
        echo "Unsupported OS. Install manually."
        exit 1
    fi
fi

# macOS-specific: Ensure Xcode command line tools
if [[ "$PLATFORM" == *"darwin"* ]]; then
    if ! xcode-select -p &> /dev/null; then
        echo "Installing Xcode command line tools..."
        xcode-select --install
    fi
fi

# Install frontend deps
echo "Installing frontend dependencies..."
cd frontend
npm install
cd ..

# Prepare Go modules
echo "Preparing Go modules..."
go mod tidy

# Clean any existing wailsjs to avoid duplicates/nesting
echo "Cleaning existing wailsjs directories..."
rm -rf frontend/wailsjs

# Generate Wails bindings with debug output
echo "Generating Wails bindings..."
wails generate module
if [ ! -d "frontend/wailsjs" ] || [ ! -f "frontend/wailsjs/go/main/App.js" ] || [ ! -f "frontend/wailsjs/runtime/runtime.js" ]; then
    echo "Error: Wails bindings generation failed or incomplete. Check wails.json and Go code."
    exit 1
fi
echo "Bindings generated successfully. Contents of frontend/wailsjs:"
ls -R frontend/wailsjs

# Build frontend
echo "Building frontend..."
cd frontend
npm run build
cd ..

# Download yt-dlp based on platform
mkdir -p bin
function download_ytdlp() {
    local binary_name=$1
    local url="https://github.com/yt-dlp/yt-dlp/releases/latest/download/$binary_name"
    if [[ ! -f "bin/$binary_name" ]]; then
        echo "Downloading yt-dlp: $binary_name..."
        curl -L -o "bin/$binary_name" "$url"
        chmod +x "bin/$binary_name"
    fi
}

case "$PLATFORM" in
    darwin/arm64|darwin/amd64)
        download_ytdlp "yt-dlp_macos"
        ;;
    windows/amd64)
        download_ytdlp "yt-dlp.exe"
        ;;
    linux/amd64)
        download_ytdlp "yt-dlp_linux"
        ;;
    *)
        echo "Unsupported platform: $PLATFORM"
        exit 1
        ;;
esac

# Download ffmpeg/ffprobe (platform-specific; fallback to system install if fail)
function download_ffmpeg() {
    if [[ "$PLATFORM" == *"darwin"* ]]; then
        if [[ ! -f "bin/ffmpeg" ]] || [[ ! -f "bin/ffprobe" ]]; then
            echo "Downloading ffmpeg for macOS..."
            curl -L -o "bin/ffmpeg.zip" https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip
            unzip -o "bin/ffmpeg.zip" -d "bin/" && mv bin/ffmpeg bin/ffmpeg && chmod +x bin/ffmpeg
            curl -L -o "bin/ffprobe.zip" https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip
            unzip -o "bin/ffprobe.zip" -d "bin/" && mv bin/ffprobe bin/ffprobe && chmod +x bin/ffprobe
            rm bin/*.zip
        fi
    elif [[ "$PLATFORM" == *"linux"* ]]; then
        if [[ ! -f "bin/ffmpeg" ]] || [[ ! -f "bin/ffprobe" ]]; then
            echo "Downloading ffmpeg for Linux..."
            curl -L -o "bin/ffmpeg.tar.xz" https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
            tar -xJf "bin/ffmpeg.tar.xz" -C "bin/" --strip-components=1 "ffmpeg-*/ffmpeg" "ffmpeg-*/ffprobe"
            chmod +x bin/ffmpeg bin/ffprobe
            rm bin/*.tar.xz
        fi
    elif [[ "$PLATFORM" == *"windows"* ]]; then
        if [[ ! -f "bin/ffmpeg.exe" ]] || [[ ! -f "bin/ffprobe.exe" ]]; then
            echo "Downloading ffmpeg for Windows..."
            curl -L -o "bin/ffmpeg.zip" https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip
            unzip -o "bin/ffmpeg.zip" -d "bin/" "ffmpeg-*/bin/ffmpeg.exe" "ffmpeg-*/bin/ffprobe.exe"
            mv bin/ffmpeg-*/bin/ffmpeg.exe bin/ffmpeg.exe
            mv bin/ffmpeg-*/bin/ffprobe.exe bin/ffprobe.exe
            rm -rf bin/ffmpeg-* bin/*.zip
        fi
    fi
}
download_ffmpeg

# Build the app
echo "Building for $PLATFORM..."
wails build -platform "$PLATFORM" -o GrooveSync -clean

# Copy binaries to build dir (adjust for platform bundle structure)
mkdir -p dist
if [[ "$PLATFORM" == *"darwin"* ]]; then
    cp -r build/bin/GrooveSync.app dist/
    mkdir -p dist/GrooveSync.app/Contents/MacOS/bin
    cp bin/yt-dlp_macos dist/GrooveSync.app/Contents/MacOS/bin/yt-dlp
    cp bin/ffmpeg dist/GrooveSync.app/Contents/MacOS/bin/ffmpeg
    cp bin/ffprobe dist/GrooveSync.app/Contents/MacOS/bin/ffprobe
    chmod +x dist/GrooveSync.app/Contents/MacOS/bin/*
    
    # Info.plist (enhanced with more keys for stability)
    cat > dist/GrooveSync.app/Contents/Info.plist <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>GrooveSync</string>
    <key>CFBundleExecutable</key>
    <string>GrooveSync</string>
    <key>CFBundleIdentifier</key>
    <string>com.groovesync.app</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
</dict>
</plist>
EOL

    # Fix security
    xattr -cr dist/GrooveSync.app
    chmod +x dist/GrooveSync.app/Contents/MacOS/GrooveSync
elif [[ "$PLATFORM" == *"windows"* ]]; then
    cp build/bin/GrooveSync.exe dist/
    mkdir -p dist/bin
    cp bin/yt-dlp.exe dist/bin/yt-dlp.exe
    cp bin/ffmpeg.exe dist/bin/ffmpeg.exe
    cp bin/ffprobe.exe dist/bin/ffprobe.exe
elif [[ "$PLATFORM" == *"linux"* ]]; then
    cp build/bin/GrooveSync dist/
    mkdir -p dist/bin
    cp bin/yt-dlp_linux dist/bin/yt-dlp
    cp bin/ffmpeg dist/bin/ffmpeg
    cp bin/ffprobe dist/bin/ffprobe
    chmod +x dist/bin/*
fi

echo "Build complete! Artifacts in dist/"
echo "To run on macOS: open dist/GrooveSync.app"
echo "If security warning: right-click > Open, or xattr -d com.apple.quarantine dist/GrooveSync.app"

# Auto-test launch if mac
if [[ "$PLATFORM" == *"darwin"* ]]; then
    echo "Testing launch..."
    open dist/GrooveSync.app || echo "Launch failed; check logs."
fi