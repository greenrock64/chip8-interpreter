package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	INSTRUCTION_REFRESH_RATE = 600
	TIMER_REFRESH_RATE       = 60
)

var (
	stopChan     = make(chan bool)
	isRunning    bool
	runningMutex sync.Mutex

	memory      []byte
	memoryMutex sync.Mutex

	timerMutex sync.RWMutex
	delayTimer uint8
	soundTimer uint8

	opcodePC      int32
	opcodePCMutex sync.Mutex

	pc            uint16
	indexRegister uint16
	stack         Stack
	registers     []uint8
)

func resetInterpreter() {
	// Stop the interpreter, if it's running
	runningMutex.Lock()
	if isRunning {
		stopChan <- true
	}
	runningMutex.Unlock()

	// Instantiate memory, registers, timers, and counters
	pc = uint16(512) // Program Counter
	indexRegister = uint16(0)
	stack = Stack{}
	registers = make([]uint8, 16)

	opcodePC = 0

	memoryMutex.Lock()
	defer memoryMutex.Unlock()
	memory = make([]byte, 4096)

	// Load the test ROM
	// rom, err := os.Open("../chip8-roms/tests/1-chip8-logo.ch8")
	// rom, err := os.Open("../chip8-roms/tests/2-ibm-logo.ch8")
	// rom, err := os.Open("../chip8-roms/tests/3-corax+.ch8")
	// rom, err := os.Open("../chip8-roms/tests/4-flags.ch8")
	// rom, err := os.Open("../chip8-roms/tests/5-quirks.ch8")
	rom, err := os.Open("../chip8-roms/tests/6-keypad.ch8")
	// rom, err := os.Open("../chip8-roms/octojam/octojam9title.ch8")
	if err != nil {
		log.Fatal(err)
	}
	func() {
		_, err = rom.Read(memory[512:])
		if err != io.EOF && err != nil {
			log.Fatal(err)
		}
		err = rom.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func interpreterLoop() {

	runningMutex.Lock()
	isRunning = true
	runningMutex.Unlock()

	repeatOpcode := func() {
		pc -= 2
	}
	skipNextOpcode := func() {
		pc += 2
	}
	timerChan := make(chan bool)
	go timerHandler(timerChan)

	for {
		select {
		case <-stopChan:
			runningMutex.Lock()
			isRunning = false
			timerChan <- true
			runningMutex.Unlock()
			return
		default:
			start := time.Now()
			func() {
				ins1 := memory[pc]
				pc++
				ins2 := memory[pc]
				pc++
				opcode := uint16(ins1)<<8 + uint16(ins2)

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
						clearScreen()
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
						skipNextOpcode()
					}
				case 0x40:
					// 4XNN - Skip if VX != NN
					if registers[uint8(x)] != nn {
						skipNextOpcode()
					}
				case 0x50:
					// 5XY0 - Skip if VX == VY
					if registers[uint8(x)] == registers[uint8(y)] {
						skipNextOpcode()
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
						var flag uint8 = 0
						if newVal > 255 {
							flag = 1
						}
						registers[uint8(x)] = uint8(newVal)
						registers[0xF] = flag
					case 0x5:
						// 8XY5 - Sub VY from VX (setting VF to 0 on underflow)
						var flag uint8 = 0
						if registers[uint8(x)] >= registers[uint8(y)] {
							flag = 1
						}
						registers[uint8(x)] -= registers[uint8(y)]
						registers[0xF] = flag
					case 0x6:
						// 8XY6 - Bitshift VX right 1, setting VF 1 to if LSB was shifted out
						flag := registers[uint8(x)] & 1
						registers[uint8(x)] = registers[uint8(x)] >> 1
						registers[0xF] = flag
					case 0x7:
						// 8XY7 - Set VX to VY - VX (setting VF to 0 on underflow)
						var flag uint8 = 0
						if registers[uint8(y)] >= registers[uint8(x)] {
							flag = 1
						}
						registers[uint8(x)] = registers[uint8(y)] - registers[uint8(x)]
						registers[0xF] = flag
					case 0xE:
						// 8XYE - Bitshift VX left 1, setting VF to 1 if MSB was shifted out
						flag := registers[uint8(x)] >> 7
						registers[uint8(x)] = registers[uint8(x)] << 1
						registers[0xF] = flag
					}
				case 0x90:
					// 9XY0 - Skip if VX != VY
					if registers[uint8(x)] != registers[uint8(y)] {
						skipNextOpcode()
					}
				case 0xA0:
					// ANNN - Save NNN to Index Register
					indexRegister = nnn
				case 0xB0:
					// BNNN - Jump to address NNN plus V0
					pc = nnn + uint16(registers[0])
				case 0xC0:
					// CXNN - Set VX to the NN & Rand
					rand := rand.Intn(255)
					registers[x] = nn & uint8(rand)
				case 0xD0:
					// DXYN - Draw to display
					memPos := indexRegister
					posX := int(registers[x]) % horizontalPixelCount
					posY := int(registers[y]) % verticalPixelCount

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
									if displayPosX >= horizontalPixelCount {
										break
									}
									display[displayPosX][posY] = !display[displayPosX][posY]
									if !display[displayPosX][posY] {
										didUnset = true
									}
								}
							}

							memPos++
							posY++
							if posY >= verticalPixelCount {
								break
							}
						}
					}()
					if didUnset {
						registers[0x0F] = 1
					}
				case 0xE0:
					switch nn {
					case 0x9E:
						fallthrough
					case 0xA1:
						func() {
							inputMutex.Lock()
							defer inputMutex.Unlock()
							key := registers[x]
							if input[key] && nn == 0x9E {
								skipNextOpcode()
							}
							if !input[key] && nn == 0xA1 {
								skipNextOpcode()
							}
						}()
					}
				case 0xF0:
					switch nn {
					case 0x07:
						// FX07 - Set VX to the value of the delay timer
						func() {
							timerMutex.RLock()
							defer timerMutex.RUnlock()
							registers[x] = uint8(delayTimer)
						}()
					case 0x0A:
						// FX0A - Await keypress
						func() {
							inputMutex.Lock()
							defer inputMutex.Unlock()
							keyIsPressed := false
							for _, key := range input {
								if key {
									keyIsPressed = true
								}
							}
							if !keyIsPressed {
								repeatOpcode()
							}
						}()
					case 0x15:
						// FX15 - Set the delay timer to VX
						func() {
							timerMutex.RLock()
							defer timerMutex.RUnlock()
							delayTimer = registers[x]
						}()
					case 0x18:
						// FX18 - Set the sound timer to VX
						func() {
							timerMutex.RLock()
							defer timerMutex.RUnlock()
							soundTimer = registers[x]
						}()
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
						tens := (registers[uint8(x)] - (100 * hundreds)) / 10
						ones := registers[uint8(x)] - (100 * hundreds) - (10 * tens)
						memory[indexRegister] = hundreds
						memory[indexRegister+1] = tens
						memory[indexRegister+2] = ones
					case 0x55:
						// FX55 - Stores V0 to VX in memory, starting at address I
						for i := 0; i <= int(x); i++ {
							memory[indexRegister+uint16(i)] = registers[i]
						}
					case 0x65:
						// FX65 - Fetches values for V0 to VX from memory, starting at address I
						for i := 0; i <= int(x); i++ {
							registers[i] = memory[indexRegister+uint16(i)]
						}
					}
				default:
					unsupportedOpcode(opcode)
				}

				opcodePC = int32((pc - 512) / 2)
			}()
			t := time.Now()
			elapsed := t.Sub(start)
			// Cap to 60Hz
			delay := (1000 / INSTRUCTION_REFRESH_RATE) - elapsed
			time.Sleep(time.Millisecond * time.Duration(delay))
		}
	}
}

func unsupportedOpcode(opcode uint16) {
	fmt.Printf("Unsupported opcode (%x)\n", opcode)
}

func timerHandler(quit chan bool) {
	for {
		select {
		case <-quit:
			return
		default:
			start := time.Now()

			func() {
				timerMutex.Lock()
				defer timerMutex.Unlock()

				if delayTimer > 0 {
					delayTimer--
				}
				if soundTimer > 0 {
					soundTimer--
				}
			}()

			t := time.Now()
			elapsed := t.Sub(start)

			// Cap to 60Hz
			delay := (1000 / TIMER_REFRESH_RATE) - elapsed
			time.Sleep(delay * time.Millisecond)
		}
	}
}
