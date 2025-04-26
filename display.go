package main

import (
	"fmt"
	"sync"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	DISPLAY_REFRESH_RATE = 60
)

var (
	window            *sdl.Window
	isDisplaying      bool
	isDisplayingMutex sync.Mutex

	display      = make([][]bool, 64)
	displayMutex sync.Mutex

	pixelWidth           = 8
	pixelHeight          = 8
	horizontalPixelCount = 64
	verticalPixelCount   = 32
	windowWidth          = pixelWidth * horizontalPixelCount
	windowHeight         = pixelHeight * verticalPixelCount

	input      = make([]bool, 16)
	inputMutex sync.Mutex
)

func clearScreen() {
	displayMutex.Lock()
	defer displayMutex.Unlock()
	display = make([][]bool, 64)
	for i := range display {
		display[i] = make([]bool, 32)
	}
}

func resetDisplay() {
	isDisplayingMutex.Lock()
	defer isDisplayingMutex.Unlock()
	if isDisplaying {
		fmt.Println("Closing display on reset")
		quitChan <- true
	}
	clearScreen()
	fmt.Println("Display reset")
}

func tryCloseDisplay() {
	isDisplayingMutex.Lock()
	defer isDisplayingMutex.Unlock()
	if isDisplaying {
		quitChan <- true
	}
}

func initialiseWindow() {
	// Setup SDL Display
	err := sdl.Init(sdl.INIT_EVERYTHING)
	if err != nil {
		panic(err)
	}
	window, err = sdl.CreateWindow("CHIP-8 Interpreter", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(windowWidth), int32(windowHeight), sdl.WINDOW_HIDDEN)
	if err != nil {
		panic(err)
	}
}

func windowLoop() {
	fmt.Println("New windowloop")
	isDisplayingMutex.Lock()
	isDisplaying = true
	fmt.Println("Set isDisplaying")
	isDisplayingMutex.Unlock()

	window.Show()
	surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	running := true
	for running {
		select {
		case <-quitChan:
			isDisplayingMutex.Lock()
			running = false
			fmt.Println("Quitchan received")
		default:
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch event := event.(type) {
				case *sdl.KeyboardEvent:
					func() {
						if event.GetType() == sdl.KEYDOWN {
							handleKeyEvent(event.Keysym.Sym, true)
						} else {
							// KEYUP
							handleKeyEvent(event.Keysym.Sym, false)
						}
					}()
				case *sdl.QuitEvent:
					running = false
				}
			}
			if !running {
				isDisplayingMutex.Lock()
				break
			}

			loopTime := sdlLoop(surface)
			window.UpdateSurface()

			delay := (1000 / DISPLAY_REFRESH_RATE) - loopTime
			sdl.Delay(delay)
		}
	}
	window.Hide()
	defer isDisplayingMutex.Unlock()
	isDisplaying = false
	fmt.Println("Displayloop ended")
}

func sdlLoop(surface *sdl.Surface) uint32 {
	// Clear the surface
	surface.FillRect(nil, 0)

	// Set the pixel's colour and map it to the display's colourspace
	colour := sdl.Color{R: 255, G: 255, B: 255, A: 255} // White
	pixel := sdl.MapRGBA(surface.Format, colour.R, colour.G, colour.B, colour.A)

	displayMutex.Lock()
	defer displayMutex.Unlock()
	for x := range len(display) {
		for y := range len(display[x]) {
			if display[x][y] {
				// Determine the pixels location
				rect := sdl.Rect{X: int32(x * pixelWidth), Y: int32(y * pixelHeight), W: int32(pixelWidth), H: int32(pixelHeight)}
				// Draw a rectangle
				surface.FillRect(&rect, pixel)
			}
		}
	}

	return 0
}

func handleKeyEvent(keyCode sdl.Keycode, isPressed bool) {
	inputMutex.Lock()
	defer inputMutex.Unlock()
	// TODO - Handle input a bit more sanely, instead of this big switch
	switch keyCode {
	case sdl.GetKeyFromName("1"):
		input[1] = isPressed
	case sdl.GetKeyFromName("2"):
		input[2] = isPressed
	case sdl.GetKeyFromName("3"):
		input[3] = isPressed
	case sdl.GetKeyFromName("4"):
		input[12] = isPressed
	case sdl.GetKeyFromName("Q"):
		input[4] = isPressed
	case sdl.GetKeyFromName("W"):
		input[5] = isPressed
	case sdl.GetKeyFromName("E"):
		input[6] = isPressed
	case sdl.GetKeyFromName("R"):
		input[13] = isPressed
	case sdl.GetKeyFromName("A"):
		input[7] = isPressed
	case sdl.GetKeyFromName("S"):
		input[8] = isPressed
	case sdl.GetKeyFromName("D"):
		input[9] = isPressed
	case sdl.GetKeyFromName("F"):
		input[14] = isPressed
	case sdl.GetKeyFromName("Z"):
		input[10] = isPressed
	case sdl.GetKeyFromName("X"):
		input[0] = isPressed
	case sdl.GetKeyFromName("C"):
		input[11] = isPressed
	case sdl.GetKeyFromName("V"):
		input[15] = isPressed
	}
}
