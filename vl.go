package vl

import (
	"github.com/Konstantin8105/tf"
	"github.com/gdamore/tcell/v2"
)

var (
	ScreenStyle        tcell.Style
	TextStyle          tcell.Style
	ButtonStyle        tcell.Style
	ButtonFocusStyle   tcell.Style
	InputboxStyle      tcell.Style
	InputboxFocusStyle tcell.Style
)

type Drawer = func(row, col uint, s tcell.Style, r rune)

func PrintDrawer(row, col uint, s tcell.Style, dr Drawer, rs []rune) {
	for i := range rs {
		dr(row, col+uint(i), s, rs[i])
	}
}

type Widget interface {
	Focus(focus bool)
	Render(width uint, dr Drawer) (height uint)
	Event(ev tcell.Event)
}

// Template for widgets:
//
//	func (...) Focus(focus bool) {}
//	func (...) Render(width uint, dr Drawer) (height uint) {}
//	func (...) Event(ev tcell.Event) {}

///////////////////////////////////////////////////////////////////////////////

type Screen struct {
	Width  uint
	Height uint
	Root   Widget
}

func (screen *Screen) Render(width uint, dr Drawer) (height uint) {
	if width == 0 {
		return
	}
	// draw default spaces
	var col, row uint
	for col = 0; col < screen.Width; col++ {
		for row = 0; row < screen.Height; row++ {
			dr(row, col, ScreenStyle, ' ')
		}
	}
	// draw root widget
	draw := func(row, col uint, s tcell.Style, r rune) {
		if screen.Height <= row {
			return
		}
		if screen.Width <= col {
			return
		}
		dr(row, col, s, r)
	}
	if screen.Root != nil {
		_ = screen.Root.Render(width, draw) // ignore height
	}
	return screen.Height
}

func (screen *Screen) Event(ev tcell.Event) {
	if screen.Root != nil {
		screen.Root.Event(ev)
	}
}

///////////////////////////////////////////////////////////////////////////////

type Text struct {
	container
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

func (t *Text) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		t.set(width, height)
	}()
	draw := func(row, col uint, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row, col, TextStyle, r)
	}
	if !t.content.NoUpdate {
		t.content.SetWidth(width)
	}
	height = t.content.Render(draw, nil) // nil - not view cursor
	return
}

///////////////////////////////////////////////////////////////////////////////

type Scroll struct {
	offset uint
	Root   Widget

	size struct {
		height uint
		width  uint
	}
}

func (sc *Scroll) Focus(focus bool) {
	if sc.Root == nil {
		return
	}
	sc.Root.Focus(focus)
}

func (sc *Scroll) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		sc.size.height = height
		sc.size.width = width
	}()
	if sc.Root == nil {
		return
	}
	draw := func(row, col uint, st tcell.Style, r rune) {
		if width < col {
			return
		}
		if row < sc.offset {
			return
		}
		row -= sc.offset
		dr(row, col, st, r)
	}
	height = sc.Root.Render(width, draw)
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
		case tcell.WheelDown:
			sc.offset++
			const minViewLines uint = 2 // constant
			h := sc.size.height
			if h < minViewLines {
				break
			}
			var maxOffset uint = h - minViewLines
			if maxOffset < sc.offset {
				sc.offset = maxOffset
			}
		default:
			// unfocus
			sc.Focus(false)
			col, row := ev.Position()
			if col < 0 {
				return
			}
			if int(sc.size.width) < col {
				return
			}
			row = row + int(sc.offset)
			if row < 0 {
				return
			}
			if sc.Root == nil {
				return
			}
			sc.Focus(true)
			sc.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))
		}
	case *tcell.EventKey:
		if sc.Root == nil {
			return
		}
		sc.Root.Event(ev)
	}
}

///////////////////////////////////////////////////////////////////////////////

type List struct {
	size struct {
		heights []uint
		width   uint
	}
	ws    []Widget
	focus bool
}

func (l *List) Focus(focus bool) {
	l.focus = focus
}

