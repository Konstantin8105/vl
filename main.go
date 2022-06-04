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

var (
	screen tcell.Screen
	line int
)

// Text("...")
func Text(str string) {
	defer func() {
		line++
	}()
	c := 0 // counter
	for _, r := range str {
		screen.SetCell(c, line, StyleDefault, r)
		c++
	}
}

// Input(&str)
func Input(str *string) {
	defer func() {
		line++
	}()
	width, _ := screen.Size()
	c := 0 // counter
	for _, r := range *str {
		screen.SetCell(c, line, StyleInput, r)
		c++
	}
	for ; c < width; c++ {
		screen.SetCell(c, line, StyleInput, ' ')
	}
}

// InputUint(&is)
func InputUint(is *uint32) {
	var s string = fmt.Sprintf("%d", *is)
	Input(&s)
}

// InputFloat(&x)
func InputFloat(fl *float32) {
	var s string = fmt.Sprintf("%f", *fl)
	Input(&s)
}

func box(screen tcell.Screen) {

	// var x float32
	//
	// var cb uint16
	// ComboBox(&cb, []widgets{...})
	//
	// var but bool
	// Button(&but, &str)

	w, h := screen.Size()

	if w == 0 || h == 0 {
		return
	}

	Text("Пример текста. Hello world")
	Input(&inp)
	InputUint(&is)
	InputFloat(&fl)
}

var (
	inp string  = "input"
	is  uint32  = 233
	fl  float32 = 0.444
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
				}
			case *tcell.EventMouse:
				switch ev.Buttons() {
				// case tcell.WheelUp:
				// 	t--
				// case tcell.WheelDown:
				// 	t++
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
		box(screen)
		screen.Show()
		line = 0 
	}

	screen.Fini()
}
