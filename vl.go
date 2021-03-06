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
	ScreenStyle        tcell.Style = style(tcell.ColorBlack, tcell.ColorWhite)
	TextStyle          tcell.Style = ScreenStyle
	ButtonStyle        tcell.Style = style(tcell.ColorBlack, tcell.ColorYellow)
	ButtonFocusStyle   tcell.Style = style(tcell.ColorBlack, tcell.ColorViolet)
	InputboxStyle      tcell.Style = style(tcell.ColorBlack, tcell.ColorYellow)
	InputboxFocusStyle tcell.Style = style(tcell.ColorBlack, tcell.ColorViolet)

	// select
	InputboxSelectStyle tcell.Style = style(tcell.ColorBlack, tcell.ColorMaroon)
)

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
	return screen.hmax
}

func (screen *Screen) SetHeight(hmax uint) {
	screen.containerVerticalFix.SetHeight(hmax)
	if screen.Root == nil {
		return
	}
	if vf, ok := screen.Root.(VerticalFix); ok {
		vf.SetHeight(hmax)
	}
}

func (screen *Screen) Event(ev tcell.Event) {
	if screen.Root == nil {
		return
	}
	screen.Root.Event(ev)
}

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

///////////////////////////////////////////////////////////////////////////////

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
	if sc.hmax <= 0 {
		height = sc.Root.Render(width, draw)
	} else {
		const scrollBarWidth uint = 1
		height = sc.Root.Render(width-scrollBarWidth, draw)
		// calculate location
		if 2 < sc.hmax {
			var value float32 // 0 ... 1
			if sc.hmax <= height {
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
				dr(r, width, st, '|')
			}
			dr(0, width, st, '???')
			dr(sc.hmax-1, width, st, '???')
			pos := uint(value * float32(sc.hmax-2))
			if pos == 0 {
				pos = 1
			}
			if pos == sc.hmax-1 {
				pos = sc.hmax - 2
			}
			dr(pos, width, st, '*')
		}
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
		switch ev.Buttons() {
		case tcell.WheelUp:
			if sc.offset == 0 {
				break
			}
			sc.offset--
		case tcell.WheelDown:
			sc.offset++
		default:
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
		sc.Root.Event(ev)
	}
}

///////////////////////////////////////////////////////////////////////////////

