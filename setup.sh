#!/bin/bash

# Step 1: Check and install Go if not present
check_go_installation() {
    echo "Checking if Go is installed..."
    if ! command -v go &> /dev/null; then
        echo "Go is not installed. Installing Go..."
        
        # Set the version of Go you want to install
        GO_VERSION="1.23.0"

        # Detect architecture
        ARCH=$(uname -m)
        case $ARCH in
            "x86_64")
                ARCH="amd64"
                ;;
            "arm64")
                ARCH="arm64"
                ;;
            *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
        esac

        # Detect OS
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')

        # Download Go installer
        GO_TAR="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
        echo "Downloading Go (${GO_TAR})..."
        curl -LO "https://go.dev/dl/${GO_TAR}"

        # Extract Go and move to /usr/local
        echo "Installing Go..."
        sudo tar -C /usr/local -xzf "${GO_TAR}"
        rm "${GO_TAR}"

        # Add Go to PATH
        export PATH=$PATH:/usr/local/go/bin
        echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bash_profile
        echo "Go installed successfully."
    else
        echo "Go is already installed. Skipping installation."
    fi
}

# Step 2: Build the Go binary
build_binary() {
    echo "Building GrooveSync binary..."
    GOOS=darwin GOARCH=arm64 go build -o GrooveSync ./cmd/main.go
    if [ $? -ne 0 ]; then
        echo "Failed to build the binary. Please check your Go installation."
        exit 1
    fi
}

# Step 3: Create the .app bundle structure
create_app_bundle() {
    echo "Creating .app bundle..."
    mkdir -p GrooveSync.app/Contents/{MacOS,Resources}

    # Move the binary to the app bundle
    mv GrooveSync GrooveSync.app/Contents/MacOS/

    # Copy resources (yt-dlp binaries and assets)
    echo "Copying dependencies..."
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
}

# Step 4: Main script execution
main() {
    check_go_installation
    build_binary
    create_app_bundle

    echo "Packaging complete!"
    echo "GrooveSync.app is ready to test."
    echo "To run the app:"
    echo "  1. Double-click GrooveSync.app."
    echo "  2. If macOS blocks the app, go to System Preferences > Security & Privacy and allow it."
}

main