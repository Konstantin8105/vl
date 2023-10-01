package vl

import (
	"fmt"
	"sync"
	"time"

	"github.com/Konstantin8105/tf"
	"github.com/gdamore/tcell/v2"
)

func style(fd, bd tcell.Color) tcell.Style {
	return tcell.StyleDefault.Foreground(fd).Background(bd)
}

var (
	white  = tcell.ColorWhite
	yellow = tcell.ColorYellow
	focus  = tcell.ColorDeepPink // ColorDeepPink
	red    = tcell.ColorRed
	green  = tcell.ColorGreen
	black  = tcell.ColorBlack

	ScreenStyle        tcell.Style = style(black, white)
	TextStyle          tcell.Style = ScreenStyle
	ButtonStyle        tcell.Style = style(black, yellow)
	ButtonFocusStyle   tcell.Style = style(black, focus)
	InputboxStyle      tcell.Style = style(black, yellow)
	InputboxFocusStyle tcell.Style = style(black, focus)
	// cursor
	CursorStyle tcell.Style = style(white, red)
	// select
	InputboxSelectStyle tcell.Style = style(black, green)
)

///////////////////////////////////////////////////////////////////////////////

// specific symbols for borders
// default symbol '-' if not initialized symbols
var (
	LineHorizontalFocus    rune = '-'
	LineHorizontalUnfocus       = '-'
	LineVerticalFocus           = '-'
	LineVerticalUnfocus         = '-'
	CornerLeftUpFocus           = '-'
	CornerLeftDownFocus         = '-'
	CornerRightUpFocus          = '-'
	CornerRightDownFocus        = '-'
	CornerLeftUpUnfocus         = '-'
	CornerLeftDownUnfocus       = '-'
	CornerRightUpUnfocus        = '-'
	CornerRightDownUnfocus      = '-'
	ScrollLine                  = '-'
	ScrollUp                    = '-'
	ScrollDown                  = '-'
	ScrollSquare                = '-'
	TreeUpDown                  = '-'
	TreeUp                      = '-'
)

func init() {
	SpecificSymbol(true)
}

func SpecificSymbol(ascii bool) {
	for _, v := range []struct {
		r       *rune
		acsii   rune
		unicode rune
	}{
		{&LineHorizontalFocus, '=', '\u2550'},
		{&LineHorizontalUnfocus, '-', '\u2500'},
		{&LineVerticalFocus, 'I', '\u2551'},
		{&LineVerticalUnfocus, '|', '\u2502'},
		{&CornerLeftUpFocus, '+', '\u2554'},
		{&CornerLeftDownFocus, '+', '\u255A'},
		{&CornerRightUpFocus, '+', '\u2557'},
		{&CornerRightDownFocus, '+', '\u255D'},
		{&CornerLeftUpUnfocus, '+', '\u250C'},
		{&CornerLeftDownUnfocus, '+', '\u2514'},
		{&CornerRightUpUnfocus, '+', '\u2510'},
		{&CornerRightDownUnfocus, '+', '\u2518'},
		{&ScrollLine, '|', '\u2506'},
		{&ScrollUp, '-', '\u252C'},
		{&ScrollDown, '-', '\u2534'},
		{&ScrollSquare, '*', '\u25A0'},
		{&TreeUpDown, '+', '\u251D'},
		{&TreeUp, '+', '\u2514'},
	} {
		if ascii {
			*v.r = v.acsii
			continue
		}
		*v.r = v.unicode
	}
}

///////////////////////////////////////////////////////////////////////////////

type Drawer = func(row, col uint, s tcell.Style, r rune)

func PrintDrawer(row, col uint, s tcell.Style, dr Drawer, rs []rune) {
	for i := range rs {
		dr(row, col+uint(i), s, rs[i])
	}
}

type Offset struct {
	row uint // vertical root offset
	col uint // horizontal root offset
}

// WidgetV is widget with vertical fix height
type VerticalFix interface {
	SetHeight(hmax uint)
}

type Widget interface {
	Focus(focus bool)
	Render(width uint, dr Drawer) (height uint)
	Set(width, height uint)
	Event(ev tcell.Event)
}

// Template for widgets:
//
//	func (...) Focus(focus bool) {}
//	func (...) Render(width uint, dr Drawer) (height uint) {}
//	func (...) Set(width, height uint) {}
//	func (...) Event(ev tcell.Event) {}

///////////////////////////////////////////////////////////////////////////////

type Cell struct {
	S tcell.Style
	R rune
}

type Screen struct {
	containerVerticalFix
	Root Widget
	//	dialog struct {
	//		Root             Widget
	//		offsetX, offsetY uint
	//	}
}

func (screen *Screen) GetContents(width uint, cells *[][]Cell) {
	var ar, ac uint
	drawer := func(row, col uint, s tcell.Style, r rune) {
		for i := len(*cells); i <= int(row); i++ {
			*cells = append(*cells, make([]Cell, 0))
		}
		for i := len((*cells)[row]); i <= int(col); i++ {
			(*cells)[row] = append((*cells)[row], Cell{R: ' '})
		}
		if ar < row {
			ar = row
		}
		if ac < col {
			ac = col
		}
		(*cells)[row][col] = Cell{S: s, R: r}
	}
	_ = screen.Render(width, drawer) // ignore height
	// resize cells matrix
	if len(*cells)-1 != int(ar) {
		*cells = (*cells)[:int(ar)]
	}
	for i := range *cells {
		if len((*cells)[i])-1 != int(ac) {
			(*cells)[i] = (*cells)[i][:int(ac)]
		}
	}
	return
}

func Convert(cells [][]Cell) string {
	var str string
	var w int
	for r := range cells {
		str += fmt.Sprintf("%09d|", r+1)
		for c := range cells[r] {
			str += string(cells[r][c].R)
		}
		if width := len(cells[r]); w < width {
			w = width
		}
		str += fmt.Sprintf("| width:%09d\n", len(cells[r]))
	}
	str += fmt.Sprintf("rows  = %3d\n", len(cells))
	str += fmt.Sprintf("width = %3d\n", w)
	return str
}

func (screen *Screen) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		screen.Set(width, height)
	}()
	if width == 0 {
		return
	}
	// draw default spaces
	var col, row uint
	for col = 0; col < width; col++ {
		for row = 0; row < screen.hmax; row++ {
			dr(row, col, ScreenStyle, ' ')
		}
	}
	// draw root widget
	draw := func(row, col uint, s tcell.Style, r rune) {
		if screen.hmax <= row {
			return
		}
		if width <= col {
			return
		}
		dr(row, col, s, r)
	}
	if screen.Root != nil {
		_ = screen.Root.Render(width, draw) // ignore height
	}
	// draw dialog
	// if d := screen.dialog.Root; d != nil {
	// 	_ = d.Render(width, draw)
	// 	if c, ok := d.(*container); ok {
	// 		screen.dialog.offsetX = (width - c.width) / 2
	// 		screen.dialog.offsetY = (screen.hmax - c.height) / 2
	// 	}
	// }
	return screen.hmax
}

