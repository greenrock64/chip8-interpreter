package main

import (
	"io"
	"log"
	"os"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	FRAMERATE = 60
)

var (
	display      = make([][]bool, 64)
	pixelWidth   = int32(13)
	pixelHeight  = int32(13)
	windowWidth  = pixelWidth * 64
	windowHeight = pixelHeight * 32
)

func main() {
	// Instantiate memory, registers, timers, and counters
	memory := make([]byte, 4096)
	pc := uint16(512) // Program Counter
	indexRegister := uint16(0)
	// stack := Stack{}
	delayTimer := 0
	soundTimer := 0
	registers := make([]uint8, 16)

	resetDisplay()

	// Load the test ROM
	// rom, err := os.Open("../chip8-roms/tests/1-chip8-logo.ch8")
	rom, err := os.Open("../chip8-roms/tests/2-ibm-logo.ch8")
	if err != nil {
		log.Fatal(err)
	}
	_, err = rom.Read(memory[512:])
	if err != io.EOF && err != nil {
		log.Fatal(err)
	}
	err = rom.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Window Operations
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()
	window, err := sdl.CreateWindow("CHIP-8 Interpreter", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		windowWidth, windowHeight, sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()
	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	// Main Loop
	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
			}
		}

		ins1 := memory[pc]
		pc++
		ins2 := memory[pc]
		pc++
		opcode := uint16(ins1)<<8 + uint16(ins2)

		cmdCategory := ins1 & 0xF0
		x := ins1 & 0x0F
		y := (ins2 & 0xF0) >> 4
		n := ins2 & 0x0F
		nn := uint8(ins2)
		nnn := (uint16(x) << 8) + uint16(nn)

		switch cmdCategory {
		case 0x00:
			switch opcode {
			case 0x00E0: // Clear Screen
				clearWindow(surface)
			case 0x00EE:
				unsupportedOpcode(opcode)
			}
		case 0x10:
			// 1NNN - Jump
			pc = nnn
		case 0x60:
			// 6XNN - Save NN to Register
			registers[uint8(x)] = nn
		case 0x70:
			// 7XNN - Add NN to VX
			registers[uint8(x)] += nn
		case 0xA0:
			// ANNN - Save NNN to Index Register
			indexRegister = nnn
		case 0xD0:
			// DXYN - Draw to display
			memPos := indexRegister
			posX := int(registers[x])
			posY := registers[y]

			didUnset := false
			for y := 0; y < int(n); y++ {
				sprite := memory[memPos]

				setPixels := []bool{}
				for k := 128; k >= 1; k = k / 2 {
					if int(sprite)&k > 0 {
						setPixels = append(setPixels, true)
					} else {
						setPixels = append(setPixels, false)
					}
					if k == 1 {
						break
					}
				}

				for x := 0; x < 8; x++ {
					if setPixels[x] {
						displayPosX := posX + (x)
						display[displayPosX][posY] = !display[displayPosX][posY]
						if !display[displayPosX][posY] {
							didUnset = true
						}
					}
				}

				memPos++
				posY++
			}
			if didUnset {
				registers[0x0F] = 1
			}
		default:
			unsupportedOpcode(opcode)
		}

		if delayTimer > 0 {
			delayTimer--
		}
		if soundTimer > 0 {
			soundTimer--
		}

		loopTime := updateWindow(surface)
		window.UpdateSurface()

		// Cap to 60Hz
		delay := (1000 / FRAMERATE) - loopTime
		sdl.Delay(delay)
	}
}

func unsupportedOpcode(opcode uint16) {
	log.Fatalf("Unsupported opcode (%x)", opcode)
}

func resetDisplay() {
	display = make([][]bool, 64)
	for i := range display {
		display[i] = make([]bool, 32)
	}
}

func clearWindow(surface *sdl.Surface) {
	// Clear the surface
	surface.FillRect(nil, 0)
}

func updateWindow(surface *sdl.Surface) (looptime uint32) {
	// Get time at the start of the function
	startTime := sdl.GetTicks()
	clearWindow(surface)

	// Set the pixel's colour and map it to the display's colourspace
	colour := sdl.Color{R: 255, G: 255, B: 255, A: 255} // White
	pixel := sdl.MapRGBA(surface.Format, colour.R, colour.G, colour.B, colour.A)

	for x := range len(display) {
		for y := range len(display[x]) {
			if display[x][y] {
				// Determine the pixels location
				rect := sdl.Rect{X: int32(x) * pixelWidth, Y: int32(y) * pixelHeight, W: pixelWidth, H: pixelHeight}
				// Draw a rectangle
				surface.FillRect(&rect, pixel)
			}
		}
	}

	// Calculate time passed since the start of the function
	endTime := sdl.GetTicks()
	return endTime - startTime
}
