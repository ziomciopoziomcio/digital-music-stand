#!/bin/bash

GITHUB_OWNER="ziomciopoziomcio"
GITHUB_REPO="digital-music-stand"

echo "=== Setting up Digital Music Stand environment ==="

echo "[1/3] Installing system dependencies..."
sudo apt-get update
sudo apt-get install -y \
    curl \
    jq \
    network-manager \
    alsa-utils \
    brightnessctl \
    x11-xserver-utils \
    matchbox-keyboard

echo "[2/3] Configuring user permissions..."
sudo usermod -aG video $USER
sudo usermod -aG audio $USER
sudo usermod -aG netdev $USER

ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    TARGET_ASSET="linux-amd64"
elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    TARGET_ASSET="linux-arm64"
elif [ "$ARCH" = "armv7l" ]; then
    TARGET_ASSET="linux-arm"
else
    echo "Unknown architecture: $ARCH. Aborting download."
    exit 1
fi

echo "Detected architecture: $TARGET_ASSET"

echo "[3/3] Fetching latest client release from GitHub..."
DOWNLOAD_URL=$(curl -s "https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/releases" | \
    jq -r "[.[] | select(.tag_name | startswith(\"client-\"))][0].assets[] | select(.name | contains(\"$TARGET_ASSET\")) | .browser_download_url" | head -n 1)

if [ -z "$DOWNLOAD_URL" ] || [ "$DOWNLOAD_URL" == "null" ]; then
    echo "Error: Could not find a suitable release asset for $TARGET_ASSET."
    exit 1
fi

echo "New release found! Downloading from: $DOWNLOAD_URL"
curl -sL "$DOWNLOAD_URL" -o digital-music-stand
chmod +x digital-music-stand

echo ""
echo "=== Setup complete! ==="
echo "A system reboot (or logout) might be required for permission changes to take effect."
echo "Run the application using: ./digital-music-stand"