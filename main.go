package main

var (
	quitChan = make(chan bool)
)

func main() {
	resetDisplay()
	resetInterpreter()

	go interpreterLoop()
	windowLoop()
	quitChan <- true
}