func (screen *Screen) SetHeight(hmax uint) {
	screen.containerVerticalFix.SetHeight(hmax)
	if screen.Root != nil {
		if _, ok := screen.Root.(VerticalFix); ok {
			screen.Root.(VerticalFix).SetHeight(hmax)
		}
	}
	//	if screen.dialog.Root != nil {
	//		if _, ok := screen.dialog.Root.(VerticalFix); ok {
	//			screen.dialog.Root.(VerticalFix).SetHeight(hmax)
	//		}
	//	}
}

func (screen *Screen) Event(ev tcell.Event) {
	if screen.Root == nil {
		return
	}
	// if screen.dialog.Root != nil {
	// 	screen.dialog.Root.Event(ev)
	// 	return
	// }
	screen.Root.Event(ev)
}

// func (screen *Screen) Close() {
// 	screen.dialog.Root = nil
// }
//
// func (screen *Screen) AddDialog(name string, dialog Widget) {
// 	var frame Frame
// 	frame.Header = TextStatic(name)
// 	var list List
// 	if dialog != nil {
// 		list.Add(dialog)
// 	}
// 	var btn Button
// 	btn.Compress = true
// 	btn.OnClick = func() {
// 		screen.Close()
// 	}
// 	list.Add(&btn)
// 	frame.Root = &list
// 	frame.SetHeight(screen.hmax)
// 	screen.dialog.Root = &frame
// }

///////////////////////////////////////////////////////////////////////////////

type Separator struct{ container }

func (s *Separator) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		s.Set(width, height)
	}()
	return 1
}

///////////////////////////////////////////////////////////////////////////////

type Text struct {
	container
	content tf.TextFieldLimit
}

func TextStatic(str string) *Text {
	t := new(Text)
	t.content.Text = []rune(str)
	t.content.NoUpdate = false
	return t
}

func (t *Text) SetLinesLimit(limit uint) {
	t.content.NoUpdate = false
	t.content.SetLinesLimit(limit)
}

func (t *Text) SetText(str string) {
	t.content.Text = []rune(str)
	t.content.NoUpdate = false
}

func (t *Text) GetText() string {
	return string(t.content.Text)
}

func (t *Text) Filter(f func(r rune) (insert bool)) {
	t.content.NoUpdate = false
	t.content.Filter = f
}

func (t *Text) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		t.Set(width, height)
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
	// added for colorize unvisible lines too
	h := t.content.GetRenderHeight()
	if height < h {
		height = h
	}
	return
}

// /////////////////////////////////////////////////////////////////////////////
const scrollBarWidth uint = 1

type Scroll struct {
	containerVerticalFix

	offset uint
	Root   Widget
}

func (sc *Scroll) Focus(focus bool) {
	if sc.Root == nil {
		return
	}
	sc.container.Focus(focus)
	sc.Root.Focus(focus)
}

func (sc *Scroll) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		sc.Set(width, height)
	}()
	if sc.Root == nil {
		return
	}
	sc.fixOffset() // fix offset position
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
	if sc.addlimit {
		if width < scrollBarWidth {
			return
		}
		height = sc.Root.Render(width-scrollBarWidth, draw)
		// calculate location
		if 2 < sc.hmax {
			var value float32 // 0 ... 1
			if sc.hmax < height {
				value = float32(sc.offset) / float32(height-sc.hmax)
			} else {
				value = 1.0
			}
			if 1 < value {
				value = 1.0
			}
			if value < 0 {
				value = 0.0
			}
			st := TextStyle
			for r := uint(0); r < sc.hmax; r++ {
				dr(r, width-scrollBarWidth, st, ScrollLine)
			}
			dr(0, width-scrollBarWidth, st, ScrollUp)
			dr(sc.hmax-1, width-scrollBarWidth, st, ScrollDown)
			pos := uint(value * float32(sc.hmax-2))
			if pos == 0 {
				pos = 1
			}
			if pos == sc.hmax-scrollBarWidth {
				pos = sc.hmax - 2
			}
			dr(pos, width-scrollBarWidth, st, ScrollSquare)
		}
	} else {
		height = sc.Root.Render(width, draw)
	}
	return
}

func (sc *Scroll) fixOffset() {
	const minViewLines uint = 2 // constant
	if sc.height < minViewLines {
		return
	}
	var maxOffset uint = sc.height - minViewLines
	if 0 < sc.hmax {
		if sc.hmax < sc.height {
			if sc.height < sc.hmax+sc.offset {
				sc.offset = sc.height - sc.hmax
			}
		} else {
			sc.offset = 0
		}
	} else if maxOffset < sc.offset {
		sc.offset = maxOffset
	}
}

func (sc *Scroll) Event(ev tcell.Event) {
	if sc.Root == nil {
		return
	}

	_, ok := sc.onFocus(ev)
	if ok {
		sc.Focus(true)
	}

	if !sc.focus {
		return
	}
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		col, row := ev.Position()
		if col < 0 {
			return
		}
		if sc.width <= uint(col) {
			return
		}
		_, _ = col, row
		switch ev.Buttons() {
		case tcell.WheelUp:
			if sc.offset == 0 {
				break
			}
			sc.offset--
		case tcell.WheelDown:
			sc.offset++
		default:
			if 0 < row && 2 < sc.hmax && ev.Buttons() == tcell.Button1 &&
				col == int(sc.width-scrollBarWidth) && 0 < sc.hmax {
				ratio := float32(row-1) / float32(sc.hmax-2)
				dh := float32(sc.height)
				if 0 < dh {
					sc.offset = uint(dh * ratio)
				}
				sc.fixOffset() // fix offset position
				break
			}
			// unfocus
			sc.Focus(false)
			sc.Root.Focus(false)
			col, row := ev.Position()
			if col < 0 {
				return
			}
			if int(sc.width) < col {
				return
			}
			row = row + int(sc.offset)
			if row < 0 {
				return
			}
			sc.Focus(true)
			sc.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))
		}
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyPgDn:
			if 0 < sc.hmax {
				sc.offset += sc.hmax / 2
				sc.fixOffset() // fix offset position
			}
		case tcell.KeyPgUp:
			if 0 < sc.hmax {
				if sc.offset < sc.hmax/2 {
					sc.offset = 0
				} else {
					sc.offset -= sc.hmax / 2
				}
				sc.fixOffset() // fix offset position
			}
		default:
			sc.Root.Event(ev)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type List struct {
	containerVerticalFix

	heights []uint
	ws      []Widget

	IgnoreVerticalFix bool
}

