package internal

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
	MEM_FONT_DATA_START      = 0x0050
)

type InterpreterMode int

const (
	MODE_NONE InterpreterMode = iota
	MODE_CHIP8
	MODE_SUPERCHIP
	MODE_XOCHIP
)

var roms = map[string]string{
	"testsuite1":    "../chip8-roms/tests/1-chip8-logo.ch8",
	"testsuite2":    "../chip8-roms/tests/2-ibm-logo.ch8",
	"testsuite3":    "../chip8-roms/tests/3-corax+.ch8",
	"testsuite4":    "../chip8-roms/tests/4-flags.ch8",
	"testsuite5":    "../chip8-roms/tests/5-quirks.ch8",
	"testsuite6":    "../chip8-roms/tests/6-keypad.ch8",
	"octojam9title": "../chip8-roms/octojam/octojam9title.ch8",
}

var fontData = []byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

var (
	isRunning           bool
	runningMutex        sync.Mutex
	stopInterpreterChan = make(chan bool)

	memory          []byte
	memoryMutex     sync.Mutex
	interpreterMode InterpreterMode

	timerMutex sync.RWMutex
	delayTimer uint8
	soundTimer uint8

	opcodePC      int32
	opcodePCMutex sync.Mutex

	pc            uint16
	indexRegister uint16
	stack         Stack
	registers     []uint8

	keyAwaitingRelease *int
)

func resetInterpreter(mode InterpreterMode) {
	// Stop the interpreter, if it's running
	tryStopInterpreter()

	// Instantiate memory, registers, timers, and counters
	pc = uint16(512) // Program Counter
	indexRegister = uint16(0)
	stack = Stack{}
	registers = make([]uint8, 16)

	opcodePC = 0

	memoryMutex.Lock()
	defer memoryMutex.Unlock()
	memory = make([]byte, 4096)
	// Load fonts into memory, starting at MEM_FONT_DATA_START
	for i, data := range fontData {
		memory[MEM_FONT_DATA_START+i] = data
	}

	interpreterMode = mode
}

func tryStartInterpreter() {
	runningMutex.Lock()
	isRunning := isRunning
	runningMutex.Unlock()
	if !isRunning {
		go interpreterLoop()
	}
}

// tryStopInterpreter sends a stop signal if an interpreter loop is running
func tryStopInterpreter() {
	runningMutex.Lock()
	isRunning := isRunning
	runningMutex.Unlock()
	if isRunning {
		fmt.Printf("Stopping interpreter")
		stopInterpreterChan <- true
		// Await confirmation that the interpreter has stopped
		<-stopInterpreterChan
		return
	}
}

func loadRom(romName string) {
	// Open the ROM on the filesystem
	rom, err := os.Open(roms[romName])
	if err != nil {
		log.Fatal(err)
	}
	loadRomData(rom)
}

func loadRomData(romFile RomFileReader) {
	func() {
		// Load the ROM into CHIP-8 memory
		memoryMutex.Lock()
		defer memoryMutex.Unlock()

		_, err := romFile.Read(memory[512:])
		if err != io.EOF && err != nil {
			log.Fatal(err)
		}
		err = romFile.Close()
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
		case <-stopInterpreterChan:
			runningMutex.Lock()
			defer runningMutex.Unlock()
			isRunning = false
			timerChan <- true
			// Ping back on the channel to confirm that we're closed
			select {
			case stopInterpreterChan <- true:
			default:
			}
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
						clearDisplay()
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
						if interpreterMode == MODE_CHIP8 {
							registers[0x0F] = 0
						}
					case 0x2:
						// 8XY2 - Set VX to VX and VY (bitwise)
						registers[uint8(x)] = registers[uint8(x)] & registers[uint8(y)]
						if interpreterMode == MODE_CHIP8 {
							registers[0x0F] = 0
						}
					case 0x3:
						// 8XY3 - Set VX to VX xor VY
						registers[uint8(x)] = registers[uint8(x)] ^ registers[uint8(y)]
						if interpreterMode == MODE_CHIP8 {
							registers[0x0F] = 0
						}
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
						if interpreterMode == MODE_CHIP8 {
							registers[uint8(x)] = registers[uint8(y)]
						}
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
						if interpreterMode == MODE_CHIP8 {
							registers[uint8(x)] = registers[uint8(y)]
						}
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
						if interpreterMode == MODE_CHIP8 {
							<-verticalBlankChan
						}
						displayMutex.Lock()
						defer displayMutex.Unlock()
						for i := 0; i < int(n); i++ {
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
					} else {
						registers[0x0F] = 0
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
							keypressDetected := false
							if interpreterMode == MODE_CHIP8 {
								if keyAwaitingRelease != nil {
									// Wait for the previously flagged 'pressed' key to be released
									if !input[*keyAwaitingRelease] {
										keypressDetected = true
										registers[x] = uint8(*keyAwaitingRelease)
										keyAwaitingRelease = nil
									}
								} else {
									for i, key := range input {
										if key {
											// Flag the first pressed key
											keyAwaitingRelease = &i
										}
									}
								}
							} else {
								for i, key := range input {
									if key {
										keypressDetected = true
										registers[x] = uint8(i)
									}
								}
							}
							if !keypressDetected {
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
						setChar := registers[uint8(x)]
						indexRegister = uint16(MEM_FONT_DATA_START + setChar*5)
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
						if interpreterMode == MODE_CHIP8 {
							indexRegister += uint16(x) + 1
						}
					case 0x65:
						// FX65 - Fetches values for V0 to VX from memory, starting at address I
						for i := 0; i <= int(x); i++ {
							registers[i] = memory[indexRegister+uint16(i)]
						}
						if interpreterMode == MODE_CHIP8 {
							indexRegister += uint16(x) + 1
						}
					}
				default:
					unsupportedOpcode(opcode)
				}

				opcodePC = int32((pc - 512) / 2)
			}()
			t := time.Now()
			elapsed := t.Sub(start)
			// Restrict refresh rate
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
