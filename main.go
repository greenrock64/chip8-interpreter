package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	quitChan = make(chan bool)
)

func main() {
	fyneApp := app.New()
	fyneWindow := fyneApp.NewWindow("CHIP-8 Controller")
	fyneWindow.Resize(fyne.NewSize(800, 600))

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
		fyne.NewMenuItem("Close Interpreter", func() { resetInterpreter(); resetDisplay() }),
	)
	mainMenu := fyne.NewMainMenu(
		fileMenu,
	)
	fyneWindow.SetMainMenu(mainMenu)

	// Setup SDL Display
	initialiseWindow()
	defer sdl.Quit()
	defer window.Destroy()

	fyneWindow.ShowAndRun()
	tryCloseDisplay()
}