func (l *List) Size() int {
	return len(l.ws)
}

func (l *List) Focus(focus bool) {
	if !focus {
		for i := range l.ws {
			if w := l.ws[i]; w != nil {
				w.Focus(focus)
			}
		}
	}
	l.focus = focus
}

func (l List) getItemHmax() uint {
	return l.hmax / uint(len(l.ws))
}

func (l *List) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		l.Set(width, height)
	}()
	if len(l.ws) == 0 {
		return
	}
	var heightMax uint
	if l.addlimit {
		heightMax = l.getItemHmax()
	}
	draw := func(row, col uint, st tcell.Style, r rune) {
		if width < col {
			return
		}
		if l.addlimit && heightMax <= row {
			return
		}
		row += height
		dr(row, col, st, r)
	}
	// prepare list of heights
	if len(l.heights)+1 != len(l.ws) || len(l.heights) == 0 {
		l.heights = make([]uint, len(l.ws)+1)
	}
	l.heights[0] = height // default first element is zero
	// render each widget
	for i := range l.ws {
		var h uint
		if l.ws[i] != nil {
			h = l.ws[i].Render(width, draw)
		}
		if l.addlimit {
			height += heightMax
		} else {
			height += h
		}
		l.heights[i+1] = height
	}
	return
}

func (l *List) Event(ev tcell.Event) {
	_, ok := l.onFocus(ev)
	if ok {
		l.Focus(true)
	}
	if !l.focus {
		return
	}
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
		if int(l.width) < col {
			return
		}
		if row < 0 {
			return
		}
		// find focus widget
		for i := range l.heights {
			if i == 0 {
				continue
			}
			if l.heights[i-1] <= uint(row) &&
				uint(row) < l.heights[i] {
				// row correction
				row -= int(l.heights[i-1])
				// index correction
				i--
				// focus
				l.Focus(true)
				if l.ws[i] == nil {
					continue
				}
				//l.ws[i].Focus(true)
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

func (l *List) Get(index int) Widget {
	if index < 0 || len(l.ws) <= index {
		// not valid index
		return nil
	}
	return l.ws[index]
}

func (l *List) Update(index int, w Widget) {
	if index < 0 || len(l.ws) <= index {
		// not valid index
		return
	}
	l.ws[index] = w
}

func (l *List) Add(w Widget) {
	l.ws = append(l.ws, w)
}

func (l *List) Clear() {
	l.ws = nil
	l.heights = nil
}

func (l *List) SetHeight(hmax uint) {
	if l.IgnoreVerticalFix {
		return
	}
	l.containerVerticalFix.SetHeight(hmax)
	for i := range l.ws {
		if l.ws[i] == nil {
			continue
		}
		if vf, ok := l.ws[i].(VerticalFix); ok {
			vf.SetHeight(l.getItemHmax())
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// Example of menu line:
// [ File ] [ Edit ] [ Select ] [ Groups ] [ Help ]
//
// Elements for submenu:
//	* Button
//	* Checkbox
//	* RadioGroup

type Menu struct {
	containerVerticalFix
	header ListH
	Root   Widget
}

func (menu *Menu) SetHeight(hmax uint) {
	menu.containerVerticalFix.SetHeight(hmax)
	if menu.Root != nil {
		if _, ok := menu.Root.(VerticalFix); ok {
			menu.Root.(VerticalFix).SetHeight(hmax)
		}
	}
}

func (menu *Menu) Add(nodes ...interface {
	Widget
	Compressable
}) {
	for i := range nodes {
		nodes[i].Compress()
		menu.header.Add(nodes[i])
	}
}

func (menu *Menu) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		menu.Set(width, height)
	}()
	menu.header.Compress()
	h := menu.header.Render(width, dr)
	droot := func(row, col uint, s tcell.Style, r rune) {
		if menu.hmax < row && menu.addlimit {
			return
		}
		dr(row+h, col, s, r)
	}
	height = menu.Root.Render(width, droot)
	height += h // for menu
	return
}

func (menu *Menu) Event(ev tcell.Event) {
	_, ok := menu.onFocus(ev)
	if ok {
		menu.Focus(true)
	}
	if !menu.focus {
		return
	}
	{
		// menu
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			if row < int(menu.height) {
				menu.header.Event(tcell.NewEventMouse(
					col, row,
					ev.Buttons(),
					ev.Modifiers()))
			}

		case *tcell.EventKey:
			menu.header.Event(ev)
		}
	}
	if menu.Root != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			if int(menu.height) < row {
				return
			}
			row -= int(menu.header.height)
			if row < 0 {
				return
			}
			menu.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			menu.Root.Event(ev)
		}
	}
}

// func (m *Menu) AddMenu(menu *Menu)               {
// 	m.nodes = append(m.nodes, menu)
// }

// type MenuItem struct {
// 	container
// 	w Widget
// 	a func()
// }
//
// func (m *MenuItem) Create(name string, action func()) {
// 	m.w = TextStatic(name)
// 	m.a = action
// }
//
// func (m *MenuItem) Render(width uint, dr Drawer) (height uint) {
// 	defer func() {
// 		m.width = width
// 		m.height = height
// 	}()
// 	if width < 6 {
// 		return 1
// 	}
// 	st := InputboxStyle
// 	if m.focus {
// 		st = InputboxFocusStyle
// 	} else {
// 		st = InputboxStyle
// 	}
// 	PrintDrawer(0, 0, st, dr, []rune(" "))
// 	const banner = 1
// 	draw := func(row, col uint, s tcell.Style, r rune) {
// 		if width < col {
// 			panic("Text width")
// 		}
// 		dr(row, col+banner, TextStyle, r)
// 	}
// 	if m.w != nil {
// 		height = m.w.Render(width-banner, draw)
// 	}
// 	if height < 2 {
// 		height = 1
// 	}
// 	return
//
// }
//
// func (m *MenuItem) Event(ev tcell.Event) {
// 	mouse, ok := m.onFocus(ev)
// 	if ok {
// 		m.Focus(true)
// 	} else {
// 		m.Focus(false)
// 	}
// 	if mouse[0] && m.a != nil {
// 		m.a()
// 	}
// }

///////////////////////////////////////////////////////////////////////////////

// Button examples
//
//	Minimal width:
//	[  ]
//	Single text:
//	[ Text ] Button
//	Long text:
//	[ Text                ] Button
//	 Multiline text:
//	[ Line 1              ] Button
//	[ Line 2              ]
type Button struct {
	Text
	compress
	OnClick func()
}

func (b *Button) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		b.Set(width, height)
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
	if b.compress.state {
		// added for create buttons with minimal width
		w := b.content.GetRenderWidth() + 2*buttonOffset + 2
		if w < width {
			width = w
			b.content.SetWidth(width - 2*buttonOffset)
		}
	}
	height = b.content.Render(draw, nil)
	return
}