func (l *List) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		l.size.width = width
	}()
	draw := func(row, col uint, st tcell.Style, r rune) {
		if width < col {
			return
		}
		row += height
		dr(row, col, st, r)
	}
	if len(l.size.heights) != len(l.ws) {
		l.size.heights = make([]uint, len(l.ws)+1)
	}
	l.size.heights[0] = height
	for i := range l.ws {
		if l.ws[i] == nil {
			l.size.heights[i+1] = height
			continue
		}
		height += l.ws[i].Render(width, draw)
		l.size.heights[i+1] = height
	}
	return
}

func (l *List) Event(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		// unfocus
		l.Focus(false)
		for i := range l.ws {
			if w := l.ws[i]; w != nil {
				w.Focus(false)
			}
		}
		col, row := ev.Position()
		if col < 0 {
			return
		}
		if int(l.size.width) < col {
			return
		}
		if row < 0 {
			return
		}
		// find focus widget
		for i := range l.size.heights {
			if i == 0 {
				continue
			}
			if l.size.heights[i-1] <= uint(row) &&
				uint(row) < l.size.heights[i] {
				row -= int(l.size.heights[i-1])
				i--
				l.Focus(true)
				if l.ws[i] == nil {
					continue
				}
				l.ws[i].Focus(true)
				l.ws[i].Event(tcell.NewEventMouse(
					col, row,
					ev.Buttons(),
					ev.Modifiers()))
				return
			}
		}
	case *tcell.EventKey:
		for i := range l.ws {
			if w := l.ws[i]; w != nil {
				w.Event(ev)
			}
		}
	}
}

func (l *List) Add(w Widget) {
	l.ws = append(l.ws, w)
}

///////////////////////////////////////////////////////////////////////////////

// Button examples:
// Minimal width:
//	[  ]
// Single text:
//	[ Text ] Button
// Long text:
//	[ Text                ] Button
// Multiline text:
//	[ Line 1              ] Button
//	[ Line 2              ]
type Button struct {
	Text
	OnClick func()
}

func (b *Button) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		b.set(width, height)
	}()
	// default style
	st := ButtonStyle
	if b.focus {
		st = ButtonFocusStyle
	}
	// show button row
	var emptyLines int = -1
	showRow := func(row uint) {
		if int(row) <= emptyLines {
			return
		}
		// draw empty button
		var i uint
		for i = 0; i < width; i++ {
			dr(row, i, st, ' ')
		}
		dr(row, 0, st, '[')
		dr(row, width-1, st, ']')
		emptyLines = int(row)
	}
	// constant
	const buttonOffset = 2
	if width < 2*buttonOffset {
		width = 2 * buttonOffset
	}
	// draw runes
	draw := func(row, col uint, r rune) {
		if width < col {
			return
		}
		// draw empty lines
		var i uint
		for i = 0; i <= row; i++ {
			showRow(i)
		}
		// draw symbol
		dr(row, col+buttonOffset, st, r)
	}
	// update content
	if !b.content.NoUpdate {
		b.content.SetWidth(width - 2*buttonOffset)
	}
	height = b.content.Render(draw, nil)
	return
}

func (b *Button) Event(ev tcell.Event) {
	focus, mouse, ok := b.onFocus(ev)
	if ok {
		b.Focus(focus)
	}
	if mouse[0] && b.OnClick != nil {
		b.OnClick()
	}
}

///////////////////////////////////////////////////////////////////////////////

// Frame examples:
//	+- Header ---------+
//	|      Root        |
//	+------------------+
type Frame struct {
	container

	Header Widget
	Root   Widget
	offset struct {
		row uint // vertical root offset
		col uint // horizontal root offset
	}
}

