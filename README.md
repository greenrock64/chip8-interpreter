# CHIP-8 Interpreter #

A CHIP-8 Interpreter and debugger written in Go. Supports ROMS written for the COSMAC CHIP-8 interpreter.

## Features ##

- CHIP-8 instruction support (COSMAC)
- UI for loading ROMS and managing the interpreter.
- File picker for loading ROM files

## Compiling ##
The app is written in Go, and uses SDL to render the CHIP-8 window. Following the steps below should be sufficient to get it compiling.
- Install [Go v1.24+](https://go.dev/dl).
- Setup go-sdl2 as per the [README](https://github.com/veandco/go-sdl2/tree/v0.4.x?tab=readme-ov-file#requirements).
- `go run .`

Written and tested on Windows, but there shouldn't be any reason it wouldn't work on Linux/MacOS.

## References ##

- https://github.com/Timendus/chip8-test-suite
- https://tobiasvl.github.io/blog/write-a-chip-8-emulator/
- https://en.wikipedia.org/wiki/CHIP-8