func (b *Button) Event(ev tcell.Event) {
	mouse, ok := b.onFocus(ev)
	if ok {
		b.Focus(true)
	}
	if mouse[0] && b.OnClick != nil {
		b.OnClick()
	}
}

///////////////////////////////////////////////////////////////////////////////

// Frame examples
//
//	+- Header ---------+
//	|      Root        |
//	+------------------+
type Frame struct {
	containerVerticalFix

	Header       Widget
	offsetHeader Offset
	Root         Widget
	offsetRoot   Offset
}

func (f *Frame) Focus(focus bool) {
	if !focus {
		if w := f.Header; w != nil {
			w.Focus(focus)
		}
		if w := f.Root; w != nil {
			w.Focus(focus)
		}
	}
	f.container.Focus(focus)
}

func (f *Frame) Render(width uint, drg Drawer) (height uint) {
	defer func() {
		f.Set(width, height)
	}()
	dr := func(row, col uint, s tcell.Style, r rune) {
		if f.hmax < row && f.addlimit {
			return
		}
		drg(row, col, s, r)
	}
	if width < 4 {
		return 1
	}
	// draw frame
	drawRow := func(row uint) {
		var i uint
		for i = 0; i < width; i++ {
			if f.focus {
				dr(row, i, TextStyle, LineHorizontalFocus)
			} else {
				dr(row, i, TextStyle, LineHorizontalUnfocus)
			}
		}
	}
	drawRow(0)
	defer func() {
		drawRow(height)
		var r uint
		for r = 0; r < height; r++ {
			if f.focus {
				dr(r, 0, TextStyle, LineVerticalFocus)
				dr(r, width-1, TextStyle, LineVerticalFocus)
			} else {
				dr(r, 0, TextStyle, LineVerticalUnfocus)
				dr(r, width-1, TextStyle, LineVerticalUnfocus)
			}
		}
		if f.focus {
			dr(0, 0, TextStyle, CornerLeftUpFocus)
			dr(0, width-1, TextStyle, CornerRightUpFocus)
			dr(height, 0, TextStyle, CornerLeftDownFocus)
			dr(height, width-1, TextStyle, CornerRightDownFocus)
		} else {
			dr(0, 0, TextStyle, CornerLeftUpUnfocus)
			dr(0, width-1, TextStyle, CornerRightUpUnfocus)
			dr(height, 0, TextStyle, CornerLeftDownUnfocus)
			dr(height, width-1, TextStyle, CornerRightDownUnfocus)
		}
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
	// add limit of height
	if f.addlimit {
		hmax := f.hmax - height - 2
		if f.Root != nil {
			if _, ok := f.Root.(VerticalFix); ok {
				f.Root.(VerticalFix).SetHeight(hmax)
			}
		}
	}
	// next step
	f.offsetRoot.row = height + 1
	f.offsetRoot.col = 2
	f.offsetHeader.row = 0
	f.offsetHeader.col = 2
	// draw root widget
	droot := func(row, col uint, s tcell.Style, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row+height+1, col+2, s, r)
	}
	if f.Root != nil {
		height += f.Root.Render(width-4, droot) + 2
	}
	if f.addlimit {
		height = f.hmax - 1
	}
	return
}

func (f *Frame) Event(ev tcell.Event) {
	_, ok := f.onFocus(ev)
	if ok {
		f.Focus(true)
	}
	if !f.focus {
		return
	}
	if f.Root != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(f.offsetRoot.col)
			row -= int(f.offsetRoot.row)
			f.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			f.Root.Event(ev)
		}
	}
	if f.Header != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(f.offsetHeader.col)
			row -= int(f.offsetHeader.row)
			f.Header.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			f.Header.Event(ev)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type radio struct {
	container

	choosed bool
	Root    Widget
}

func (r *radio) Focus(focus bool) {
	r.container.Focus(focus)
	if r.Root != nil {
		r.Root.Focus(focus)
	}
}

const banner = 4 // banner for CheckBox and radio

func (r *radio) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		r.Set(width, height)
	}()
	if width < 6 {
		return 1
	}
	st := InputboxStyle
	if r.choosed {
		st = InputboxSelectStyle
	}
	if r.focus {
		st = InputboxFocusStyle
	}
	if r.choosed {
		PrintDrawer(0, 0, st, dr, []rune("(*)"))
	} else {
		PrintDrawer(0, 0, st, dr, []rune("( )"))
	}
	if r.Root != nil {
		if ch, ok := r.Root.(*CollapsingHeader); ok {
			ch.Open(r.choosed)
		}
		droot := func(row, col uint, s tcell.Style, r rune) {
			if width < col {
				panic("Text width")
			}
			dr(row, col+banner, s, r)
		}
		height = r.Root.Render(width-banner, droot)
	}
	if height < 2 {
		height = 1
	}
	return
}

func (r *radio) Event(ev tcell.Event) {
	mouse, ok := r.onFocus(ev)
	if ok {
		r.Focus(true)
	}
	if !r.focus {
		return
	}
	if mouse[0] {
		r.Focus(true)
	}
	if r.Root != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			if col < 0 {
				return
			}
			if row < 0 {
				return
			}
			if col <= banner {
				return
			}
			col -= banner
			r.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			r.Root.Event(ev)
		}
	}
}

// Radio - button with single choose
//
//	Example:
//	(0) choose one
//	( ) choose two
type RadioGroup struct {
	container

	list     List
	pos      uint
	OnChange func()
}

func (rg *RadioGroup) Add(w Widget) {
	var r radio
	r.Root = w
	rg.list.Add(&r)
	rg.pos = uint(len(rg.list.ws) - 1)
	if f := rg.OnChange; f != nil {
		f()
	}
}

func (rg *RadioGroup) AddText(ts ...string) {
	for _, s := range ts {
		rg.Add(TextStatic(s))
	}
}

func (rg *RadioGroup) Clear() {
	rg.pos = 0
	rg.OnChange = nil
	rg.list.Clear()
}

func (rg *RadioGroup) SetPos(pos uint) {
	rg.pos = pos
	if len(rg.list.ws) <= int(rg.pos) {
		rg.pos = 0
	}
	if f := rg.OnChange; f != nil {
		f()
	}
}

func (rg *RadioGroup) GetPos() uint {
	return rg.pos
}

func (rg *RadioGroup) Focus(focus bool) {
	rg.container.Focus(focus)
	rg.list.Focus(focus)
}