func (f *Frame) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		f.set(width, height)
	}()
	if width < 4 {
		return 1
	}
	// draw frame
	drawRow := func(row uint) {
		var i uint
		for i = 0; i < width; i++ {
			if f.focus {
				dr(row, i, TextStyle, '-')
			} else {
				dr(row, i, TextStyle, '=')
			}
		}
	}
	drawRow(0)
	defer func() {
		drawRow(height)
		var r uint
		for r = 0; r < height; r++ {
			dr(r, 0, TextStyle, '|')
			dr(r, width-1, TextStyle, '|')
		}
		dr(0, 0, TextStyle, '+')
		dr(0, width-1, TextStyle, '+')
		dr(height, 0, TextStyle, '+')
		dr(height, width-1, TextStyle, '+')
		height++
	}()
	// draw text
	if f.Header != nil {
		draw := func(row, col uint, st tcell.Style, r rune) {
			if width < col {
				panic("Text width")
			}
			dr(row, col+2, st, r)
		}
		height = f.Header.Render(width-4, draw)
	} else {
		height = 1
	}
	f.offset.row = height
	f.offset.col = 1
	// draw root widget
	droot := func(row, col uint, s tcell.Style, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row+height, col+1, s, r)
	}
	if f.Root != nil {
		height += f.Root.Render(width-2, droot)
	}
	return
}

func (f *Frame) Event(ev tcell.Event) {
	focus, _, ok := f.onFocus(ev)
	if ok {
		f.Focus(focus)
	}
	if focus && f.Root != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			// recalculate position of mouse
			col, row := ev.Position()
			if col <= int(f.offset.col) {
				return
			}
			col -= int(f.offset.col)
			if row <= int(f.offset.row) {
				return
			}
			row -= int(f.offset.row)
			f.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			f.Root.Event(ev)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type radio struct {
	Text
	choosed bool
}

func (r *radio) Render(width uint, dr Drawer) (height uint) {
	if width < 6 {
		return 1
	}
	const banner = 5
	if r.choosed {
		PrintDrawer(0, 0, TextStyle, dr, []rune(" (*) "))
	} else {
		PrintDrawer(0, 0, TextStyle, dr, []rune(" ( ) "))
	}
	if !r.content.NoUpdate {
		r.content.SetWidth(width - banner)
	}
	draw := func(row, col uint, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row, col+banner, TextStyle, r)
	}
	height = r.content.Render(draw, nil)
	if height < 2 {
		height = 1
	}
	return
}

func (r *radio) Event(ev tcell.Event) {
	// ignore
}

// Radio - button with single choose
// Example:
//	(0) choose one
//	( ) choose two
type RadioGroup struct {
	list List
	pos  uint
}

func (rg *RadioGroup) SetText(ts []string) {
	for i := range ts {
		var r radio
		r.content.Text = []rune(ts[i])
		r.content.NoUpdate = false
		rg.list.Add(&r)
	}
}

func (rg *RadioGroup) GetPos() uint {
	return rg.pos
}

func (rg *RadioGroup) Focus(focus bool) {
	// ignore
}

func (rg *RadioGroup) Render(width uint, dr Drawer) (height uint) {
	if len(rg.list.ws) <= int(rg.pos) {
		rg.pos = 0
	}
	for i := range rg.list.ws {
		if uint(i) == rg.pos {
			rg.list.ws[i].(*radio).choosed = true
			continue
		}
		rg.list.ws[i].(*radio).choosed = false
	}
	height = rg.list.Render(width, dr)
	return
}

func (rg *RadioGroup) Event(ev tcell.Event) {
	rg.list.Event(ev)
	if rg.list.focus {
		// change radio position
		for i := range rg.list.ws {
			if rg.list.ws[i].(*radio).focus {
				rg.pos = uint(i)
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// Widget : CheckBox
// Design : [V] Option

type CheckBox struct {
	Checked bool
	Text
}

func (ch *CheckBox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		ch.width = width
		ch.height = height
	}()
	if width < 6 {
		return 1
	}
	const banner = 5
	if ch.Checked {
		PrintDrawer(0, 0, TextStyle, dr, []rune(" [v] "))
	} else {
		PrintDrawer(0, 0, TextStyle, dr, []rune(" [ ] "))
	}
	if !ch.content.NoUpdate {
		ch.content.SetWidth(width - banner)
	}
	draw := func(row, col uint, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row, col+banner, TextStyle, r)
	}
	height = ch.content.Render(draw, nil)
	if height < 2 {
		height = 1
	}
	return
}

func (ch *CheckBox) Event(ev tcell.Event) {
	focus, mouse, ok := ch.onFocus(ev)
	if !ok {
		ch.Focus(focus)
	}
	if ch.focus && mouse[0] {
		ch.Checked = !ch.Checked
	}
}

///////////////////////////////////////////////////////////////////////////////

type Inputbox struct {
	Text
}

var Cursor rune = '█'

func (in *Inputbox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		in.set(width, height)
	}()
	st := InputboxStyle
	if in.focus {
		st = InputboxFocusStyle
	}
	draw := func(row, col uint, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row, col, st, r)
	}
	cur := func(row, col uint) {
		if width < col {
			panic("Text width")
		}
		dr(row, col, st, Cursor)
	}
	if !in.content.NoUpdate {
		in.content.SetWidth(width)
	}
	if !in.focus {
		cur = nil // hide cursor for not-focus inputbox
	}
	height = in.content.Render(draw, cur)
	return
}

