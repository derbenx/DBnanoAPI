package main
import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)
func main() {
	a := app.New()
	w := a.NewWindow("Test")
	_, ok := w.(desktop.Window)
	fmt.Println("Is desktop window:", ok)
}
