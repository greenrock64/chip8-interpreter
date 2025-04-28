package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	selectedInterpreterMode InterpreterMode = MODE_CHIP8
)

func StartRom(romName string) {
	clearDisplay()
	tryOpenDisplay()

	resetInterpreter(selectedInterpreterMode)
	loadRom(romName)
	tryStartInterpreter()
}

func CloseInterpreter() {
	resetInterpreter(MODE_NONE)
	tryCloseDisplay()
}

func main() {
	fyneApp := app.New()
	fyneWindow := fyneApp.NewWindow("CHIP-8 Controller")
	fyneWindow.Resize(fyne.NewSize(400, 400))

	loadRomMenu := fyne.NewMenuItem("Load ROM", nil)
	loadRomMenu.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("Test Suite 1", func() { go StartRom("testsuite1") }),
		fyne.NewMenuItem("Test Suite 2", func() { go StartRom("testsuite2") }),
		fyne.NewMenuItem("Test Suite 3", func() { go StartRom("testsuite3") }),
		fyne.NewMenuItem("Test Suite 4", func() { go StartRom("testsuite4") }),
		fyne.NewMenuItem("Test Suite 5", func() { go StartRom("testsuite5") }),
		fyne.NewMenuItem("Test Suite 6", func() { go StartRom("testsuite6") }),
		fyne.NewMenuItem("Octojam - Title 9", func() { go StartRom("octojam9title") }),
	)
	fileMenu := fyne.NewMenu("CHIP-8",
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

	// Setup SDL Display
	initialiseWindow()
	defer sdl.Quit()
	defer window.Destroy()

	fyneWindow.ShowAndRun()
	go CloseInterpreter()
}
