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

# Step 3: Create the .app bundle
echo "Creating .app bundle..."
mkdir -p GrooveSync.app/Contents/{MacOS,Resources}
cp GrooveSync GrooveSync.app/Contents/MacOS/
cp -R bin/ GrooveSync.app/Contents/Resources/
cp -R assets/ GrooveSync.app/Contents/Resources/

# Generate Info.plist
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

# Step 4: Fix macOS security attributes
echo "Fixing macOS security settings..."
xattr -c GrooveSync.app
chmod +x GrooveSync.app/Contents/MacOS/GrooveSync

# Step 5: Inform the user
echo "Packaging complete!"
echo "GrooveSync.app is ready to test."

echo "To run the app:"
echo "  1. Double-click GrooveSync.app."
echo "  2. If macOS blocks the app, go to System Preferences > Security & Privacy and allow it."