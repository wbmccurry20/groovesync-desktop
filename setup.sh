#!/bin/bash

# Step 1: Ensure macOS dependencies (only required for macOS security)
echo "Ensuring macOS dependencies..."
if ! command -v xattr &> /dev/null; then
    echo "Error: xattr is required but not installed. Please ensure macOS developer tools are installed."
    exit 1
fi

# Step 2: Prepare the .app bundle (prebuilt binary)
echo "Creating .app bundle..."

mkdir -p GrooveSync.app/Contents/{MacOS,Resources}

# Move the prebuilt binary to the app bundle
cp GrooveSync GrooveSync.app/Contents/MacOS/

# Copy dependencies (yt-dlp binaries and assets)
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

# Step 3: Fix macOS security attributes
echo "Fixing macOS security settings..."
xattr -c GrooveSync.app
chmod +x GrooveSync.app/Contents/MacOS/GrooveSync

# Step 4: Inform the user
echo "Packaging complete!"
echo "GrooveSync.app is ready to test."

echo "To run the app:"
echo "  1. Double-click GrooveSync.app."
echo "  2. If macOS blocks the app, go to System Preferences > Security & Privacy and allow it."