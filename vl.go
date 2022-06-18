package vl

import (
	"github.com/Konstantin8105/tf"
	"github.com/gdamore/tcell/v2"
)

var (
	ScreenStyle tcell.Style
	TextStyle   tcell.Style
)

type Drawer = func(row, col uint, s tcell.Style, r rune)

type Widget interface {
	Focus(focus bool)
	Render(width uint, dr Drawer) (height uint)
	Event(ev tcell.Event)
}

// Template for widgets:
//
//	func (...) Focus(focus bool) {}
//	func (...) Render(width uint, dr Drawer) (height int) {}
//	func (...) Event(ev tcell.Event) {}

type Coordinate struct{ Row, Col int }

///////////////////////////////////////////////////////////////////////////////

type Screen struct {
	Width  uint
	Height uint
	Root   Widget
}

// ignore
func (b *Screen) Focus(focus bool) {}

func (b *Screen) Render(width uint, dr Drawer) (height uint) {
	if width == 0 {
		return
	}
	// draw default spaces
	var col, row uint
	for col = 0; col < b.Width; col++ {
		for row = 0; row < b.Height; row++ {
			dr(row, col, ScreenStyle, ' ')
		}
	}
	// draw root widget
	draw := func(row, col uint, s tcell.Style, r rune) {
		if b.Height <= row {
			return
		}
		if b.Width <= col {
			return
		}
		dr(row, col, s, r)
	}
	if b.Root != nil {
		_ = b.Root.Render(width, draw) // ignore height
	}
	return b.Height
}

// ignore
func (b *Screen) Event(ev tcell.Event) {
	if b.Root != nil {
		b.Root.Event(ev)
	}
}

///////////////////////////////////////////////////////////////////////////////

type Text struct {
	content tf.TextField
}

func TextStatic(str string) *Text {
	t := new(Text)
	t.content.Text = []rune(str)
	return t
}

func (t *Text) SetText(str string) {
	t.content.Text = []rune(str)
	t.content.NoUpdate = false
}

// ignore any actions
func (t *Text) Focus(focus bool) {}

func (t *Text) Render(width uint, dr Drawer) (height uint) {
	draw := func(row, col uint, r rune) {
		dr(row, col, TextStyle, r)
	}
	if !t.content.NoUpdate {
		t.content.SetWidth(uint(width))
	}
	height = t.content.Render(draw, nil)
	return
}

// ignore any actions
func (t *Text) Event(ev tcell.Event) {}

///////////////////////////////////////////////////////////////////////////////

type Scroll struct {
	size struct {
		heights []uint
		width   uint
	}
	offset uint
	ws     []Widget
	focus  bool
}

func (sc Scroll) heightSumm() uint {
	var s uint = 0
	for _, h := range sc.size.heights {
		s += h
	}
	return s
}

func (sc *Scroll) Focus(focus bool) {
	sc.focus = focus
}

func (sc *Scroll) Render(width uint, dr Drawer) (height uint) {
	draw := func(row, col uint, st tcell.Style, r rune) {
		rowI := int(row) + int(height) - int(sc.offset)
		if rowI < 0 {
			return
		}
		dr(uint(rowI), col, st, r)
	}
	if len(sc.size.heights) != len(sc.ws) {
		sc.size.heights = make([]uint, len(sc.ws)+1)
	}
	sc.size.heights[0] = 0
	for i := range sc.ws {
		if sc.ws[i] == nil {
			continue
		}
		height += sc.ws[i].Render(width, draw)
		sc.size.heights[i+1] = height
	}
	return
}

func (sc *Scroll) Event(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		switch ev.Buttons() {
		case tcell.WheelUp:
			if sc.offset == 0 {
				break
			}
			sc.offset--
			return
		case tcell.WheelDown:
			sc.offset++
			const minViewLines uint = 2 // constant
			h := sc.heightSumm()
			if h < minViewLines {
				break
			}
			var maxOffset uint = h - minViewLines
			if maxOffset < sc.offset {
				sc.offset = maxOffset
			}
			return
		default:
			col, row := ev.Position()
			if col < 0 {
				return
			}
			if col < int(sc.size.width) {
				return
			}
			row += int(sc.offset)
			if row < 0 {
				return
			}
			// unfocus
			for i := range sc.ws {
				if w := sc.ws[i]; w != nil {
					w.Focus(false)
				}
			}
			// find focus widget
			for i := range sc.size.heights {
				if i == 0 {
					continue
				}
				if sc.size.heights[i-1] <= uint(row) && uint(row) < sc.size.heights[i] {
					row -= int(sc.size.heights[i-1])
					i--
					sc.Focus(true)
					if sc.ws[i] != nil {
						sc.ws[i].Focus(true)
						sc.ws[i].Event(tcell.NewEventMouse(
							col, row,
							ev.Buttons(),
							ev.Modifiers()))
					}
					return
				}
			}
		}
	case *tcell.EventKey:
		for i := range sc.ws {
			if w := sc.ws[i]; w != nil {
				w.Event(ev)
			}
		}
	}
}

func (sc *Scroll) Add(w Widget) {
	sc.ws = append(sc.ws, w)
}

///////////////////////////////////////////////////////////////////////////////
