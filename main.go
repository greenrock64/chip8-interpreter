package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	quitChan                                = make(chan bool)
	selectedInterpreterMode InterpreterMode = MODE_CHIP8
)

func main() {
	fyneApp := app.New()
	fyneWindow := fyneApp.NewWindow("CHIP-8 Controller")
	fyneWindow.Resize(fyne.NewSize(400, 400))

	loadRomMenu := fyne.NewMenuItem("Load ROM", nil)
	loadRomMenu.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("Test Suite 1", func() { StartRom("testsuite1") }),
		fyne.NewMenuItem("Test Suite 2", func() { StartRom("testsuite2") }),
		fyne.NewMenuItem("Test Suite 3", func() { StartRom("testsuite3") }),
		fyne.NewMenuItem("Test Suite 4", func() { StartRom("testsuite4") }),
		fyne.NewMenuItem("Test Suite 5", func() { StartRom("testsuite5") }),
		fyne.NewMenuItem("Test Suite 6", func() { StartRom("testsuite6") }),
		fyne.NewMenuItem("Octojam - Title 9", func() { StartRom("octojam9title") }),
	)
	fileMenu := fyne.NewMenu("CHIP-8",
		loadRomMenu,
		fyne.NewMenuItem("Close Interpreter", func() { resetInterpreter(MODE_NONE); resetDisplay() }),
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
	optionsMenu := fyne.NewMenu("Options",
		selectModeMenu,
	)
	mainMenu := fyne.NewMainMenu(
		fileMenu,
		optionsMenu,
	)
	fyneWindow.SetMainMenu(mainMenu)

	// Setup SDL Display
	initialiseWindow()
	defer sdl.Quit()
	defer window.Destroy()

	fyneWindow.ShowAndRun()
	tryCloseDisplay()
}
