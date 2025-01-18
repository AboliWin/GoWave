# GoWave

GoWave is a simple, lightweight radio application written in Go (Golang). It allows users to stream music from a selection of channels and provides basic audio controls, such as volume adjustments. The application is designed to be easy to use while offering essential features for music playback.

## Features

- Stream a variety of music channels.
- Control volume levels (turn up/down).
- Lightweight and simple design.
- Open-source, built with Go (Golang).

## Known Issues

- **UI Delay**: There may be a slight delay when updating the user interface, particularly when making volume adjustments or changing channels.
- **Audio Playback**: Some issues exist with the library used for playing audio, which could lead to inconsistent behavior.
- **Cross-Compilation**: The application may not compile correctly on certain operating systems. For more details on this issue, refer to the [cross-compiling guide](https://github.com/ebitengine/oto#crosscompiling).

## Build Instructions

To clone, build, and run GoWave, follow these steps:

```bash
# Clone the repository
git clone https://github.com/AboliWin/GoWave.git

# Navigate to the project directory
cd GoWave

# Install dependencies
go mod tidy

# Build the project
go build main.go

# Run the application
./main
```

## Contribution

Feel free to contribute to GoWave! If you encounter any issues or have feature suggestions, please open an issue or submit a pull request.


![{679108A6-E8AB-4789-9447-5207707DD32D}](https://github.com/user-attachments/assets/70d94e98-947f-4a62-82ad-76b59ad717f4)
