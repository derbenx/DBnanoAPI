package main
import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)
func main() {
	a := app.New()
	w := a.NewWindow("Test")
	var _ desktop.Window = w.(desktop.Window)
}
