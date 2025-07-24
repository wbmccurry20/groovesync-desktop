#!/bin/bash

# Exit on any error
set -e

# Optional arg for platform (e.g., darwin/arm64, windows/amd64, linux/amd64)
PLATFORM="${1:-darwin/arm64}"

# Versions (pin for stability)
YTDLP_VERSION="2025.07.21"
FFMPEG_VERSION="7.0.2"

# Clean previous artifacts
echo "Cleaning up previous build artifacts..."
rm -rf dist build/bin GrooveSync.app bin

# Ensure Go is installed (1.22+)
echo "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing latest Go..."
    if [[ "$(uname)" == "Darwin" ]]; then
        brew install go || (echo "Install Homebrew first: /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""; exit 1)
    elif [[ "$(uname)" == "Linux" ]]; then
        sudo apt update && sudo apt install -y golang-go
    else
        echo "Unsupported OS. Install Go manually from https://go.dev/dl/"
        exit 1
    fi
fi
export PATH=$PATH:$(go env GOPATH)/bin:$HOME/go/bin

# Ensure Wails is installed (v2.10.1+)
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

# Clean existing wailsjs
echo "Cleaning existing wailsjs directories..."
rm -rf frontend/wailsjs

# Generate Wails bindings
echo "Generating Wails bindings..."
wails generate module
if [ ! -d "frontend/wailsjs" ]; then
    echo "Error: Wails bindings generation failed."
    exit 1
fi

# Build frontend
echo "Building frontend..."
cd frontend
npm run build
cd ..

# Create bin dir for staging
mkdir -p bin

# Download yt-dlp (platform-specific)
echo "Downloading yt-dlp v$YTDLP_VERSION for $PLATFORM..."
case "$PLATFORM" in
    darwin/arm64|darwin/amd64)
        curl -L -o "bin/yt-dlp" "https://github.com/yt-dlp/yt-dlp/releases/download/$YTDLP_VERSION/yt-dlp_macos"
        ;;
    windows/amd64)
        curl -L -o "bin/yt-dlp.exe" "https://github.com/yt-dlp/yt-dlp/releases/download/$YTDLP_VERSION/yt-dlp.exe"
        ;;
    linux/amd64)
        curl -L -o "bin/yt-dlp" "https://github.com/yt-dlp/yt-dlp/releases/download/$YTDLP_VERSION/yt-dlp_linux"
        ;;
    *)
        echo "Unsupported platform: $PLATFORM"
        exit 1
        ;;
esac
chmod +x bin/yt-dlp*

# Download ffmpeg and ffprobe (separate for evermeet)
echo "Downloading ffmpeg v$FFMPEG_VERSION for $PLATFORM..."
case "$PLATFORM" in
    darwin/arm64|darwin/amd64)
        curl -L -o "bin/ffmpeg.zip" "https://evermeet.cx/ffmpeg/ffmpeg-$FFMPEG_VERSION.zip"
        unzip -o "bin/ffmpeg.zip" -d "bin/"
        rm "bin/ffmpeg.zip"
        
        echo "Downloading ffprobe v$FFMPEG_VERSION..."
        curl -L -o "bin/ffprobe.zip" "https://evermeet.cx/ffmpeg/ffprobe-$FFMPEG_VERSION.zip"
        unzip -o "bin/ffprobe.zip" -d "bin/"
        rm "bin/ffprobe.zip"
        ;;
    windows/amd64)
        curl -L -o "bin/ffmpeg.zip" "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
        unzip -o "bin/ffmpeg.zip" "ffmpeg-$FFMPEG_VERSION-essentials_build/bin/ffmpeg.exe" "ffmpeg-$FFMPEG_VERSION-essentials_build/bin/ffprobe.exe" -d "bin/"
        mv bin/ffmpeg-$FFMPEG_VERSION-essentials_build/bin/ffmpeg.exe bin/ffmpeg.exe
        mv bin/ffmpeg-$FFMPEG_VERSION-essentials_build/bin/ffprobe.exe bin/ffprobe.exe
        rm -rf bin/ffmpeg-$FFMPEG_VERSION-essentials_build bin/ffmpeg.zip
        ;;
    linux/amd64)
        curl -L -o "bin/ffmpeg.tar.xz" "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"
        tar -xJf "bin/ffmpeg.tar.xz" -C "bin/" --strip-components=1 "ffmpeg-*/ffmpeg" "ffmpeg-*/ffprobe"
        rm "bin/ffmpeg.tar.xz"
        ;;
    *)
        echo "Unsupported platform for ffmpeg: $PLATFORM"
        exit 1
        ;;
esac
chmod +x bin/ffmpeg* bin/ffprobe*

# Build the app
echo "Building for $PLATFORM..."
wails build -platform "$PLATFORM" -o GrooveSync -clean

# Copy binaries to bundle
mkdir -p dist
if [[ "$PLATFORM" == *"darwin"* ]]; then
    cp -r build/bin/GrooveSync.app dist/
    mkdir -p dist/GrooveSync.app/Contents/MacOS/bin
    cp bin/yt-dlp dist/GrooveSync.app/Contents/MacOS/bin/yt-dlp
    cp bin/ffmpeg dist/GrooveSync.app/Contents/MacOS/bin/ffmpeg
    cp bin/ffprobe dist/GrooveSync.app/Contents/MacOS/bin/ffprobe
    chmod +x dist/GrooveSync.app/Contents/MacOS/bin/*

    # Info.plist (enhanced for stability)
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

    # Fix security (remove quarantine for double-click launch)
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
    cp bin/yt-dlp dist/bin/yt-dlp
    cp bin/ffmpeg dist/bin/ffmpeg
    cp bin/ffprobe dist/bin/ffprobe
    chmod +x dist/*
fi

# Clean up staging bin
rm -rf bin

echo "Build complete! Artifacts in dist/"
echo "To run on macOS: open dist/GrooveSync.app (double-click should work; if warning, right-click > Open first time)"