func (rg *RadioGroup) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		rg.Set(width, height)
	}()
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
		last := rg.pos
		for i := range rg.list.ws {
			if rg.list.ws[i].(*radio).focus {
				rg.pos = uint(i)
			}
		}
		if last != rg.pos {
			if f := rg.OnChange; f != nil {
				f()
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// CheckBox example
//
// [v] Option
type CheckBox struct {
	pair    [2]string
	Checked bool
	Text
	OnChange func()
}

func (ch *CheckBox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		ch.width = width
		ch.height = height
	}()
	if width < 6 {
		return 1
	}
	st := InputboxStyle
	if ch.Checked {
		st = InputboxSelectStyle
	}
	if ch.focus {
		st = InputboxFocusStyle
	}
	if len(ch.pair[0]) == 0 || len(ch.pair[1]) == 0 {
		// default values
		ch.pair = [2]string{"[v]", "[ ]"}
	}
	var lenght uint = 0
	if ch.Checked {
		PrintDrawer(0, 0, st, dr, []rune(ch.pair[0]))
		lenght = uint(len(ch.pair[0]))
	} else {
		PrintDrawer(0, 0, st, dr, []rune(ch.pair[1]))
		lenght = uint(len(ch.pair[1]))
	}
	if !ch.content.NoUpdate {
		ch.content.SetWidth(width - lenght)
	}
	dr(0, lenght, TextStyle, ' ')
	draw := func(row, col uint, r rune) {
		if width < col {
			panic("Text width")
		}
		dr(row, col+lenght+1, TextStyle, r)
	}
	height = ch.content.Render(draw, nil)
	if height < 2 {
		height = 1
	}
	return
}

func (ch *CheckBox) Event(ev tcell.Event) {
	mouse, ok := ch.onFocus(ev)
	if ok {
		ch.Focus(true)
	}
	if !ch.focus {
		return
	}
	if mouse[0] {
		ch.Checked = !ch.Checked
		if f := ch.OnChange; f != nil {
			f()
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type Inputbox struct {
	Text
}

var Cursor rune = '_'

func (in *Inputbox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		in.Set(width, height)
	}()
	st := InputboxStyle
	if in.focus {
		st = InputboxFocusStyle
	}
	// default line color
	h := in.content.GetRenderHeight()
	for row := uint(0); row < h; row++ {
		// draw empty line
		for i := uint(0); i < width; i++ {
			dr(row, i, st, ' ')
		}
	}
	// draw
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
		st := CursorStyle
		dr(row, col, st, Cursor)
	}
	if !in.content.NoUpdate {
		in.content.SetWidth(width)
	}
	if !in.focus {
		cur = nil // hide cursor for not-focus inputbox
	}
	height = in.content.Render(draw, cur)
	if height < h {
		height = h
	}
	return
}

func (in *Inputbox) Event(ev tcell.Event) {
	_, ok := in.onFocus(ev)
	if ok {
		in.Focus(true)
	}
	if !in.focus {
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

type CollapsingHeader struct {
	frame Frame
	open  bool
	cb    CheckBox
	Root  Widget
	init  bool
}

func (c *CollapsingHeader) Focus(focus bool) {
	c.frame.Focus(focus)
}

func (c *CollapsingHeader) SetText(str string) {
	c.cb.SetText(str)
}

func (c *CollapsingHeader) Open(state bool) {
	c.open = state
	c.cb.Checked = state
}

func (c *CollapsingHeader) Render(width uint, dr Drawer) (height uint) {
	if !c.init {
		c.cb.pair = [2]string{"[ > ]", "[ < ]"}
		c.cb.OnChange = func() {
			c.open = !c.open
		}
		c.frame.Header = &c.cb
		c.init = true
	}
	if c.open {
		c.frame.Root = c.Root
	} else {
		c.frame.Root = nil
	}
	return c.frame.Render(width, dr)
}

func (c *CollapsingHeader) Set(width, height uint) {
	c.frame.Set(width, height)
}

func (c *CollapsingHeader) Event(ev tcell.Event) {
	c.frame.Event(ev)
}

///////////////////////////////////////////////////////////////////////////////

type listNode struct {
	w Widget
	// widths
	from int
	to   int
}

// Widget: Horizontal list
type ListH struct {
	containerVerticalFix
	compress

	nodes    []listNode
	minWidth uint
}

func (l *ListH) Focus(focus bool) {
	if !focus {
		for i := range l.nodes {
			if w := l.nodes[i].w; w != nil {
				w.Focus(focus)
			}
		}
	}
	l.focus = focus
}

func (l *ListH) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		l.Set(width, height)
	}()
	if len(l.nodes) == 0 {
		return
	}
	if l.nodes[len(l.nodes)-1].to != int(width) {
		if l.compress.state {
			for i := range l.nodes {
				// initialize sizes of widgets
				l.nodes[i].w.Render(width, nil)
			}
			l.nodes[0].from = 0
			for i := range l.nodes {
				if c, ok := l.nodes[i].w.(Compressable); ok {
					c.Compress()
				} else {
					panic(fmt.Errorf("add Compressable widget: %T", l.nodes[i].w))
				}
				if c, ok := l.nodes[i].w.(interface{ Get() (w, h uint) }); ok {
					w, _ := c.Get()
					l.nodes[i].to = l.nodes[i].from + int(w)
					if i+1 != len(l.nodes) {
						l.nodes[i+1].from = l.nodes[i].to + 1
					}
				}
			}
			l.nodes[len(l.nodes)-1].to = int(width)
		} else {
			// width of each element
			// gap 1 symbol between widgets
			dw := int(float32(width-uint(len(l.nodes)-1)) / float32(len(l.nodes)))
			if dw < int(l.minWidth) {
				dw = int(l.minWidth)
			}
			// calculate widths
			for i := range l.nodes {
				l.nodes[i].from = i * (dw + 1)
				l.nodes[i].to = l.nodes[i].from + dw
			}
			// limits of width
			for i := range l.nodes {
				if int(width) < l.nodes[i].from {
					l.nodes[i].from = int(width)
				}
				if int(width) < l.nodes[i].to {
					l.nodes[i].to = int(width)
				}
				if l.nodes[i].to < l.nodes[i].from {
					l.nodes[i].from = l.nodes[i].to
				}
			}
		}
	}
	for i := range l.nodes {
		draw := func(row, col uint, st tcell.Style, r rune) {
			if 0 < l.hmax && l.hmax < row {
				return
			}
			col += uint(l.nodes[i].from)
			dr(row, col, st, r)
		}
		if l.nodes[i].w == nil {
			continue
		}
		h := l.nodes[i].w.Render(uint(l.nodes[i].to-l.nodes[i].from), draw)
		if height < h {
			height = h
		}
	}
	return
}

func (l *ListH) SetHeight(hmax uint) {
	l.containerVerticalFix.SetHeight(hmax)
	for i := range l.nodes {
		if l.nodes[i].w == nil {
			continue
		}
		if vf, ok := l.nodes[i].w.(VerticalFix); ok {
			vf.SetHeight(hmax)
		}
	}
}

func (l *ListH) Event(ev tcell.Event) {
	_, ok := l.onFocus(ev)
	if ok {
		l.Focus(true)
	}
	if !l.focus {
		return
	}
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		// unfocus
		l.Focus(false)
		for i := range l.nodes {
			if w := l.nodes[i].w; w != nil {
				w.Focus(false)
			}
		}
		col, row := ev.Position()
		if col < 0 {
			return
		}
		if int(l.width) < col {
			return
		}
		if row < 0 {
			return
		}
		// find focus widget
		for i := range l.nodes {
			if l.nodes[i].w == nil {
				continue
			}
			if l.nodes[i].from <= col && col < l.nodes[i].to {
				// row correction
				col := col - int(l.nodes[i].from)
				// focus
				l.Focus(true)
				//l.ws[i].Focus(true)
				l.nodes[i].w.Event(tcell.NewEventMouse(
					col, row,
					ev.Buttons(),
					ev.Modifiers()))
				return
			}
		}
	case *tcell.EventKey:
		for i := range l.nodes {
			if w := l.nodes[i].w; w != nil {
				w.Event(ev)
			}
		}
	}
}

func (l *ListH) Add(w Widget) {
	l.nodes = append(l.nodes, listNode{w: w, from: 0, to: 0})
}

func (l *ListH) Clear() {
	l.nodes = nil
	l.minWidth = 0
}

func (l *ListH) MinWidth1element(width uint) {
	l.minWidth = width
}

///////////////////////////////////////////////////////////////////////////////

// Combobox example
//
//	Name03
//	+-| > | Choose: ----+
//	|                   |
//	| ( ) Name 01       |
//	| ( ) Name 02       |
//	| (*) Name 03       |
//	| ( ) Name 04       |
//	|                   |
//	+-------------------+
type Combobox struct {
	ch       CollapsingHeader
	rg       RadioGroup
	ts       []string
	OnChange func()
}

func (c *Combobox) Add(ts ...string) {
	for _, s := range ts {
		c.ts = append(c.ts, s)
		c.rg.Add(TextStatic(s))
	}
	if f := c.rg.OnChange; f != nil {
		f()
	}
}

func (c *Combobox) Clear() {
	c.rg.Clear()
	c.ts = []string{}
	c.OnChange = nil
}

func (c *Combobox) SetPos(pos uint) {
	c.rg.SetPos(pos)
	if f := c.OnChange; f != nil {
		f()
	}
	if f := c.rg.OnChange; f != nil {
		f()
	}
}

func (c *Combobox) GetPos() uint {
	return c.rg.GetPos()
}

func (c *Combobox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		c.Set(width, height)
	}()
	if width < 4 {
		return 1
	}
	if c.ch.Root == nil {
		c.ch.Root = &c.rg
		c.rg.OnChange = func() {
			if len(c.ts) == 0 {
				// empty list
				return
			}
			if len(c.ts) <= int(c.rg.pos) {
				// outside of range - this is strange
				// try to analyze your code
				return
			}
			c.ch.SetText(c.ts[c.rg.pos])
			if f := c.OnChange; f != nil {
				f()
			}
		}
		c.rg.OnChange()
	}
	return c.ch.Render(width, dr)
}

