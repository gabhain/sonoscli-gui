# sonoscli-gui

A modern, cross-platform, card-based GUI for controlling Sonos speakers, built with [Go](https://go.dev) and the [Fyne](https://fyne.io) toolkit.

This project provides a sleek and intuitive visual interface for managing your Sonos system, from basic playback to complex multi-room grouping and system-wide "scenes."

## 🚀 Key Features

- **Automatic Discovery & Manual Connection**: Instantly finds all Sonos speakers on your local network using SSDP/UPnP, with a manual IP fallback for complex networks.
- **Rich "Now Playing" Experience**: High-quality album art display, real-time track progress bars, and detailed metadata.
- **Comprehensive Playback Control**: Effortlessly play, pause, skip, seek, and toggle repeat/shuffle modes.
- **Advanced Audio Settings**: A clean, card-based interface for managing:
  - Bass & Treble Equalizer
  - Loudness
  - Night Mode & Speech Enhancement (for Home Theater devices)
  - Line-Out Control (Variable, Fixed, Pass-Through for Port/Connect)
- **Dynamic Grouping**: Real-time management of speaker groups. Join, leave, or adjust group-wide volume and mute settings.
- **Favorites & Queue Management**: Quick access to your Sonos Favorites and full visibility/control of the current playback queue.
- **Physical Inputs**: Switch to TV (HDMI/Optical) or Line-In inputs with a single click.
- **System Scenes**: Save your entire system's state—including grouping and volume levels—and restore it instantly.
- **Web Remote Control**: A built-in, mobile-friendly web dashboard accessible from any device on your network.
  - Real-time synchronization with the desktop application.
  - Full playback, volume, and EQ control from your phone or tablet browser.
  - No additional setup required—starts automatically with the main app.

## 📱 Web Remote Control

The application includes a sleek, real-time web interface. Once the main application is running, you can access the remote control from any device on your local network:

1.  Open your browser to `http://localhost:8081` (or use your computer's local IP address).
2.  The web UI provides a mobile-optimized view of your entire system, perfect for controlling music from your couch.
3.  Changes made on the web are reflected instantly in the desktop app and vice versa.

### Prerequisites
- **Go 1.26+**
- System-specific graphics libraries (see below).

### Quick Start
You can install all necessary dependencies and build the application using the provided `Makefile`:

```bash
# 1. Install dependencies (requires sudo on Linux)
make deps

# 2. Build the application
make build

# 3. Run it
./sonoscli-gui
```

### Platform-Specific Dependencies

#### macOS
Ensure you have the Xcode Command Line Tools installed:
```bash
xcode-select --install
```

#### Linux
You will need development headers for X11 and OpenGL. On Ubuntu/Debian:
```bash
sudo apt-get install -y libgl1-mesa-dev xorg-dev libwayland-dev libx11-dev libxrandr-dev libxinerama-dev libxcursor-dev libxi-dev
```

#### Windows
We recommend using [MSYS2](https://www.msys2.org/) with the Mingw-w64 toolchain for the best compatibility with Fyne.

## 🛠 Development

### Building with Fyne CLI
For a more polished experience (including application icons and metadata), we recommend using the `fyne` tool:

```bash
go install fyne.io/tools/cmd/fyne@latest
fyne package -os darwin -icon Icon.png
```

### Project Structure
- `main.go`: Entry point and UI orchestration.
- `internal/sonos/`: Core UPnP/SOAP protocol implementations for Sonos.
- `internal/web/`: High-performance Go web server and static assets for the remote dashboard.
- `internal/scenes/`: Logic for saving and restoring system states.
- `internal/appconfig/`: Persistent application configuration.

## 📜 Credits

This project is a GUI frontend that leverages core Sonos logic inspired by the [sonoscli](https://github.com/skaringa/sonoscli) project. Special thanks to the original contributors for their robust implementation of the Sonos protocol.

## 📄 License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.
