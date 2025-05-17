#!/bin/bash
# Step 1: Check for Go installation
echo "Checking for Go installation..."
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Installing Go..."

# Define Go version
GO_VERSION="1.21.0"

# Download and install Go
if [[ "$(uname)" == "Darwin" ]]; then
    # macOS
    curl -OL "https://go.dev/dl/go${GO_VERSION}.darwin-arm64.pkg"
    sudo installer -pkg "go${GO_VERSION}.darwin-arm64.pkg" -target /
    rm "go${GO_VERSION}.darwin-arm64.pkg"
elif [[ "$(uname)" == "Linux" ]]; then
    # Linux
    curl -OL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm "go${GO_VERSION}.linux-amd64.tar.gz"
    export PATH=$PATH:/usr/local/go/bin
else
    echo "Unsupported operating system. Please install Go manually."
    exit 1
fi

fi
# Verify Go installation
if ! command -v go &> /dev/null; then
    echo "Failed to install Go. Please install it manually and try again."
    exit 1
fi
echo "Go installed successfully."
# Step 2: Build the GrooveSync binary
echo "Building GrooveSync binary..."
GOOS=darwin GOARCH=arm64 go build -o GrooveSync ./cmd/main.go
if [[ $? -ne 0 ]]; then
    echo "Error: Failed to build GrooveSync binary."
    exit 1
fi
# Step 3: Create the .app bundle structure
echo "Creating .app bundle..."
mkdir -p GrooveSync.app/Contents/{MacOS,Resources}
cp GrooveSync GrooveSync.app/Contents/MacOS/
# Step 4: Copy yt-dlp binary into the app bundle
echo "Copying yt-dlp binary..."
YT_DLP_BINARY="bin/yt-dlp_macos"
if [[ ! -f $YT_DLP_BINARY ]]; then
    echo "yt-dlp_macos binary not found at $YT_DLP_BINARY. Downloading..."
    mkdir -p bin
    curl -L -o "$YT_DLP_BINARY" https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_macos
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to download yt-dlp_macos."
        exit 1
    fi
    chmod +x "$YT_DLP_BINARY"
fi
if [[ ! -f "$YT_DLP_BINARY" ]]; then
    echo "Error: yt-dlp_macos binary still not found at $YT_DLP_BINARY."
    exit 1
fi
mkdir -p GrooveSync.app/Contents/MacOS/bin
cp $YT_DLP_BINARY GrooveSync.app/Contents/MacOS/bin/yt-dlp_macos
# Step 5: Copy ffmpeg and ffprobe into the app bundle
echo "Copying ffmpeg and ffprobe..."
FFMPEG_BINARY="bin/ffmpeg"
FFPROBE_BINARY="bin/ffprobe"
# Download ffmpeg and ffprobe if not present (macOS static builds)
if [[ ! -f $FFMPEG_BINARY ]] || [[ ! -f $FFPROBE_BINARY ]]; then
    echo "ffmpeg/ffprobe not found in bin/. Downloading..."
    mkdir -p bin
    # Download ffmpeg static build for macOS (includes ffprobe)
    curl -L -o "bin/ffmpeg.zip" https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to download ffmpeg."
        exit 1
    fi
    unzip -o "bin/ffmpeg.zip" -d "bin/"
    mv "bin/ffmpeg" "bin/ffmpeg_temp"
    mv "bin/ffmpeg_temp" "bin/ffmpeg"
    chmod +x "bin/ffmpeg"
    curl -L -o "bin/ffprobe.zip" https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to download ffprobe."
        exit 1
    fi
    unzip -o "bin/ffprobe.zip" -d "bin/"
    mv "bin/ffprobe" "bin/ffprobe_temp"
    mv "bin/ffprobe_temp" "bin/ffprobe"
    chmod +x "bin/ffprobe"
fi
if [[ ! -f "$FFMPEG_BINARY" ]] || [[ ! -f "$FFPROBE_BINARY" ]]; then
    echo "Error: ffmpeg or ffprobe binary still not found in bin/."
    exit 1
fi
cp $FFMPEG_BINARY GrooveSync.app/Contents/MacOS/bin/ffmpeg
cp $FFPROBE_BINARY GrooveSync.app/Contents/MacOS/bin/ffprobe
chmod +x GrooveSync.app/Contents/MacOS/bin/ffmpeg
chmod +x GrooveSync.app/Contents/MacOS/bin/ffprobe
# Step 6: Generate Info.plist
echo "Generating Info.plist..."
cat > GrooveSync.app/Contents/Info.plist <<EOL
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
# Step 7: Fix macOS security attributes
echo "Fixing macOS security settings..."
xattr -cr GrooveSync.app
chmod +x GrooveSync.app/Contents/MacOS/GrooveSync
chmod +x GrooveSync.app/Contents/MacOS/bin/yt-dlp_macos
# Step 8: Verify the app bundle
echo "Verifying app bundle structure..."
if [[ ! -f GrooveSync.app/Contents/MacOS/GrooveSync ]]; then
    echo "Error: GrooveSync binary is missing in the app bundle."
    exit 1
fi
if [[ ! -f GrooveSync.app/Contents/MacOS/bin/yt-dlp_macos ]]; then
    echo "Error: yt-dlp_macos binary is missing in the app bundle at Contents/MacOS/bin/yt-dlp_macos."
    exit 1
fi
if [[ ! -f GrooveSync.app/Contents/MacOS/bin/ffmpeg ]]; then
    echo "Error: ffmpeg binary is missing in the app bundle at Contents/MacOS/bin/ffmpeg."
    exit 1
fi
if [[ ! -f GrooveSync.app/Contents/MacOS/bin/ffprobe ]]; then
    echo "Error: ffprobe binary is missing in the app bundle at Contents/MacOS/bin/ffprobe."
    exit 1
fi
if [[ ! -f GrooveSync.app/Contents/Info.plist ]]; then
    echo "Error: Info.plist is missing in the app bundle."
    exit 1
fi
# Step 9: Inform the user
echo "Packaging complete!"
echo "GrooveSync.app is ready to test."
echo "To run the app:"
echo "  1. Double-click GrooveSync.app."
echo "  2. If macOS blocks the app, go to System Preferences > Security & Privacy and allow it."