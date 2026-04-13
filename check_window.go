package main
import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)
func main() {
	a := app.New()
	w := a.NewWindow("Test")
	fmt.Printf("%T\n", w)
}
