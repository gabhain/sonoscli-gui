#!/bin/bash

# install_deps.sh - Install system dependencies for sonoscli-gui (Fyne)

OS="$(uname -s)"

echo "Detecting OS: $OS"

case "$OS" in
    Darwin)
        echo "Installing macOS dependencies..."
        if ! xcode-select -p &>/dev/null; then
            echo "Installing Xcode Command Line Tools..."
            xcode-select --install
        else
            echo "Xcode Command Line Tools already installed."
        fi
        ;;
    Linux)
        if [ -f /etc/debian_version ]; then
            echo "Detected Debian/Ubuntu-based system."
            sudo apt-get update
            sudo apt-get install -y libgl1-mesa-dev xorg-dev libwayland-dev libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev gcc
        elif [ -f /etc/fedora-release ]; then
            echo "Detected Fedora-based system."
            sudo dnf groupinstall -y "Development Tools" "X Software Development"
            sudo dnf install -y libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel
        elif [ -f /etc/arch-release ]; then
            echo "Detected Arch-based system."
            sudo pacman -S --needed base-devel libx11 libxcursor libxrandr libxinerama mesa libxi
        else
            echo "Unsupported Linux distribution. Please install Fyne dependencies manually."
            exit 1
        fi
        ;;
    *)
        echo "Unsupported OS: $OS"
        echo "Please refer to https://developer.fyne.io/started/ for manual installation."
        exit 1
        ;;
esac

echo "System dependencies installed successfully."