type List struct {
	container

	heights []uint
	ws      []Widget
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

func (l *List) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		l.Set(width, height)
	}()
	if len(l.ws) == 0 {
		return
	}
	draw := func(row, col uint, st tcell.Style, r rune) {
		if width < col {
			return
		}
		row += height
		dr(row, col, st, r)
	}
	if len(l.heights)+1 != len(l.ws) || len(l.heights) == 0 {
		l.heights = make([]uint, len(l.ws)+1)
	}
	l.heights[0] = height
	for i := range l.ws {
		if l.ws[i] == nil {
			l.heights[i+1] = height
			continue
		}
		height += l.ws[i].Render(width, draw)
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

func (l *List) Add(w Widget) {
	l.ws = append(l.ws, w)
}

func (l *List) Clear() {
	l.ws = nil
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
	OnClick  func()
	Compress bool
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
	if b.Compress {
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

// Frame examples:
//	+- Header ---------+
//	|      Root        |
//	+------------------+
type Frame struct {
	container

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

func (f *Frame) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		f.Set(width, height)
	}()
	if width < 4 {
		return 1
	}
	// draw frame
	drawRow := func(row uint) {
		var i uint
		for i = 0; i < width; i++ {
			if f.focus {
				dr(row, i, TextStyle, '=')
			} else {
				dr(row, i, TextStyle, '-')
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

func (r *radio) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		r.Set(width, height)
	}()
	if width < 6 {
		return 1
	}
	const banner = 4
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
		r.Root.Event(ev)
	}
}

// Radio - button with single choose
// Example:
//	(0) choose one
//	( ) choose two
type RadioGroup struct {
	container

	list     List
	pos      uint
	onChange func()
}

func (rg *RadioGroup) OnChange(f func()) {
	rg.onChange = f
}

func (rg *RadioGroup) Add(w Widget) {
	var r radio
	r.Root = w
	rg.list.Add(&r)
}

func (rg *RadioGroup) SetText(ts []string) {
	rg.list.Clear()
	for i := range ts {
		rg.Add(TextStatic(ts[i]))
	}
	if len(ts) <= int(rg.pos) {
		rg.pos = uint(len(ts))
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
			if f := rg.onChange; f != nil {
				f()
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
	onChange func()
}

func (ch *CheckBox) OnChange(f func()) {
	ch.onChange = f
}

func (ch *CheckBox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		ch.width = width
		ch.height = height
	}()
	if width < 6 {
		return 1
	}
	const banner = 4
	st := InputboxStyle
	if ch.Checked {
		st = InputboxSelectStyle
	}
	if ch.focus {
		st = InputboxFocusStyle
	}
	if ch.Checked {
		PrintDrawer(0, 0, st, dr, []rune("[v]"))
	} else {
		PrintDrawer(0, 0, st, dr, []rune("[ ]"))
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
	mouse, ok := ch.onFocus(ev)
	if ok {
		ch.Focus(true)
	}
	if !ch.focus {
		return
	}
	if mouse[0] {
		ch.Checked = !ch.Checked
		if f := ch.onChange; f != nil {
			f()
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type Inputbox struct {
	Text
}

var Cursor rune = '???'

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
	frame   Frame
	open    bool
	b       Button
	content string
	Root    Widget
	init    bool
}

func (c *CollapsingHeader) Focus(focus bool) {
	c.frame.Focus(focus)
	c.init = false
}

func (c *CollapsingHeader) SetText(str string) {
	c.content = str
	c.init = false
}

func (c *CollapsingHeader) Open(state bool) {
	c.open = state
	c.init = false
}

func (c *CollapsingHeader) Render(width uint, dr Drawer) (height uint) {
	if !c.init {
		c.b.OnClick = func() {
			if c.open {
				c.b.SetText("| ??? | " + c.content)
			} else {
				c.b.SetText("| ??? | " + c.content)
			}
			c.open = !c.open
		}
		c.b.OnClick()
		c.b.OnClick() // two times for closed by default
		c.frame.Header = &c.b
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

// Widget: Horizontal list
type ListH struct {
	containerVerticalFix

	widths []uint
	ws     []Widget
}

func (l *ListH) Focus(focus bool) {
	if !focus {
		for i := range l.ws {
			if w := l.ws[i]; w != nil {
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
	if len(l.ws) == 0 {
		return
	}
	if len(l.widths) != len(l.ws) {
		// width of each element
		w := uint(float32(width) / float32(len(l.ws)))
		// calculate widths
		l.widths = make([]uint, len(l.ws)+1)
		l.widths[0] = 0
		for i := range l.ws {
			l.widths[i+1] = l.widths[i] + w
		}
		l.widths[len(l.widths)-1] = width
	}
	for i := range l.ws {
		draw := func(row, col uint, st tcell.Style, r rune) {
			if 0 < l.hmax && l.hmax < row {
				return
			}
			col += l.widths[i]
			dr(row, col, st, r)
		}
		if l.ws[i] == nil {
			continue
		}
		h := l.ws[i].Render(l.widths[i+1]-l.widths[i], draw)
		if height < h {
			height = h
		}
	}
	return
}

func (l *ListH) SetHeight(hmax uint) {
	l.containerVerticalFix.SetHeight(hmax)
	for i := range l.ws {
		if l.ws[i] == nil {
			continue
		}
		if vf, ok := l.ws[i].(VerticalFix); ok {
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
		for i := range l.widths {
			if i == 0 {
				continue
			}
			if l.widths[i-1] <= uint(col) &&
				uint(col) < l.widths[i] {
				// row correction
				col -= int(l.widths[i-1])
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

func (l *ListH) Add(w Widget) {
	l.ws = append(l.ws, w)
}

///////////////////////////////////////////////////////////////////////////////

// Widget : Combobox
// Design :
// +-------------------+
// |                   |
// |                   |
// +-------------------+

///////////////////////////////////////////////////////////////////////////////

// Widget: Table

///////////////////////////////////////////////////////////////////////////////

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
// type Tabs struct {
// }

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

func Demo() (root Widget, action chan func()) {
	var (
		scroll Scroll
		list   List
	)

	action = make(chan func())

	scroll.Root = &list
	{
		var listh ListH
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
				ch.OnChange(func() {
					var str string = "Result:\n"
					for i := range option {
						str += fmt.Sprintf("Option %01d is ", i)
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
				})
				list.Add(&ch)
			}

			list.Add(&optionInfo)
		}

		{
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
			rg.SetText(names)
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
				ch.Root = &l
				rg.Add(&ch)
			}
			{
				var ch CollapsingHeader
				ch.SetText("CollapsingHeader example")
				ch.Root = TextStatic("Hello inside")
				rg.Add(&ch)
			}
			rg.OnChange(func() {
				var str string = "Result:\n"
				str += fmt.Sprintf("Choosed position: %02d", rg.GetPos())
				optionInfo.SetText(str)
			})
			list.Add(&rg)

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
		short.Compress = true
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
		ch.Root = &il

		il.Add(TextStatic("Hello world!"))

		list.Add(&ch)
	}

	return &scroll, action
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
	hmax uint
}

func (c *containerVerticalFix) SetHeight(hmax uint) {
	c.hmax = hmax
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
	chEvent := make(chan tcell.Event)
	go func() {
		for {
			if quit {
				break
			}
			chEvent <- screen.PollEvent()
		}
	}()

	for {
		if quit {
			break
		}

		select {
		case ev := <-chEvent:
			switch ev.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				for i := range quitKeys {
					if quitKeys[i] == ev.(*tcell.EventKey).Key() {
						quit = true
						break
					}
				}
			}
			if ev != nil && root != nil {
				mu.Lock()
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