func (c *Combobox) Focus(focus bool) { c.ch.Focus(focus) }
func (c *Combobox) Set(width, height uint) {
	c.ch.Set(width, height)
}
func (c *Combobox) Event(ev tcell.Event) {
	c.ch.Event(ev)
}

///////////////////////////////////////////////////////////////////////////////

// TODO
// Widget: Table

///////////////////////////////////////////////////////////////////////////////

// TODO
// Tabs examples:
//
//	<------------------------------------------*>|
//	+----------+ +----------+ +----------+       |
//	| [ TAB1 ] | | [ TAB2 ] | | [ TAB3 ] |       |
//	+          +-+----------+-+----------+-------|
//
//	<------------------------------------------*>|
//	+----------+ +----------+ +----------+       |
//	| [ TAB1 ] | | [ TAB2 ] | | [ TAB3 ] |       |
//	+----------+-+          +-+----------+-------|
//
//	<-----*--------------->
//	+----------+ +--------|
//	| [ TAB1 ] | | [ TAB2 |
//	+----------+-+        |
//
//	<-----*----------------->
//	+-------------++--------|
//	| [ TAB1 ][X] || [ TAB2 | // with close
//	+-------------++        |
//

type Tabs struct {
	Frame
	header ListH
}

func (t *Tabs) Add(name string, root Widget) {
	if len(t.header.nodes) == 0 {
		t.Header = &t.header
		t.Root = root
	}
	var btn Button
	btn.SetText(name)
	btn.OnClick = func() {
		t.Root = root
	}
	btn.Compress()
	t.header.Add(&btn)
}

///////////////////////////////////////////////////////////////////////////////

// TODO
// Widget: Dialog windows
// Widget: Open/Save/Save as dialog windows

///////////////////////////////////////////////////////////////////////////////

// TODO
// Widget: Menu

///////////////////////////////////////////////////////////////////////////////

// TODO
// Widget: ContextMenu

///////////////////////////////////////////////////////////////////////////////

// Tree examples:
//
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
type Tree struct {
	container

	Root        Widget
	offsetRoot  Offset
	Nodes       []Tree
	offsetNodes []Offset
}

func (tr *Tree) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		tr.Set(width, height)
	}()

	if width <= 1 {
		// hide unvisual tree elements
		return 1
	}

	if w := tr.Root; w != nil {
		height = w.Render(width, dr)
	}
	tr.offsetRoot.row = 0
	tr.offsetRoot.col = 0
	hs := []uint{height}

	if len(tr.offsetNodes) != len(tr.Nodes) {
		tr.offsetNodes = make([]Offset, len(tr.Nodes))
	}

	for i := range tr.Nodes {
		draw := func(row, col uint, st tcell.Style, r rune) {
			if width < col {
				panic("Text width")
			}
			dr(row+height, col+2, st, r)
		}
		tr.offsetNodes[i].col = 2
		tr.offsetNodes[i].row = height
		h := tr.Nodes[i].Render(width-2, draw)
		height += h
		hs = append(hs, height)
	}
	for i := range hs {
		if i == len(hs)-1 {
			continue
		}
		if 0 < i {
			for h := hs[i-1] + 1; h < hs[i]; h++ {
				dr(h, 0, TextStyle, LineVerticalUnfocus)
			}
		}
		if i == len(hs)-1-1 {
			dr(hs[i], 0, TextStyle, TreeUp)
		} else {
			dr(hs[i], 0, TextStyle, TreeUpDown)
		}
		dr(hs[i], 1, TextStyle, LineHorizontalUnfocus)
	}
	if 1 < len(hs) {
		height += 1
	}
	return
}

