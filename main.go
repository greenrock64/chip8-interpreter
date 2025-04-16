package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"sync"
	"time"

	g "github.com/AllenDang/giu"
)

const (
	FRAMERATE = 60
)

var (
	display      = make([][]bool, 64)
	displayMutex sync.Mutex
	memory       = make([]byte, 4096)
	memoryMutex  sync.Mutex

	opcodePC      = int32(0)
	opcodePCMutex sync.Mutex

	pixelWidth   = 8
	pixelHeight  = 8
	windowWidth  = pixelWidth * 64
	windowHeight = pixelHeight * 32
)

func main() {
	// Instantiate memory, registers, timers, and counters
	pc := uint16(512) // Program Counter
	indexRegister := uint16(0)
	stack := Stack{}
	delayTimer := 0
	soundTimer := 0
	registers := make([]uint8, 16)

	// Display setup
	resetDisplay()

	// Load the test ROM
	// rom, err := os.Open("../chip8-roms/tests/1-chip8-logo.ch8")
	// rom, err := os.Open("../chip8-roms/tests/2-ibm-logo.ch8")
	rom, err := os.Open("../chip8-roms/tests/3-corax+.ch8")
	if err != nil {
		log.Fatal(err)
	}
	func() {
		memoryMutex.Lock()
		defer memoryMutex.Unlock()
		_, err = rom.Read(memory[512:])
		if err != io.EOF && err != nil {
			log.Fatal(err)
		}
		err = rom.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Window Operations
	window := g.NewMasterWindow("CHIP-8 Interpreter", 1000, 800, 0)

	// Main Loop
	running := true
	go func() {
		for running {
			func() {
				ins1 := memory[pc]
				pc++
				ins2 := memory[pc]
				pc++
				opcode := uint16(ins1)<<8 + uint16(ins2)

				fmt.Printf("Opcode - %X\n", opcode)
				opcodePCMutex.Lock()
				defer opcodePCMutex.Unlock()

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
						resetDisplay()
					case 0x00EE: // Return from Subroutine
						pc = stack.Pop()
					}
				case 0x10:
					// 1NNN - Jump
					pc = nnn
				case 0x20:
					// 2NNN -  Call subroutine at NNN
					stack.Push(pc)
					pc = nnn
				case 0x30:
					// 3XNN - Skip if VX = NN
					if registers[uint8(x)] == nn {
						pc += 2
					}
				case 0x40:
					// 4XNN - Skip if VX != NN
					if registers[uint8(x)] != nn {
						pc += 2
					}
				case 0x50:
					// 5XY0 - Skip if VX == VY
					if registers[uint8(x)] == registers[uint8(y)] {
						pc += 2
					}
				case 0x60:
					// 6XNN - Save NN to Register
					registers[uint8(x)] = nn
				case 0x70:
					// 7XNN - Add NN to VX
					registers[uint8(x)] += nn
				case 0x80:
					switch n {
					case 0x0:
						// 8XY0 - Set VX to VY
						registers[uint8(x)] = registers[uint8(y)]
					case 0x1:
						// 8XY1 - Set VX to VX or VY (bitwise)
						registers[uint8(x)] = registers[uint8(x)] | registers[uint8(y)]
					case 0x2:
						// 8XY2 - Set VX to VX and VY (bitwise)
						registers[uint8(x)] = registers[uint8(x)] & registers[uint8(y)]
					case 0x3:
						// 8XY3 - Set VX to VX xor VY
						registers[uint8(x)] = registers[uint8(x)] ^ registers[uint8(y)]
					case 0x4:
						// 8XY4 - Add VY to VX (setting VF to 1 on overflow)
						newVal := uint16(registers[uint8(x)]) + uint16(registers[uint8(y)])
						if newVal > 255 {
							registers[0xF] = 1
						} else {
							registers[0xF] = 0
						}
						registers[uint8(x)] = uint8(newVal)
					case 0x5:
						// 8XY5 - Sub VY from VX (setting VF to 0 on underflow)
						if registers[uint8(x)] >= registers[uint8(y)] {
							registers[0xF] = 1
						} else {
							registers[0xF] = 0
						}
						registers[uint8(x)] -= registers[uint8(y)]
					case 0x6:
						// 8XY6 - Bitshift VX right 1, storing LSB in VF
						registers[0xF] = registers[uint8(x)] & 1
						registers[uint8(x)] = registers[uint8(x)] >> 1
					case 0x7:
						// 8XY7 - Set VX to VY - VX (setting VF to 0 on underflow)
						if registers[uint8(y)] >= registers[uint8(x)] {
							registers[0xF] = 1
						} else {
							registers[0xF] = 0
						}
						registers[uint8(x)] = registers[uint8(y)] - registers[uint8(x)]
					case 0xE:
						// 8XYE - Bitshift VX left 1, storing MSB in VF
						registers[0xF] = registers[uint8(x)] & 128
						registers[uint8(x)] = registers[uint8(x)] << 1
					}
				case 0x90:
					// 9XY0 - Skip if VX != VY
					if registers[uint8(x)] != registers[uint8(y)] {
						pc += 2
					}
				case 0xA0:
					// ANNN - Save NNN to Index Register
					indexRegister = nnn
				case 0xB0:
					// BNNN - Jump to address NNN plus V0
					pc = nnn + uint16(registers[0])
				case 0xD0:
					// DXYN - Draw to display
					memPos := indexRegister
					posX := int(registers[x])
					posY := registers[y]

					didUnset := false
					func() {
						displayMutex.Lock()
						defer displayMutex.Unlock()
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
					}()
					if didUnset {
						registers[0x0F] = 1
					}
				case 0xF:
					switch nn {
					case 0x07:
						// FX07 - Set VX to the value of the delay timer
						unsupportedOpcode(opcode)
					case 0x0A:
						// FX0A - Await keypress
						unsupportedOpcode(opcode)
					case 0x15:
						// FX15 - Set the delay timer to VX
						unsupportedOpcode(opcode)
					case 0x18:
						// FX18 - Set the sound timer to VX
						unsupportedOpcode(opcode)
					case 0x1E:
						// FX1E - Add VX to I
						indexRegister += uint16(registers[uint8(x)])
					case 0x29:
						// FX29 - Set I to the location of the sprite for character VX
						unsupportedOpcode(opcode)
					case 0x33:
						// FX33 - Store a BCD representation of VX to memory location I
						// Representation is i = hundreds, i+1 = tens, i+2 = ones
						hundreds := registers[uint8(x)] / 100
						tens := registers[uint8(x)] - (100*hundreds)/10
						ones := registers[uint8(x)] - (100 * hundreds) - (10 * tens)
						memory[indexRegister] = hundreds
						memory[indexRegister+1] = tens
						memory[indexRegister+2] = ones
					case 0x55:
						// FX55 - Stores V0 to VX in memory, starting at address I
						for i := 0; i <= int(x); i++ {
							memory[indexRegister+uint16(i)] = registers[uint8(int(x)+i)]
						}
					case 0x65:
						// FX65 - Fetches values for V0 to VX from memory, starting at address I
						for i := 0; i <= int(x); i++ {
							registers[uint8(x)] = memory[indexRegister+uint16(i)]
						}
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

				opcodePC = int32((pc - 512) / 2)
			}()
			// Cap to 60Hz
			delay := (1000 / FRAMERATE) /*  - loopTime */
			time.Sleep(time.Millisecond * time.Duration(delay))
		}
	}()

	window.Run(windowLoop)
	running = false
}

func unsupportedOpcode(opcode uint16) {
	fmt.Printf("Unsupported opcode (%x)\n", opcode)
}

func resetDisplay() {
	display = make([][]bool, 64)
	for i := range display {
		display[i] = make([]bool, 32)
	}
}

func windowLoop() {
	g.SingleWindow().Layout(
		g.Row(
			g.Column(
				g.Child().Size(float32(windowWidth), float32(windowHeight)).Border(false).Flags(g.WindowFlagsNoBackground).Layout(
					g.Custom(func() {
						displayMutex.Lock()
						defer displayMutex.Unlock()
						canvas := g.GetCanvas()
						pos := g.GetCursorScreenPos()
						col := color.RGBA{200, 75, 75, 255}
						// Border
						canvas.AddRect(pos.Add(image.Pt(0, 0)), pos.Add(image.Pt(windowWidth, windowHeight)), col, 5, g.DrawFlagsNone, 1)
						// Content
						for x := range len(display) {
							for y := range len(display[x]) {
								if display[x][y] {
									canvas.AddRectFilled(
										pos.Add(image.Pt(x*pixelWidth, y*pixelHeight)),
										pos.Add(image.Pt(x*pixelWidth+pixelWidth, y*pixelHeight+pixelHeight)),
										col, 0, 0,
									)
								}
							}
						}
					}),
				),
			),
			g.Column(
				g.Child().Size(300, 200).Layout(
					g.Label("Memory"),
					g.ListClipper().Layout(
						g.Custom(func() {
							memoryMutex.Lock()
							defer memoryMutex.Unlock()
							opcodePCMutex.Lock()
							defer opcodePCMutex.Unlock()

							memoryDisplay := []string{}
							// FIXME - Organise opcodes for display outside the window logic
							for i := 512; i < 4096; i += 2 {
								opcode := uint16(memory[i])<<8 + uint16(memory[i+1])
								memoryDisplay = append(memoryDisplay, fmt.Sprintf("(%#04x) - %#04X", i, opcode))
							}
							test := g.ListBox(memoryDisplay).SelectedIndex(&opcodePC)
							test.Build()
						}),
					),
				),
			),
		),
	)
}
