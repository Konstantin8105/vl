//go:build ignore

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

var StyleFocus = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorYellow)

var StyleInput = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorBlue)

const MinWidth = 50

type VerticalScroll struct {
	position int16
	ws       []Widget
}

type Widget interface {
	GetId() uint16
	SetWidth() uint16
	GetHeight() uint16
	GetContent(x, y int) (mainc rune, combc []rune, style tcell.Style, width int)
}

var (
	idstart uint16
	widgets []Widget
	screen  tcell.Screen
	line    int

	focusId int

// 	offsetLine int
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
// func Text(str string) {
// 	defer func() {
// 		line++
// 	}()
// 	posLine := line - offsetLine
// 	c := 0 // counter
// 	for _, r := range str {
// 		screen.SetCell(c, posLine, StyleDefault, r)
// 		c++
// 	}
// }

type Text struct {
	id    int
	label *[]rune
}

func NewText(label *[]rune, id int) *Text {
	return &(Text{label: label, id: id})
}

type drawer = func(row, col int, st tcell.Style, r rune)

func (t *Text) Draw(width int, draw drawer) (height int) {
	pos := 0
	row := 0
	var st tcell.Style = StyleDefault
	if t.id == focusId {
		st = StyleFocus
	}
	for {
		if len(*t.label) <= pos {
			break
		}
		col := 0
		for ; pos < len(*t.label); pos++ {
			if (*t.label)[pos] == '\n' {
				pos++
				break
			}
			if col == width {
				break
			}
			draw(row, col, st, (*t.label)[pos])
			col++
		}
		row++
	}
	return row
}

// Input(&str)
// func Input(str []rune) {
// 	defer func() {
// 		line++
// 	}()
// 	width, _ := screen.Size()
// 	posLine := line - offsetLine
// 	c := 0 // counter
// 	for _, r := range str {
// 		screen.SetCell(c, posLine, StyleInput, r)
// 		c++
// 	}
//
// 	blink := StyleInput.Reverse(true) // Blink(true)
// 	screen.SetCell(c, posLine, blink, '|')
// 	c++
//
// 	for ; c < width; c++ {
// 		screen.SetCell(c, posLine, StyleInput, ' ')
// 	}
// }

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
	inp1 []rune  = []rune("Not onwqd jkhd kljahdfkljashd flkjsdhf klajsdhf aklsjdh flkasjdh lkasdjhf kjasdhf lkjasdh flkasjdh falksdjh falksdh fads fads fasd fa\nNew line\nSecond")
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

	offsetLine := 0

	var mouse struct {
		x, y int
	}

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
				case tcell.Button1: // Left mouse
					mouse.x, mouse.y = ev.Position()
					mouse.y += offsetLine
					focusId++
					if 2 < focusId {
						focusId = 0
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}()

	cnt := 0
	dur := time.Duration(0)

	draw := func(row, col int, st tcell.Style, r rune) {
		row += line - offsetLine
		// 		if mouse.y == row{
		// 			focusId=2
		// 		}
		//
		screen.SetCell(col, row, st, r)
	}

	id := 0

	GetId := func() int {
		id++
		return id
	}

	t1 := NewText(&inp1, GetId())
	t2 := NewText(&inp2, GetId())

loop:
	for {
		start := time.Now()
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 50):
			// 50 ms :  20 fps
			//  5 ms : 150 fps
			//  1 ms : 500 fps
		}

		// action
		screen.Clear()

		if width, h := screen.Size(); 0 < width && 0 < h {
			for _, w := range []interface {
				Draw(width int, dr drawer) (h int)
			}{
				t1, t2,
			} {
				if width < MinWidth {
					width = MinWidth
				}
				h := w.Draw(width, draw)
				line += h
			}

			for i := 0; i < 50; i++ {
				fps := []rune(fmt.Sprintf("FPS : %.3f\n", (float64(cnt) / dur.Seconds())))
				h3 := NewText(&fps, 0).Draw(width, draw)
				line += h3
			}
			//Text("Пример текста. Hello world")
			// 			Input(inp1)
			// 			for l := 0; l < 50; l++ {
			// 				//	Text(fmt.Sprintf("Input fields: %d", l))
			// 				Input(inp2)
			// 				// InputUint(&is)
			// 				// InputFloat(&fl)
			// 			}
		}

		screen.Show()
		line = 0

		dur += time.Now().Sub(start)
		cnt++
	}

	screen.Fini()
	fmt.Printf("Amount frames: %d frames\n", cnt)
	fmt.Printf("Duration     : %.5f second\n", dur.Seconds())
	fmt.Printf("FPS          : %.1f\n", (float64(cnt) / dur.Seconds()))
}