func (in *Inputbox) Event(ev tcell.Event) {
	focus, _, ok := in.onFocus(ev)
	if ok {
		in.Focus(focus)
	}
	if !focus {
		return
	}
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		// recalculate position of mouse
		col, row := ev.Position()
		if col < 0 {
			return
		}
		if int(in.width) < col {
			return
		}
		if row < 0 {
			return
		}
		in.content.CursorPosition(uint(row), uint(col))
		return
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyUp:
			in.content.CursorMoveUp()
		case tcell.KeyDown:
			in.content.CursorMoveDown()
		case tcell.KeyLeft:
			in.content.CursorMoveLeft()
		case tcell.KeyRight:
			in.content.CursorMoveRight()
		case tcell.KeyEnter:
			in.content.Insert('\n')
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			in.content.KeyBackspace()
		case tcell.KeyDelete:
			in.content.KeyDel()
		default:
			in.content.Insert(ev.Rune())
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// Widget : Combobox
// Design :
// +-------------------+
// |                   |
// |                   |
// +-------------------+

///////////////////////////////////////////////////////////////////////////////

// Widget : CollapsingHeader

///////////////////////////////////////////////////////////////////////////////

// Widget: Horizontal list

///////////////////////////////////////////////////////////////////////////////

// Widget: Table

///////////////////////////////////////////////////////////////////////////////

// Widget: Tabs

///////////////////////////////////////////////////////////////////////////////

// Widget: Menu

///////////////////////////////////////////////////////////////////////////////

// Widget: ContextMenu

///////////////////////////////////////////////////////////////////////////////

// Tree examples:
//	Main
//	|
//	+-+ Node 0
//	| |
//	| +- Node 01
//	| |
//	| +- Node 02
//	|
//	+-+ Node 1
//	  |
//	  +- Node 01
//	  |
//	  +- Node 02
// type Tree struct {
// 	open  bool
// 	Name  Widget
// 	Nodes []Tree
// }

///////////////////////////////////////////////////////////////////////////////

// Widget: ModalDialog

///////////////////////////////////////////////////////////////////////////////

///////////////////////////////////////////////////////////////////////////////

type container struct {
	focus  bool
	width  uint
	height uint
}

func (c *container) Focus(focus bool) {
	c.focus = focus
}

func (c *container) set(width, height uint) {
	c.width = width
	c.height = height
}

func (c *container) Event(ev tcell.Event) {
	focus, _, ok := c.onFocus(ev)
	if ok {
		c.Focus(focus)
	}
}

func (c *container) onFocus(ev tcell.Event) (focus bool, button [3]bool, ok bool) {
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		// check on focus
		col, row := ev.Position()
		if col < 0 {
			break
		}
		if int(c.width) < col {
			break
		}
		if row < 0 {
			break
		}
		switch ev.Buttons() {
		case tcell.Button1:
			button[0] = true // Left mouse button
			focus = true     // focus
		case tcell.Button3:
			button[1] = true // Middle mouse button
			focus = true     // focus
		case tcell.Button2:
			button[2] = true // Right mouse button
			focus = true     // focus
		}
		ok = true
	}
	return
}
