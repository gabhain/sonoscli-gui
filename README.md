# sonoscli-gui

A modern, card-based GUI for controlling Sonos speakers, built with Go and the [Fyne](https://fyne.io) toolkit.

This project provides a visual interface for common Sonos operations, including playback control, volume/EQ management, grouping, and more.

## Features

- **Room & Group Discovery**: Automatically finds all Sonos speakers on your network.
- **Now Playing**: View track information, album art, and real-time progress.
- **Playback Control**: Play, pause, skip, and seek.
- **Audio Settings**: Card-based interface for Bass, Treble, Loudness, Night Mode, and Speech Enhancement.
- **Line-Out Control**: Support for Sonos Port/Connect output modes (Variable, Fixed, Pass-Through).
- **Grouping**: Manage speaker groups and control group volume/mute.
- **Inputs & Favorites**: Quick access to physical inputs (TV/Line-In) and Sonos Favorites.
- **Scenes**: Save and restore entire system states (grouping and volume levels).

## Installation

### 1. Install Dependencies
This project requires Go 1.26+ and various system graphics libraries. You can install them automatically using:

```bash
make deps
```

*Note: On Linux, this will prompt for your sudo password to install development headers. On macOS, it will check for Xcode Command Line Tools.*

### 2. Build or Run
```bash
# Run directly
go run .

# Or build the binary
make build
```


## Credits

This project is a GUI frontend that leverages the core Sonos logic and internal architecture inspired by the [sonoscli](https://github.com/skaringa/sonoscli) project. 

Special thanks to the original `sonoscli` contributors for their work on the UPnP/SOAP protocol implementations and CLI structure.

## License

This project is licensed under the same terms as the original `sonoscli` project (MIT License).