func (tr *Tree) Event(ev tcell.Event) {
	_, ok := tr.onFocus(ev)
	if ok {
		tr.Focus(true)
	}
	if !tr.focus {
		return
	}
	if tr.Root != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(tr.offsetRoot.col)
			row -= int(tr.offsetRoot.row)
			tr.Root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			tr.Root.Event(ev)
		}
	}
	for i := range tr.Nodes {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(tr.offsetNodes[i].col)
			row -= int(tr.offsetNodes[i].row)
			tr.Nodes[i].Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			tr.Nodes[i].Event(ev)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

func Demo() (root Widget, action chan func()) {
	var (
		scroll Scroll
		list   List
	)

	action = make(chan func(), 10)

	scroll.Root = &list
	{
		var listh ListH
		listh.MinWidth1element(20)
		list.Add(&listh)

		{
			var frame Frame
			listh.Add(&frame)
			frame.Header = TextStatic("Checkbox test")
			var list List
			frame.Root = &list
			size := 5
			option := make([]*bool, size)

			var optionInfo Text

			list.Add(TextStatic("Choose oprion/options:"))
			for i := 0; i < size; i++ {
				var ch CheckBox
				option[i] = &ch.Checked
				ch.SetText(fmt.Sprintf("Option %01d", i))
				ch.OnChange = func() {
					var str string = "Result:\n"
					for i := range option {
						opt := fmt.Sprintf("Option %01d is ", i)
						str += opt
						if *option[i] {
							str += "ON"
						} else {
							str += "OFF"
						}
						if i != len(option)-1 {
							str += "\n"
						}
					}
					optionInfo.SetText(str)
				}
				list.Add(&ch)
			}

			list.Add(&optionInfo)
		}

		{
			var log Text
			log.SetLinesLimit(2)
			var frame Frame
			listh.Add(&frame)
			frame.Header = TextStatic("Radio button test")
			var list List
			frame.Root = &list

			size := 5
			var names []string
			list.Add(TextStatic("Radio group:"))
			for i := 0; i < size; i++ {
				names = append(names, fmt.Sprintf("radiobutton%02d", i))
			}
			var optionInfo Text
			var rg RadioGroup
			rg.AddText(names...)
			{
				var ch CollapsingHeader
				ch.SetText("CollapsingHeader with CheckBoxes")
				var c0 CheckBox
				c0.SetText("CheckBox c0")
				var c1 CheckBox
				c1.SetText("CheckBox c1")
				var l List
				l.Add(TextStatic("Main check boxes inside radio group"))
				l.Add(&c0)
				l.Add(&c1)
				l.Add(TextStatic("Text example:"))
				var inp Inputbox
				inp.SetText("123456789")
				l.Add(&inp)
				ch.Root = &l
				rg.Add(&ch)
			}
			{
				var ch CollapsingHeader
				ch.SetText("CollapsingHeader example")
				ch.Root = TextStatic("Hello inside")
				rg.Add(&ch)
			}
			rg.OnChange = func() {
				var str string = "Result:\n"
				str += fmt.Sprintf("Choosed position: %02d", rg.GetPos())
				optionInfo.SetText(str)
				log.SetText(fmt.Sprintf("%d %s", rg.GetPos(), log.GetText()))
			}
			rg.SetPos(0)
			list.Add(&rg)
			list.Add(TextStatic("Logger:"))
			list.Add(&log)

			list.Add(&optionInfo)
		}
	}
	list.Add(new(Separator))
	{
		var frame Frame
		list.Add(&frame)
		frame.Header = TextStatic("Button test")
		var list List
		frame.Root = &list

		var counter uint
		view := func() string {
			return fmt.Sprintf("Counter : %02d", counter)
		}

		list.Add(TextStatic("Counter button"))
		var b Button
		b.SetText(view())

		var short Button
		short.Compress()
		short.SetText(view())
		short.OnClick = func() {
			counter++
			short.SetText(view())
			b.SetText(view())
			short.SetText(view())
		}
		b.OnClick = short.OnClick
		list.Add(&b)
		list.Add(&short)
	}
	list.Add(new(Separator))
	// 	{
	// 		var lh ListH
	// 		names := []string{"Hello", "World", "Gophers"}
	// 		for i := 0; i < 3; i++ {
	// 			for k :=0;k < 3;k++ {
	// 				names = append(names, fmt.Sprintf("%s%02d", names[i], k))
	// 			}
	// 		}
	// 		for _, name := range names {
	// 			var m MenuItem
	// 			m.Create(name, nil)
	// 			lh.Add(&m)
	// 		}
	// 		list.Add(&lh)
	// 	}
	list.Add(new(Separator))
	{
		var frame CollapsingHeader
		list.Add(&frame)
		frame.SetText("Inputbox test")
		var list List
		frame.Root = &list

		var ibs = []struct {
			name   string
			filter func(rune) bool
			text   func() string
		}{
			{
				name:   "String input box:",
				filter: nil,
			},
			{
				name:   "Unsigned integer input box:",
				filter: tf.UnsignedInteger,
			},
			{
				name:   "Integer input box:",
				filter: tf.Integer,
			},
			{
				name:   "Float input box:",
				filter: tf.Float,
			},
		}
		for i := range ibs {
			list.Add(TextStatic(ibs[i].name))
			var text Inputbox
			text.Filter(ibs[i].filter)
			list.Add(&text)
			ibs[i].text = text.GetText
		}
		var b Button
		b.SetText("Click for read result")
		list.Add(&b)

		var res Text
		list.Add(&res)

		b.OnClick = func() {
			var str string
			for i := range ibs {
				if t := ibs[i].text; t != nil {
					str += t() + "\n"
				}
			}
			res.SetText("Result:\n" + str)
		}
	}
	list.Add(new(Separator))
	{
		var ch CollapsingHeader
		ch.SetText("Header of collapsing header test")
		var il List
		il.SetHeight(4)

		ch.Root = &il

		var verylong string
		for i := 0; i < 100; i++ {
			verylong += "-0123456789="
		}
		il.Add(TextStatic(verylong))

		il.Add(TextStatic("Hello world!"))

		list.Add(&ch)
	}
	{
		list.Add(TextStatic("Example of tree"))

		var res Text
		res.SetText("Result:")
		var b Button
		b.SetText("Add char A and\nnew line separator")
		b.Compress()
		b.OnClick = func() {
			str := res.GetText()
			res.SetText(str + "\nA")
		}

		tr := Tree{
			Root: TextStatic("Root node"),
			Nodes: []Tree{
				Tree{Root: TextStatic("Childs 01\nLine 1\nLine 2")},
				Tree{Root: TextStatic("Childs 02: Long line. qwerty[posdifaslkdfjaskldjf;al ksdjf;alksdjf;laksdjf;laksdjfl;kasdjf;lkasdjf;lkasdjfl;kaj,vkmncx,mzxncfkasdjhkahdfiuewryhiuwehrkjdfhsadlkjfhalskdjhfaslkdjhfalskdjhflaksdjhfalksdjfhaklsdjhflkasdjhfaklsdjhflaksdjh")},
				Tree{Root: TextStatic("Childs 03\nLine 1\nLine 2"),
					Nodes: []Tree{
						Tree{Root: &res},
						Tree{Root: &b},
						Tree{Root: &res},
					},
				},
				Tree{Root: TextStatic("Childs\n04\nMultilines")},
			},
		}
		list.Add(&tr)
	}
	{
		var t Tabs
		t.Add("nil", nil)
		for i := 0; i < 10; i++ {
			var list List
			list.Add(TextStatic(fmt.Sprintf("Some text %02d", i)))
			var text Inputbox
			if i%2 == 0 {
				text.SetText(fmt.Sprintf("== %d ==", i))
			}
			list.Add(&text)
			t.Add(fmt.Sprintf("tab %02d", i), &list)
		}
		list.Add(&t)
	}
	{
		var c Combobox
		c.Add([]string{"A", "BB", "CCC"}...)
		list.Add(&c)
	}

	var menu Menu
	menu.Root = &scroll

	{
		var btn1 Button
		btn1.SetText("File")
		menu.Add(&btn1)

		var btn2 Button
		btn2.SetText("Edit")
		menu.Add(&btn2)
	}

	return &menu, action
	// return &scroll, action
}

///////////////////////////////////////////////////////////////////////////////

type container struct {
	focus  bool
	width  uint
	height uint
}

func (c *container) Focus(focus bool) {
	c.focus = focus
}

func (c *container) Set(width, height uint) {
	c.width = width
	c.height = height
}

func (c container) Get() (width, height uint) {
	return c.width, c.height
}

func (c *container) Event(ev tcell.Event) {
	_, ok := c.onFocus(ev)
	if ok {
		c.Focus(true)
	}
}

func (c *container) onFocus(ev tcell.Event) (button [3]bool, ok bool) {
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
		if int(c.height) <= row {
			break
		}
		switch ev.Buttons() {
		case tcell.Button1:
			button[0] = true // Left mouse button
		case tcell.Button3:
			button[1] = true // Middle mouse button
		case tcell.Button2:
			button[2] = true // Right mouse button
		}
		ok = true
	}
	return
}

