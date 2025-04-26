package main

import (
	"fmt"

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
	fileMenu := fyne.NewMenu("CHIP-8",
		fyne.NewMenuItem("Load ROM", func() {
			resetInterpreter()
			fmt.Println("Reset Interpreter")
			resetDisplay()
			fmt.Println("Reset DIsplay")
			go interpreterLoop()
			fmt.Println("Started ILoop")
			go windowLoop()
			fmt.Println("Started DLoop")
		}),
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
