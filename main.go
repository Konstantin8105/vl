package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

var StyleDefault = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorWhite)

var StyleInput = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorBlue)

const MaxWidth = 50

type VerticalScroll struct {
	position int16
	ws       []Widget
}

type Widget interface {
	GetId() uint16
	SetWidth() uint16
	GetHeight() uint16
	GetContent(x, y int) (mainc rune, combc []rune, style Style, width int)
}

var (
	idstart    uint16
	widgets    []Widget
	screen     tcell.Screen
	line       int
	offsetLine int
)

func getId() uint16 {
	defer func() {
		idstart++
	}()
	return idstart
}


// Widget : Button
// Design : [ Ok ] [ Cancel ]

// Widget : Radiobutton
// Design : (0) choose one

// Widget : CheckBox
// Design : [V] Option

// Widget : Frame
// Design :
// +- Name ------------+
// |                   |
// +-------------------+

// Widget : Combobox
// Design :
// +-------------------+
// |                   |
// |                   |
// +-------------------+

// Widget : Scrollbar

// Widget : CollapsingHeader

// Text("...")
func Text(str string) {
	defer func() {
		line++
	}()
	posLine := line - offsetLine
	c := 0 // counter
	for _, r := range str {
		screen.SetCell(c, posLine, StyleDefault, r)
		c++
	}
}

// Input(&str)
func Input(str []rune) {
	defer func() {
		line++
	}()
	width, _ := screen.Size()
	posLine := line - offsetLine
	c := 0 // counter
	for _, r := range str {
		screen.SetCell(c, posLine, StyleInput, r)
		c++
	}

	blink := StyleInput.Reverse(true) // Blink(true)
	screen.SetCell(c, posLine, blink, '|')
	c++

	for ; c < width; c++ {
		screen.SetCell(c, posLine, StyleInput, ' ')
	}
}

// InputUint(&is)
// func InputUint(is *uint32) {
// 	var s string = fmt.Sprintf("%d", *is)
// 	Input(&s)
// }
//
// InputFloat(&x)
// func InputFloat(fl *float32) {
// 	var s string = fmt.Sprintf("%f", *fl)
// 	Input(&s)
// }
//
// var cb uint16
// ComboBox(&cb, []widgets{...})
//
// var but bool
// Button(&but, &str)

var (
	inp1 []rune
	inp2 []rune  = []rune("Hello")
	is   uint32  = 233
	fl   float32 = 0.444
)

func main() {
	tcell.SetEncodingFallback(tcell.EncodingFallbackUTF8)
	var err error
	screen, err = tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err = screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	screen.EnableMouse()
	screen.EnablePaste() // ?

	screen.SetStyle(StyleDefault)
	screen.Clear()

	quit := make(chan struct{})
	go func() {
		for {
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					close(quit)
					return
				case tcell.KeyCtrlL:
					screen.Sync()
				case tcell.KeyRune:
					inp1 = append(inp1, ev.Rune())
					inp2 = append(inp2, ev.Rune())
				// case tcell.KeyDown:
				// TODO: long vertical widgets
				// case tcell.KeyUp:
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					if 0 < len(inp1) {
						inp1 = inp1[:len(inp1)-1]
					}
					if 0 < len(inp2) {
						inp2 = inp2[:len(inp2)-1]
					}
				}
			case *tcell.EventMouse:
				switch ev.Buttons() {
				case tcell.WheelUp:
					offsetLine--
					if offsetLine < 0 {
						offsetLine = 0
					}
					// t--
				case tcell.WheelDown:
					offsetLine++
					// t++
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}()

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 50):
		}
		// action
		screen.Clear()

		if w, h := screen.Size(); 0 < w && 0 < h {
			Text("Пример текста. Hello world")
			Input(inp1)
			for l := 0; l < 50; l++ {
				Text(fmt.Sprintf("Input fields: %d", l))
				Input(inp2)
				// InputUint(&is)
				// InputFloat(&fl)
			}
		}

		screen.Show()
		line = 0
	}

	screen.Fini()
}
