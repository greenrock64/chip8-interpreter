package internal

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	selectedInterpreterMode InterpreterMode = MODE_CHIP8
)

type RomFileReader interface {
	Read([]byte) (int, error)
	Close() error
}

func StartRom(romName string, romFile RomFileReader) {
	// Reset the display
	clearDisplay()
	tryOpenDisplay()

	// Reset the Interpreter and load the ROM
	resetInterpreter(selectedInterpreterMode)
	if romName != "" {
		loadRom(romName)
	} else if romFile != nil {
		loadRomData(romFile)
	} else {
		fmt.Println("no rom data provided during Start call")
	}
	tryStartInterpreter()
}

func CloseInterpreter() {
	resetInterpreter(MODE_NONE)
	tryCloseDisplay()
}

func RunApp() {
	fyneApp := app.New()
	fyneWindow := fyneApp.NewWindow("CHIP-8 Controller")
	fyneWindow.Resize(fyne.NewSize(600, 500))

	loadRomMenu := fyne.NewMenuItem("Load ROM", nil)
	loadRomMenu.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("Test Suite 1", func() { go StartRom("testsuite1", nil) }),
		fyne.NewMenuItem("Test Suite 2", func() { go StartRom("testsuite2", nil) }),
		fyne.NewMenuItem("Test Suite 3", func() { go StartRom("testsuite3", nil) }),
		fyne.NewMenuItem("Test Suite 4", func() { go StartRom("testsuite4", nil) }),
		fyne.NewMenuItem("Test Suite 5", func() { go StartRom("testsuite5", nil) }),
		fyne.NewMenuItem("Test Suite 6", func() { go StartRom("testsuite6", nil) }),
		fyne.NewMenuItem("Octojam - Title 9", func() { go StartRom("octojam9title", nil) }),
	)
	fileMenu := fyne.NewMenu("CHIP-8",
		fyne.NewMenuItem("Open File", func() {
			fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
				if reader != nil {
					go StartRom("", reader)
				}
			}, fyneWindow)
			// Only allow reading of CHIP-8 rom files
			fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".ch8"}))

			curDir, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			listableURI, err := storage.ListerForURI(storage.NewFileURI(curDir))
			if err != nil {
				panic(err)
			}
			fileDialog.SetLocation(listableURI)
			fileDialog.Show()
		}),
		loadRomMenu,
		fyne.NewMenuItem("Close Interpreter", func() { go CloseInterpreter() }),
	)

	selectModeMenu := fyne.NewMenuItem("Hardware Mode", nil)

	selectMode := func(mode InterpreterMode) {
		selectModeMenu.ChildMenu.Items[selectedInterpreterMode-1].Checked = false
		selectedInterpreterMode = mode
		selectModeMenu.ChildMenu.Items[mode-1].Checked = true
	}
	selectModeMenu.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("CHIP-8", func() { selectMode(MODE_CHIP8) }),
		fyne.NewMenuItem("SUPER-CHIP", func() { selectMode(MODE_SUPERCHIP) }),
		fyne.NewMenuItem("XO-CHIP", func() { selectMode(MODE_XOCHIP) }),
	)
	selectModeMenu.ChildMenu.Items[0].Checked = true
	selectModeMenu.ChildMenu.Items[1].Disabled = true
	selectModeMenu.ChildMenu.Items[2].Disabled = true
	optionsMenu := fyne.NewMenu("Options",
		selectModeMenu,
	)
	mainMenu := fyne.NewMainMenu(
		fileMenu,
		optionsMenu,
	)
	fyneWindow.SetMainMenu(mainMenu)
	fyneWindow.Show()

	// Setup SDL Display
	initialiseWindow()
	defer sdl.Quit()
	defer window.Destroy()

	fyneApp.Run()
	go CloseInterpreter()
}
