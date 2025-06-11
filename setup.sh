#!/bin/bash

# Exit on any error to ensure script fails if a command fails
set -e

# Ensure the script is idempotent by cleaning up previous artifacts
echo "Cleaning up previous build artifacts..."
rm -rf dist GrooveSync.app build/bin

# Ensure dependencies are installed
echo "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing Go..."
    GO_VERSION="1.21.0"
    if [[ "$(uname)" == "Darwin" ]]; then
        curl -OL "https://go.dev/dl/go${GO_VERSION}.darwin-arm64.pkg"
        sudo installer -pkg "go${GO_VERSION}.darwin-arm64.pkg" -target /
        rm "go${GO_VERSION}.darwin-arm64.pkg"
    elif [[ "$(uname)" == "Linux" ]]; then
        curl -OL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
        sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
        rm "go${GO_VERSION}.linux-amd64.tar.gz"
        export PATH=$PATH:/usr/local/go/bin
    else
        echo "Unsupported operating system. Please install Go manually."
        exit 1
    fi
fi

echo "Checking for Wails installation..."
if ! command -v wails &> /dev/null; then
    echo "Wails is not installed. Installing Wails..."
    go install github.com/wailsapp/wails/v2/cmd/wails@v2.10.1
fi

echo "Checking for Node.js and npm installation..."
if ! command -v npm &> /dev/null; then
    echo "Node.js and npm are not installed. Installing Node.js and npm..."
    if [[ "$(uname)" == "Darwin" ]]; then
        brew install node
    elif [[ "$(uname)" == "Linux" ]]; then
        sudo apt update
        sudo apt install -y nodejs npm
    else
        echo "Unsupported operating system. Please install Node.js and npm manually."
        exit 1
    fi
fi

echo "Building frontend..."
cd frontend
if [[ ! -d "node_modules" ]]; then
    npm install
fi
npm run build
cd ..

echo "Creating distribution directory..."
mkdir -p dist

# Function to download yt-dlp binaries for all platforms
download_ytdlp() {
    local platform=$1
    local binary_name=$2
    local url="https://github.com/yt-dlp/yt-dlp/releases/latest/download/$binary_name"
    if [[ ! -f "bin/$binary_name" ]]; then
        echo "Downloading yt-dlp for $platform..."
        curl -L -o "bin/$binary_name" "$url"
        if [[ $? -ne 0 ]]; then
            echo "Error: Failed to download $binary_name."
            exit 1
        fi
        chmod +x "bin/$binary_name"
    else
        echo "yt-dlp for $platform already exists in bin/$binary_name. Skipping download..."
    fi
}

# Download yt-dlp binaries for all platforms
mkdir -p bin
download_ytdlp "macOS" "yt-dlp_macos"
download_ytdlp "Linux" "yt-dlp_linux"
download_ytdlp "Windows" "yt-dlp.exe"

# Download ffmpeg and ffprobe (shared across platforms for simplicity)
FFMPEG_BINARY="bin/ffmpeg"
FFPROBE_BINARY="bin/ffprobe"
if [[ ! -f $FFMPEG_BINARY ]] || [[ ! -f $FFPROBE_BINARY ]]; then
    echo "ffmpeg/ffprobe not found in bin/. Downloading..."
    mkdir -p bin
    curl -L -o "bin/ffmpeg.zip" https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip
    if [[ $? -ne 0 ]]; then
        echo "Warning: Failed to download ffmpeg from evermeet.cx. Falling back to Homebrew on macOS..."
        if [[ "$(uname)" == "Darwin" ]]; then
            brew install ffmpeg
            cp $(which ffmpeg) "bin/ffmpeg"
            cp $(which ffprobe) "bin/ffprobe"
            chmod +x "bin/ffmpeg" "bin/ffprobe"
        else
            echo "Error: Please install ffmpeg and ffprobe manually for your platform."
            exit 1
        fi
    else
        unzip -o "bin/ffmpeg.zip" -d "bin/"
        mv "bin/ffmpeg" "bin/ffmpeg_temp"
        mv "bin/ffmpeg_temp" "bin/ffmpeg"
        chmod +x "bin/ffmpeg"
        curl -L -o "bin/ffprobe.zip" https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip
        if [[ $? -ne 0 ]]; then
            echo "Warning: Failed to download ffprobe from evermeet.cx. Falling back to Homebrew on macOS..."
            if [[ "$(uname)" == "Darwin" ]]; then
                brew install ffmpeg
                cp $(which ffmpeg) "bin/ffmpeg"
                cp $(which ffprobe) "bin/ffprobe"
                chmod +x "bin/ffmpeg" "bin/ffprobe"
            else
                echo "Error: Please install ffmpeg and ffprobe manually for your platform."
                exit 1
            fi
        else
            unzip -o "bin/ffprobe.zip" -d "bin/"
            mv "bin/ffprobe" "bin/ffprobe_temp"
            mv "bin/ffprobe_temp" "bin/ffprobe"
            chmod +x "bin/ffprobe"
        fi
    fi
fi

# Build for macOS
echo "Building for macOS..."
wails build -platform darwin/arm64 -o GrooveSync
if [[ ! -f "build/bin/GrooveSync.app/Contents/MacOS/GrooveSync" ]]; then
    echo "Error: macOS build failed to produce expected binary at build/bin/GrooveSync.app/Contents/MacOS/GrooveSync."
    exit 1
fi
cp -r build/bin/GrooveSync.app dist/
mkdir -p dist/GrooveSync.app/Contents/MacOS/bin
cp bin/yt-dlp_macos dist/GrooveSync.app/Contents/MacOS/bin/yt-dlp_macos
cp bin/ffmpeg dist/GrooveSync.app/Contents/MacOS/bin/ffmpeg
cp bin/ffprobe dist/GrooveSync.app/Contents/MacOS/bin/ffprobe
chmod +x dist/GrooveSync.app/Contents/MacOS/bin/yt-dlp_macos
chmod +x dist/GrooveSync.app/Contents/MacOS/bin/ffmpeg
chmod +x dist/GrooveSync.app/Contents/MacOS/bin/ffprobe

echo "Generating Info.plist for macOS..."
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
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOL

echo "Fixing macOS security settings..."
xattr -cr dist/GrooveSync.app
chmod +x dist/GrooveSync.app/Contents/MacOS/GrooveSync

echo "Packaging complete!"
echo "Artifacts are in the dist/ directory:"
echo "- macOS: dist/GrooveSync.app"
echo "To test on macOS, double-click dist/GrooveSync.app."
echo "Linux and Windows builds skipped due to cross-compilation issues on macOS. Build on a Linux or Windows machine, or set up cross-compilation."