///////////////////////////////////////////////////////////////////////////////

type containerVerticalFix struct {
	container
	hmax     uint
	addlimit bool
}

func (c *containerVerticalFix) SetHeight(hmax uint) {
	c.hmax = hmax
	c.addlimit = true
}

///////////////////////////////////////////////////////////////////////////////

type Compressable interface {
	Compress()
}

type compress struct {
	state bool
}

func (c *compress) Compress() {
	c.state = true
}

///////////////////////////////////////////////////////////////////////////////

var TimeFrameSleep time.Duration

func init() {
	// Sleep between frames updates
	if TimeFrameSleep <= 0 {
		TimeFrameSleep = time.Second * 5
	}
}

var debugs []string

var (
	screen     tcell.Screen
	simulation bool
)

func Run(root Widget, action chan func(), chQuit <-chan struct{}, quitKeys ...tcell.Key) (err error) {
	defer func() {
		for i := range debugs {
			fmt.Println(i, ":", debugs[i])
		}
	}()

	if root == nil {
		err = fmt.Errorf("root widget is nil")
		return
	}

	tcell.SetEncodingFallback(tcell.EncodingFallbackUTF8)
	if simulation {
		screen = tcell.NewSimulationScreen("")
	} else {
		if screen, err = tcell.NewScreen(); err != nil {
			return
		}
	}
	if err = screen.Init(); err != nil {
		return
	}

	screen.EnableMouse(tcell.MouseButtonEvents) // Click event only
	screen.EnablePaste()                        // ?
	screen.SetStyle(ScreenStyle)
	screen.Clear()

	defer func() {
		screen.Fini()
	}()

	var mu sync.Mutex
	var quit bool

	// event actions
	chEvent := make(chan tcell.Event, 1)
	go func() {
		for {
			if quit {
				break
			}
			chEvent <- screen.PollEvent()
		}
	}()

	var ignore bool
	for {
		if quit {
			break
		}

		ignore = false

		select {
		case ev := <-chEvent:
			switch ev := ev.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				for i := range quitKeys {
					if quitKeys[i] == ev.Key() {
						quit = true
						break
					}
				}
			case *tcell.EventMouse:
				if ev.Buttons() == tcell.ButtonNone {
					ignore = true
					continue
				}
			}
			if quit {
				break
			}
			if ev != nil && root != nil {
				mu.Lock()
				if p, ok := ev.(*tcell.EventMouse); ok {
					bm := p.Buttons()
					if bm == tcell.Button1 || bm == tcell.Button2 || bm == tcell.Button3 {
						time.Sleep(time.Millisecond * 500) // sleep for Windows
					}
				}
				root.Event(ev)
				mu.Unlock()
			}
		case <-time.After(TimeFrameSleep):
			// time sleep beween frames
			// do nothing

		case <-chQuit:
			quit = true
		case f := <-action:
			if f == nil {
				// do nothing
				continue
			}
			// default action
			f()
		}
		// render

		if quit {
			break
		}
		if ignore {
			continue
		}

		// clear screen
		mu.Lock()
		screen.Clear()
		// draw root widget
		if width, height := screen.Size(); 0 < width && 0 < height {
			const widthOffset uint = 1 // for avoid terminal collisions
			// root wigdets
			if vf, ok := root.(VerticalFix); ok {
				vf.SetHeight(uint(height))
			}
			// ignore height of root widget height
			_ = root.Render(uint(width)-widthOffset,
				func(row, col uint, st tcell.Style, r rune) {
					if row < 0 || uint(height) < row {
						return
					}
					if col < 0 || uint(width) < col {
						return
					}
					screen.SetCell(int(col), int(row), st, r)
				})
		}
		// show screen result
		screen.Show()
		mu.Unlock()
	}
	return
}
