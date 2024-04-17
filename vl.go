package vl

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/Konstantin8105/tf"
	"github.com/gdamore/tcell/v2"
)

func Style(fd, bd tcell.Color) tcell.Style {
	return tcell.StyleDefault.Foreground(fd).Background(bd)
}

var (
	white  = tcell.ColorWhite
	yellow = tcell.ColorYellow
	focus  = tcell.ColorDeepPink // ColorDeepPink
	red    = tcell.ColorRed
	green  = tcell.ColorGreen
	black  = tcell.ColorBlack

	ScreenStyle        tcell.Style = Style(black, white)
	TextStyle          tcell.Style = ScreenStyle
	ButtonStyle        tcell.Style = Style(black, yellow)
	ButtonFocusStyle   tcell.Style = Style(black, focus)
	ButtonSelectStyle  tcell.Style = Style(black, green)
	InputBoxStyle      tcell.Style = Style(black, yellow)
	InputBoxFocusStyle tcell.Style = Style(black, focus)
	// cursor
	CursorStyle tcell.Style = Style(white, red)
	// select
	InputBoxSelectStyle tcell.Style = Style(black, green)
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

const maxSize uint = 10000

func NilDrawer(row, col uint, s tcell.Style, r rune) {}

func DrawerLimit(
	dr Drawer,
	drow, dcol uint,
	rowFrom, rowTo uint,
	colFrom, colTo uint,
) Drawer {
	return func(row, col uint, s tcell.Style, r rune) {
		row += drow
		col += dcol
		if maxSize <= row {
			panic(fmt.Errorf("row is too big: %d", row))
		}
		if maxSize <= col {
			panic(fmt.Errorf("col is too big: %d", col))
		}
		if row < rowFrom || rowTo < row { // outside roe
			return
		}
		if col < colFrom || colTo < col { // outside col
			return
		}
		dr(row, col, s, r)
	}
}

///////////////////////////////////////////////////////////////////////////////

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
	Event(ev tcell.Event)

	// store widget size
	StoreSize(width, height uint)

	// return for widget sizes
	GetSize() (width, height uint)
}

type WidgetVerticalFix interface {
	Widget
	VerticalFix
}

///////////////////////////////////////////////////////////////////////////////

type Cell struct {
	S tcell.Style
	R rune
}

type Screen struct {
	ContainerVerticalFix
	rootable
	fill func(rune, tcell.Style)
	//	dialog struct {
	//		Root             Widget
	//		offsetX, offsetY uint
	//	}
}

func (screen *Screen) Fill(fill func(rune, tcell.Style)) {
	screen.fill = fill
}

func (screen *Screen) GetContents(width uint, cells *[][]Cell) {
	screen.width = width
	// zero width
	if screen.width == 0 {
		(*cells) = make([][]Cell, screen.hmax)
	}
	// preliminary allocation of rows/height
	switch {
	case len(*cells) < int(screen.hmax):
		(*cells) = append(*cells, make([][]Cell, int(screen.hmax)-len(*cells))...)
	case int(screen.hmax) < len(*cells):
		(*cells) = (*cells)[:screen.hmax]
	}
	// preliminary allocation of col/width
	for i := range *cells {
		switch {
		case len((*cells)[i]) < int(screen.width):
			(*cells)[i] = append((*cells)[i], make([]Cell, int(screen.width)-len((*cells)[i]))...)
		case int(screen.width) < len((*cells)[i]):
			(*cells)[i] = (*cells)[i][:screen.width]
		}
	}
	// var cleaned []bool
	drawer := func(row, col uint, s tcell.Style, r rune) {
		(*cells)[row][col] = Cell{S: s, R: r}
	}
	_ = screen.Render(screen.width, drawer) // ignore height
	return
}

func Convert(cells [][]Cell) string {
	var buf strings.Builder
	var w int
	for r := range cells {
		// draw runes
		fmt.Fprintf(&buf, "%04d|", r+1)
		for c := range cells[r] {
			buf.WriteRune(cells[r][c].R)
		}
		if width := len(cells[r]); w < width {
			w = width
		}
		buf.WriteString("|")
		// draw backgrounds
		for c := range cells[r] {
			_, bg, _ := cells[r][c].S.Decompose()
			switch bg {
			case yellow:
				buf.WriteString("Y")
			case focus:
				buf.WriteString("F")
			case white:
				buf.WriteString(".")
			default:
				buf.WriteString("X")
			}
		}
		buf.WriteString("|\n")
	}
	fmt.Fprintf(&buf, "rows  = %3d\n", len(cells))
	fmt.Fprintf(&buf, "width = %3d\n", w)
	return buf.String()
}

func (screen *Screen) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		screen.StoreSize(width, height)
	}()
	if width == 0 {
		return
	}
	if screen.hmax == 0 {
		return
	}
	// draw default spaces
	// take a lot of resouses by performance
	if screen.fill == nil {
		for col := uint(0); col < width; col++ {
			for row := uint(0); row < screen.hmax; row++ {
				dr(row, col, ScreenStyle, ' ')
			}
		}
	} else {
		screen.fill(' ', ScreenStyle)
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
	if screen.root != nil {
		_ = screen.root.Render(width, draw) // ignore height
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
	screen.ContainerVerticalFix.SetHeight(hmax)
	if screen.root != nil {
		if _, ok := screen.root.(VerticalFix); ok {
			screen.root.(VerticalFix).SetHeight(hmax)
		}
	}
	//	if screen.dialog.root != nil {
	//		if _, ok := screen.dialog.root.(VerticalFix); ok {
	//			screen.dialog.root.(VerticalFix).SetHeight(hmax)
	//		}
	//	}
}

func (screen *Screen) Event(ev tcell.Event) {
	if screen.root == nil {
		return
	}
	// if screen.dialog.root != nil {
	// 	screen.dialog.root.Event(ev)
	// 	return
	// }
	screen.root.Event(ev)
}

// func (screen *Screen) Close() {
// 	screen.dialog.root = nil
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
// 	frame.root = &list
// 	frame.SetHeight(screen.hmax)
// 	screen.dialog.root = &frame
// }

///////////////////////////////////////////////////////////////////////////////

type Separator struct{ container }

func (s *Separator) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		s.StoreSize(width, height)
	}()
	return 1
}

///////////////////////////////////////////////////////////////////////////////

type Text struct {
	container
	content   tf.TextFieldLimit
	compress  bool
	maxLines  uint
	style     *tcell.Style
	addCursor bool
}

var DefaultMaxTextLines uint = 5

// SetMaxLines set maximal visible lines of text
func (t *Text) SetMaxLines(limit uint) {
	t.maxLines = limit
}

// SetStyle set style of text
func (t *Text) SetStyle(style *tcell.Style) {
	t.style = style
}

// SetLinesLimit set minimal visible lines of text
func (t *Text) SetLinesLimit(limit uint) {
	t.content.SetLinesLimit(limit)
}

// SetText set to new widget text
func (t *Text) SetText(str string) {
	t.content.SetText([]rune(str))
}

// GetText return widget text
func (t *Text) GetText() string {
	return string(t.content.GetText())
}

func (t *Text) Compress() {
	if !t.compress {
		t.compress = true
	}
}

func (t *Text) Filter(f func(r rune) (insert bool)) {
	t.content.Filter = f
}

func (t *Text) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		t.StoreSize(width, height)
	}()
	if width < 1 {
		width, height = 0, 0
		return
	}
	if t.style == nil {
		t.style = &TextStyle
	}
	t.content.SetWidth(width + 1)
	var cur func(row, col uint) = nil // hide cursor for not-focus inputbox
	if t.focus && t.addCursor {
		cur = func(row, col uint) {
			if width < col {
				panic("Text width")
			}
			st := CursorStyle
			dr(row, col, st, Cursor)
		}
	}

	// TODO min width

	draw := func(row, col uint, r rune) {}
	height = t.content.Render(draw, cur)
	// added for colorize unvisible lines too
	h := t.content.GetRenderHeight()
	if height < h {
		height = h
	}
	if 0 < t.maxLines && t.maxLines < height {
		height = t.maxLines
	}
	if t.compress {
		width = t.content.GetRenderWidth() + 1
	}

	// drawing
	for w := 0; w <= int(width); w++ {
		for h := 0; h < int(height); h++ {
			dr(uint(h), uint(w), *t.style, ' ')
		}
	}
	draw = func(row, col uint, r rune) {
		if maxSize < row {
			panic(fmt.Errorf("row more max size: %d", row))
		}
		if maxSize < col {
			panic(fmt.Errorf("col more max size: %d", col))
		}
		if width < col {
			return
		}
		if 0 < t.maxLines && t.maxLines <= row {
			return
		}
		dr(row, col, *t.style, r)
	}
	t.content.Render(draw, cur)
	if 0 < t.maxLines && t.maxLines < height {
		height = t.maxLines
	}

	return
}

// /////////////////////////////////////////////////////////////////////////////
type Static struct {
	Image
	lastWidth uint
	rootable
}

func (s *Static) Compress() {
	if s.root == nil {
		return
	}
	if c, ok := s.root.(Compressable); ok {
		c.Compress()
	}
}
func (s *Static) Render(width uint, dr Drawer) (height uint) {
	if width != s.lastWidth {
		s.lastWidth = width
		// rendering image and show
		s.root.Render(width, NilDrawer)
		width, height = s.root.GetSize()
		img := &s.Image.data
		if width == 0 || height == 0 {
			*img = nil
		} else {
			if uint(len(*img)) != height || uint(len((*img)[0])) != width {
				*img = make([][]Cell, height)
				for i := uint(0); i < uint(len(*img)); i++ {
					(*img)[i] = make([]Cell, width)
				}
				for i := range *img {
					for j := range (*img)[i] {
						(*img)[i][j] = Cell{S: ScreenStyle, R: ' '}
					}
				}
			}
			s.root.Render(width, func(row, col uint, s tcell.Style, r rune) {
				if col == width {
					return
				}
				(*img)[row][col] = Cell{S: s, R: r}
			})
		}
	}
	// show image
	return s.Image.Render(width, dr)
}

func TextStatic(str string) Widget {
	txt := new(Text)
	txt.content.SetText([]rune(str))
	txt.Compress()
	t := new(Static)
	t.SetRoot(txt)
	return t
}

// /////////////////////////////////////////////////////////////////////////////
const scrollBarWidth uint = 1

type Scroll struct {
	ContainerVerticalFix
	rootable
	offset uint
}

func (sc *Scroll) Focus(focus bool) {
	if sc.root == nil {
		return
	}
	sc.container.Focus(focus)
	sc.root.Focus(focus)
}

func (sc *Scroll) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		sc.StoreSize(width, height)
	}()
	if sc.root == nil {
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
	if width < 2 {
		return
	}
	if sc.addlimit {
		if width < scrollBarWidth {
			panic(fmt.Errorf("too small width %d %d", width, scrollBarWidth))
		}
		if maxSize < sc.hmax {
			panic(fmt.Errorf("too big sc.hmax: %d", sc.hmax))
		}
		height = sc.root.Render(width-scrollBarWidth, draw)
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
		height = sc.root.Render(width, draw)
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
	if sc.root == nil {
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
			sc.root.Focus(false)
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
			sc.root.Event(tcell.NewEventMouse(
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
			sc.root.Event(ev)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type List struct {
	ContainerVerticalFix
	nodes    []listNode
	compress bool
}

func (l *List) Size() int {
	return len(l.nodes)
}

func (l *List) Clear() {
	l.nodes = nil
}

func (l *List) Focus(focus bool) {
	if !focus {
		for i := range l.nodes {
			if w := l.nodes[i].w; w != nil {
				w.Focus(focus)
			}
		}
	}
	l.focus = focus
}

func (l *List) Compress() {
	l.compress = true
}

func (l List) getItemHmax() uint {
	return l.hmax / uint(len(l.nodes))
}

func (l *List) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		l.StoreSize(width, height)
	}()
	if width < 2 {
		width, height = 0, 0
		return
	}
	if len(l.nodes) == 0 {
		return
	}
	dh := l.getItemHmax()
	l.nodes[0].from = 0
	for i := range l.nodes {
		if l.nodes[i].w == nil {
			l.nodes[i].from = -1
			l.nodes[i].to = -1
			continue
		}
		// initialize sizes of widgets
		if l.compress && (l.addlimit && 0 < l.hmax) {
			l.nodes[i].w.Render(width, NilDrawer)
			_, h := l.nodes[i].w.GetSize()
			if l.hmax < h {
				h = l.hmax
			}
			l.nodes[i].to = l.nodes[i].from + int(h)
		} else if !l.addlimit {
			l.nodes[i].w.Render(width, NilDrawer)
			_, h := l.nodes[i].w.GetSize()
			l.nodes[i].to = l.nodes[i].from + int(h)
		} else {
			l.nodes[i].to = l.nodes[i].from + int(dh)
		}
		// prepare position of next node
		for pos := i + 1; pos < len(l.nodes); pos++ {
			if l.nodes[pos].w == nil {
				continue
			}
			l.nodes[pos].from = l.nodes[i].to
			break
		}
		// drawing
		l.nodes[i].w.Render(width, DrawerLimit(
			dr,
			uint(l.nodes[i].from), 0,
			uint(l.nodes[i].from), uint(l.nodes[i].to)-1,
			0, width,
		))
	}
	height = uint(l.nodes[len(l.nodes)-1].to)
	if l.addlimit {
		height = l.hmax
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
			if l.nodes[i].from <= row && row < l.nodes[i].to {
				// row correction
				row := row - int(l.nodes[i].from)
				// focus
				l.Focus(true)
				// 				if l.nodes[i].w == nil {
				// 					continue
				// 				}
				//l.nodes[i].w.Focus(true)
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

func (l *List) Get(index int) Widget {
	if index < 0 || len(l.nodes) <= index {
		// not valid index
		return nil
	}
	return l.nodes[index].w
}

func (l *List) Update(index int, w Widget) {
	if index < 0 || len(l.nodes) <= index {
		// not valid index
		return
	}
	l.nodes[index].w = w
}

func (l *List) Add(w Widget) {
	l.nodes = append(l.nodes, listNode{w: w})
}

func (l *List) SetHeight(hmax uint) {
	l.ContainerVerticalFix.SetHeight(hmax)
	for i := range l.nodes {
		if l.nodes[i].w == nil {
			continue
		}
		if vf, ok := l.nodes[i].w.(VerticalFix); ok {
			vf.SetHeight(l.getItemHmax())
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

// Menu line example:
// [ File ] [ Edit ] [ Select ] [ Groups ] [ Help ]
//
// Elements for submenu:
//   - Button
//   - Checkbox
//   - RadioGroup
type Menu struct {
	ContainerVerticalFix
	rootable

	header ListH

	// 	scroll Scroll
	frame Frame
	list  List

	readyForOpen bool
	opened       bool
	offset       Offset
	parent       *Menu
	subs         []*Menu
}

func (menu *Menu) SetHeight(hmax uint) {
	menu.ContainerVerticalFix.SetHeight(hmax)
	menu.fixRootHeight()
}

func (menu *Menu) fixRootHeight() {
	if menu.root != nil {
		h := menu.header.height
		if _, ok := menu.root.(VerticalFix); ok {
			if h <= menu.hmax {
				menu.root.(VerticalFix).SetHeight(menu.hmax - h)
			}
		}
	}
}

func (menu *Menu) AddButton(name string, OnClick func()) {
	if OnClick == nil {
		menu.AddText(name)
		return
	}
	// prepare element
	var btn Button
	// btn.SetMaxLines(1)
	// btn.SetLinesLimit(1)
	btn.SetText(name)
	btn.Compress()
	btn.OnClick = func() {
		if f := OnClick; f != nil {
			f()
		}
		menu.resetSubmenu()
	}
	// adding
	menu.list.Add(&btn)
	menu.header.Add(&btn)

	menu.frame.SetRoot(&menu.list)
}

func (menu *Menu) AddText(name string) {
	// prepare element
	txt := TextStatic(name)
	// adding
	menu.list.Add(txt)
	menu.header.Add(txt)

	menu.frame.SetRoot(&menu.list)
}

func (menu *Menu) AddMenu(name string, sub *Menu) {
	// prepare menu
	menu.subs = append(menu.subs, sub)
	pos := len(menu.subs) - 1
	for i := range menu.subs {
		menu.subs[i].parent = menu
	}
	// prepare element
	var btn Button
	btn.SetMaxLines(1)
	btn.SetLinesLimit(1)
	btn.SetText(name)
	btn.Compress()
	btn.OnClick = func() {
		menu.subs[pos].readyForOpen = true
	}
	// adding
	menu.list.Add(&btn)
	menu.header.Add(&btn)

	// menu.frame.Header = TextStatic(name)
	menu.frame.SetRoot(&menu.list)
}

var SubMenuWidth uint = 20

func (menu *Menu) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		menu.StoreSize(width, height)
	}()
	if menu.opened && menu.parent != nil {
		// menu.scroll.SetRoot(&menu.list)
		// menu.frame.SetRoot(&menu.list) // scroll)
		// menu.frame.SetHeight(23) // TODO
		// menu.list.Compress()
		if width < menu.offset.col+SubMenuWidth {
			if SubMenuWidth < width {
				menu.offset.col = width - SubMenuWidth
			} else {
				menu.offset.col = 0
			}
		}
		var w uint
		if menu.offset.col < width {
			w = width - menu.offset.col
		} else {
			if SubMenuWidth < width {
				w = SubMenuWidth
			} else {
				w = width
			}
		}
		if SubMenuWidth < w {
			w = SubMenuWidth
		}
		menu.frame.Render(w, DrawerLimit(
			dr,
			menu.offset.row, menu.offset.col,
			0, maxSize,
			0, width,
		))
	}
	if menu.parent == nil {
		menu.header.Compress()
		h := menu.header.Render(width, dr)
		if menu.root != nil {
			menu.fixRootHeight() // fix root
			height = menu.root.Render(width, DrawerLimit(
				dr,
				h, 0,
				0, menu.hmax,
				0, width,
			))
		}
		height += h // for menu
		if 0 < menu.hmax {
			height = menu.hmax
		}
	}
	// render menu only after render Root
	for _, m := range menu.subs {
		if m == nil {
			continue
		}
		if !m.opened {
			continue
		}
		m.Render(width, dr)
	}
	if menu.addlimit && 0 < menu.height {
		height = menu.hmax
	}
	return
}

func (menu *Menu) Event(ev tcell.Event) {
	var found bool
	{
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			if row < int(menu.header.height) {
				menu.resetSubmenu()
				menu.header.Event(tcell.NewEventMouse(
					col, row,
					ev.Buttons(),
					ev.Modifiers()))
				found = true
			}

			// TODO case *tcell.EventKey:
			// TODO 	menu.root.Event(ev)
		}
	}

	////////////////////////

	var isInside func(menu *Menu) (found bool)
	isInside = func(menu *Menu) (found bool) {
		defer func() {
			menu.Focus(found)
		}()
		if menu == nil {
			return
		}
		if !menu.opened {
			return
		}
		for i := range menu.subs {
			if found = isInside(menu.subs[i]); found {
				menu.subs[i].Focus(true)
				return
			}
		}
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(menu.offset.col)
			row -= int(menu.offset.row)
			if col < 0 {
				break
			}
			if row < 0 {
				break
			}
			w, h := menu.frame.GetSize()
			if int(w) < col {
				break
			}
			if int(h) < row {
				break
			}
			found = true
			menu.frame.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))
		}
		if !found {
			menu.readyForOpen = false
			menu.opened = false
		}
		return
	}
	for i := range menu.subs {
		found = found || isInside(menu.subs[i])
	}
	if !found {
		menu.resetSubmenu()
	}

	////////////////////////

	var readyForOpen func(menu *Menu)
	readyForOpen = func(menu *Menu) {
		if menu == nil {
			return
		}
		for i := range menu.subs {
			readyForOpen(menu.subs[i])
		}
		if !menu.readyForOpen {
			return
		}
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			// store submenu coordinates
			menu.readyForOpen = false
			if 0 <= col && 0 <= row {
				menu.opened = true
				menu.offset = Offset{
					col: uint(col),
					row: uint(row),
				}
				// offser of submenu for good view
				menu.offset.row += 1
			}

			// TODO case *tcell.EventKey:
			// TODO 	menu.resetSubmenu()
			// TODO 	// menu.header.Event(ev)
		}
	}
	readyForOpen(menu)
	// 	for i := range menu.subs {
	// 		readyForOpen(menu.subs[i])
	// 	}

	////////////////////////

	if !found && menu.root != nil && menu.parent == nil { // main menu
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			if row < int(menu.header.height) {
				break // return
			}
			row -= int(menu.header.height)
			if row < 0 {
				break
				// return
			}
			menu.root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			menu.root.Event(ev)
		}
	}
}

func (menu *Menu) resetSubmenu() {
	if menu.parent != nil {
		// recursive event
		menu.parent.resetSubmenu()
	}

	menu.readyForOpen = false
	menu.opened = false
	for i := range menu.subs {
		if menu.subs[i] == nil {
			continue
		}
		menu.subs[i].readyForOpen = false
		menu.subs[i].opened = false
	}
}

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
//
// button borders, for example: "[ Button ]", "< Button >"
type Button struct {
	Text
	OnClick func()
}

func (b *Button) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		b.StoreSize(width, height)
	}()
	if width < 3 {
		width, height = 0, 0
		return
	}
	// default style
	st := &ButtonStyle
	if b.focus {
		st = &ButtonFocusStyle
	}
	b.Text.style = st
	// constant
	const buttonOffset = 2
	// 	if width < 2*buttonOffset {
	// 		width = 2 * buttonOffset
	// 	}
	b.Text.Render(width-2*buttonOffset, DrawerLimit(
		dr,
		0, buttonOffset,
		0, maxSize,
		0, width-buttonOffset,
	))
	width, height = b.GetSize()
	width += 2 * buttonOffset
	for row := 0; row < int(height); row++ {
		dr(uint(row), 0, *st, '[')
		dr(uint(row), 1, *st, ' ')
		dr(uint(row), width-2, *st, ' ')
		dr(uint(row), width-1, *st, ']')
	}
	// DEBUG : for w := 0; w <= int(width); w++ {
	// DEBUG : 	dr(0, uint(w), st, '$')
	// DEBUG : }
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

type rootable struct{ root Widget }

func (rt *rootable) SetRoot(root Widget) {
	rt.root = root
}

///////////////////////////////////////////////////////////////////////////////

var _ Widget = (*Viewer)(nil)

type word struct {
	S *tcell.Style
	R []rune
}

type Colorize func(words []string) []*tcell.Style

func TypicalColorize(indicates []string, t tcell.Style) Colorize {
	clean := func(word string) string {
		word = strings.ToLower(word)
		word = strings.ReplaceAll(word, "-", " ")
		word = strings.ReplaceAll(word, ",", " ")
		fs := strings.Split(word, " ")
		word = strings.Join(fs, " ")
		word = strings.TrimSpace(word)
		return word
	}
	for i := range indicates {
		indicates[i] = clean(indicates[i])
	}
	sort.Strings(indicates) // sort
	// multi-words
	var multi [][]string
	for i := range indicates {
		fs := strings.Fields(indicates[i])
		if len(fs) < 2 {
			continue
		}
		for k := range fs {
			fs[k] = clean(fs[k])
		}
		multi = append(multi, fs)
	}
	return func(words []string) (styles []*tcell.Style) {
		styles = make([]*tcell.Style, len(words))
		for i := range words {
			words[i] = clean(words[i])
		}
		// single word indication
		for i := range words {
			pos := sort.SearchStrings(indicates, words[i])
			if pos < len(indicates) && words[i] != indicates[pos] {
				pos = len(indicates)
			}
			if pos == len(indicates) {
				continue
			}
			styles[i] = &t
		}
		// multi-word indication
		for i := range multi {
			if len(multi) < 1 {
				continue
			}
			var firsts []int
			for k := range words {
				if words[k] != multi[i][0] {
					continue
				}
				firsts = append(firsts, k)
			}
			if len(firsts) == 0 {
				continue
			}
			for _, f := range firsts {
				if len(words)-f < len(multi[i]) {
					continue
				}
				same := true
				counter := 0
				for k := 0; k < len(multi[i]); k++ {
					pos := f + k + counter
					if len(words) <= pos {
						same = false
						break
					}
					if len(words[pos]) == 0 {
						counter++
						k--
						continue
					}
					if multi[i][k] != words[pos] {
						same = false
						break
					}
				}
				if !same {
					continue
				}
				for k := 0; k < len(multi[i])+counter; k++ {
					styles[f+k] = &t
				}
			}
		}
		return
	}
}

type Viewer struct {
	ContainerVerticalFix
	colorize  []Colorize
	str       string
	noUpdate  bool
	data      [][]Cell
	linePos   [][]uint // counter
	lastWidth uint
	position  uint
}

func (v *Viewer) SetColorize(colorize ...Colorize) {
	v.colorize = colorize
	v.noUpdate = false
}

func (v *Viewer) SetText(str string) {
	v.str = str
	v.noUpdate = false
}

func (v *Viewer) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		v.StoreSize(width, height)
	}()
	if !v.noUpdate || v.lastWidth != width {
		v.render(width)
		v.noUpdate = true
		v.lastWidth = width
	}
	// drawing
	row := v.presentRow()
	for ; row < len(v.data); row++ {
		if v.addlimit && height == v.hmax {
			break
		}
		for col := range v.data[row] {
			dr(uint(height), uint(col), v.data[row][col].S, v.data[row][col].R)
		}
		height++
	}
	return
}

func (v Viewer) presentRow() int {
	for row := range v.linePos {
		for col := range v.linePos[row] {
			if v.linePos[row][col] == v.position || v.position < v.linePos[row][col] {
				return row
			}
		}
	}
	return len(v.linePos) - 1
}

func (v *Viewer) PrevPage() {
	if !v.addlimit {
		return
	}
	if v.hmax < 2 {
		return
	}
	row := v.presentRow() - int(v.hmax)
	if row <= 0 {
		v.position = 0
		return
	}
	v.position = v.linePos[row][0]
	return
}

func (v *Viewer) NextPage() {
	if !v.addlimit {
		return
	}
	if v.hmax < 2 {
		return
	}
	row := v.presentRow() + int(v.hmax)
	if len(v.linePos) <= row {
		v.position = v.linePos[len(v.linePos)-1][0]
		return
	}
	v.position = v.linePos[row][0]
	return
}

func (v *Viewer) SetPosition(position uint) {
	v.position = position
}
func (v *Viewer) GetPosition() (position uint) { return v.position }

func (v *Viewer) render(width uint) {
	// convert to string lines
	v.str = strings.ReplaceAll(v.str, "\r", "")
	v.str = strings.ReplaceAll(v.str, string(rune(160)), " ")
	lines := strings.Split(v.str, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	// constants
	const space = rune(' ')
	// parse one line
	OneLine := func(line string) (
		// return data
		data [][]Cell,
		linePos [][]uint,
	) {
		if len(line) == 0 {
			return
		}
		runes := []rune(line)
		// split by words
		ws := make([]word, 0, len(runes))
		ws = append(ws, word{S: &TextStyle, R: []rune{runes[0]}})
		for ilet := 1; ilet < len(runes); ilet++ {
			if !unicode.IsLetter(runes[ilet]) {
				ws = append(ws, word{S: &TextStyle, R: []rune{runes[ilet]}})
				continue
			}
			if unicode.IsLetter(runes[ilet-1]) {
				ws[len(ws)-1].R = append(ws[len(ws)-1].R, runes[ilet])
				continue
			}
			ws = append(ws, word{S: &TextStyle, R: []rune{runes[ilet]}})
		}
		// create list of words
		var words []string
		for n := range ws {
			words = append(words, string(ws[n].R))
		}
		// add colors
		for i := range v.colorize {
			if v.colorize[i] == nil {
				continue
			}
			styles := v.colorize[i](words)
			if len(styles) != len(words) {
				return
			}
			for n := range ws {
				if styles[n] == nil {
					continue
				}
				ws[n].S = styles[n]
			}
		}

		// drawing to image
		data = nil
		linePos = nil
		var counter uint
		render := func(width uint, dr Drawer) (height uint) {
			counter = 0
			pos := uint(0)
			for k := range ws {
				for ir := range ws[k].R {
					counter++
					dr(height, pos, *ws[k].S, ws[k].R[ir])
					pos++
					if width < pos+1 {
						height++
						pos = 0
					}
				}
			}
			height += 2
			return
		}
		// calculate height
		height := render(width, NilDrawer)
		linePos = make([][]uint, height)
		for i := 0; i < int(height); i++ {
			linePos[i] = make([]uint, width)
		}
		if 1 < height {
			height--
			data = make([][]Cell, height)
			for i := 0; i < int(height); i++ {
				data[i] = make([]Cell, width+1)
				for k := range data[i] {
					data[i][k] = Cell{S: TextStyle, R: space}
				}
			}
			dr := func(row, col uint, s tcell.Style, r rune) {
				data[row][col] = Cell{S: s, R: r}
				linePos[row][col] = counter - 1
			}
			_ = render(width, dr)
		}
		for row := 0; row < len(linePos); row++ {
			for col := 0; col < len(linePos[row]); col++ {
				if row == 0 && col == 0 {
					continue
				}
				if linePos[row][col] != 0 {
					continue
				}
				if 0 < col {
					linePos[row][col] = linePos[row][col-1]
				} else {
					linePos[row][col] = linePos[row-1][width-1]
				}
			}
		}
		return
	}

	v.data = nil
	v.linePos = nil

	datas := make([][][]Cell, len(lines))
	linePos := make([][][]uint, len(lines))
	var wg sync.WaitGroup
	for i := range lines {
		wg.Add(1)
		go func(i int) {
			datas[i], linePos[i] = OneLine(lines[i])
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := range lines {
		// data, linePos := OneLine(lines[i])
		if len(datas[i]) == 0 || len(linePos[i]) == 0 {
			continue
		}
		if 0 < i {
			add := v.linePos[len(v.linePos)-1][len(v.linePos[len(v.linePos)-1])-1]
			for row := range linePos[i] {
				for col := range linePos[i][row] {
					linePos[i][row][col] += add + 1
				}
			}
			row := make([]Cell, len(v.data[0]))
			for k := range row {
				row[k] = Cell{S: TextStyle, R: space}
			}
			v.data = append(v.data, row)
		}
		v.data = append(v.data, datas[i]...)
		v.linePos = append(v.linePos, linePos[i]...)
	}

	for row := 0; row < len(v.linePos); row++ {
		for col := 0; col < len(v.linePos[row]); col++ {
			if row*int(width)+col < int(v.linePos[row][col]) {
				panic(fmt.Errorf("%d %d %d = %d : %d",
					row, width, col, row*int(width)+col,
					v.linePos[row][col]),
				)
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

var _ Widget = (*Image)(nil)

type Image struct {
	ContainerVerticalFix
	data [][]Cell
}

func (img *Image) SetImage(data [][]Cell) {
	img.data = data
}

func (img *Image) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		img.StoreSize(width, height)
	}()
	for row := range img.data {
		for col := range img.data[row] {
			dr(uint(row), uint(col), img.data[row][col].S, img.data[row][col].R)
		}
	}
	height = uint(len(img.data))
	return
}

///////////////////////////////////////////////////////////////////////////////

// Frame examples
//
//	+- Header ---------+
//	|      Root        |
//	+------------------+
type Frame struct {
	ContainerVerticalFix
	rootable

	Header       Widget
	offsetHeader Offset
	offsetRoot   Offset

	cleaned []bool
}

func (f *Frame) SetHeight(hmax uint) {
	f.ContainerVerticalFix.SetHeight(hmax)
	if f.root != nil {
		if _, ok := f.root.(VerticalFix); ok {
			f.root.(VerticalFix).SetHeight(hmax)
		}
	}
}

func (f *Frame) Focus(focus bool) {
	if !focus {
		if w := f.Header; w != nil {
			w.Focus(focus)
		}
		if w := f.root; w != nil {
			w.Focus(focus)
		}
	}
	f.container.Focus(focus)
}

func (f *Frame) Render(width uint, drg Drawer) (height uint) {
	defer func() {
		f.StoreSize(width, height)
	}()
	{ // default cleaner
		for i := range f.cleaned {
			f.cleaned[i] = false
		}
	}
	dr := func(row, col uint, s tcell.Style, r rune) {
		if maxSize <= row {
			panic(fmt.Errorf("row is too big: %d", row))
		}
		if maxSize <= col {
			panic(fmt.Errorf("col is too big: %d", col))
		}
		if len(f.cleaned) <= int(row) {
			f.cleaned = append(f.cleaned, make([]bool, int(row)-len(f.cleaned)+1)...)
		}
		for r := uint(0); r <= row; r++ {
			if f.cleaned[r] {
				continue
			}
			for w := uint(0); w < width; w++ {
				drg(r, w, TextStyle, ' ')
			}
			f.cleaned[r] = true
		}
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
	// draw border
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
		draw := DrawerLimit(
			dr,
			0, 2,
			0, maxSize,
			0, width,
		)
		height = f.Header.Render(width-4, draw)
		// draw line
		wh, _ := f.Header.GetSize()
		for i := wh; i < width-2; i++ {
			row := uint(0)
			if f.focus {
				draw(row, i, TextStyle, LineHorizontalFocus)
			} else {
				draw(row, i, TextStyle, LineHorizontalUnfocus)
			}
		}
	} else {
		height = 1
	}
	// add limit of height
	if f.addlimit && height+2 <= f.hmax {
		hmax := f.hmax - height - 2
		if f.root != nil {
			if _, ok := f.root.(VerticalFix); ok {
				f.root.(VerticalFix).SetHeight(hmax)
			}
		}
	}
	// next step
	f.offsetRoot.row = height + 1
	f.offsetRoot.col = 2
	f.offsetHeader.row = 0
	f.offsetHeader.col = 2
	// draw root widget
	if f.root != nil {
		// TODO create empty cell at background
		//
		// _ = f.root.Render(width-2*f.offsetRoot.col, DrawerLimit(
		// 	func(row, col uint, s tcell.Style, r rune) {
		// 		// create empty background for menu
		// 		dr(row, col, s, r)
		// 	},
		// 	f.offsetRoot.row, f.offsetRoot.col,
		// 	0, maxSize,
		// 	0, width-2*f.offsetRoot.col+1,
		// ))
		h := f.root.Render(width-2*f.offsetRoot.col, DrawerLimit(
			dr,
			f.offsetRoot.row, f.offsetRoot.col,
			0, maxSize,
			0, width-2*f.offsetRoot.col+1,
		))
		height += h + 2
	}
	if f.addlimit {
		if 0 < f.hmax {
			height = f.hmax - 1
		} else {
			height = 0
		}
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
	if f.Header != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(f.offsetHeader.col)
			row -= int(f.offsetHeader.row)
			width, height := f.Header.GetSize()
			if col < 0 || row < 0 {
				break
			}
			if int(width) < col || int(height) < row {
				break
			}
			f.Header.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))
			return

		case *tcell.EventKey:
			f.Header.Event(ev)
		}
	}
	if f.root != nil {
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			col, row := ev.Position()
			col -= int(f.offsetRoot.col)
			row -= int(f.offsetRoot.row)
			f.root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))
			return

		case *tcell.EventKey:
			f.root.Event(ev)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////

type radio struct {
	container
	rootable
	choosed bool
}

func (r *radio) Focus(focus bool) {
	r.container.Focus(focus)
	if r.root != nil {
		r.root.Focus(focus)
	}
}

const banner = 4 // banner for CheckBox and radio

func (r *radio) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		r.StoreSize(width, height)
	}()
	if width < 6 {
		return 1
	}
	st := ButtonStyle
	if r.choosed {
		st = ButtonSelectStyle
	}
	if r.focus {
		st = ButtonFocusStyle
	}
	if r.choosed {
		PrintDrawer(0, 0, st, dr, []rune("(*)"))
	} else {
		PrintDrawer(0, 0, st, dr, []rune("( )"))
	}
	if r.root != nil {
		if ch, ok := r.root.(*CollapsingHeader); ok {
			ch.Open(r.choosed)
		}
		height = r.root.Render(width-banner, DrawerLimit(
			dr,
			0, banner,
			0, maxSize,
			0, width,
		))
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
	if r.root != nil {
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
			r.root.Event(tcell.NewEventMouse(
				col, row,
				ev.Buttons(),
				ev.Modifiers()))

		case *tcell.EventKey:
			r.root.Event(ev)
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
	r.root = w
	rg.list.Add(&r)
	rg.pos = uint(len(rg.list.nodes) - 1)
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
	if len(rg.list.nodes) <= int(rg.pos) {
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
		rg.StoreSize(width, height)
	}()
	if len(rg.list.nodes) <= int(rg.pos) {
		rg.pos = 0
	}
	for i := range rg.list.nodes {
		if uint(i) == rg.pos {
			rg.list.nodes[i].w.(*radio).choosed = true
			continue
		}
		rg.list.nodes[i].w.(*radio).choosed = false
	}
	height = rg.list.Render(width, dr)
	return
}

func (rg *RadioGroup) Event(ev tcell.Event) {
	rg.list.Event(ev)
	if rg.list.focus {
		// change radio position
		last := rg.pos
		for i := range rg.list.nodes {
			if rg.list.nodes[i].w.(*radio).focus {
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
	st := &ButtonStyle
	if ch.Checked {
		st = &ButtonSelectStyle
	}
	if ch.focus {
		st = &ButtonFocusStyle
	}
	if len(ch.pair[0]) == 0 || len(ch.pair[1]) == 0 {
		// default values
		ch.pair = [2]string{"[v]", "[ ]"}
	}
	if width < uint(len(ch.pair[0])+1+1) {
		// not enought for 1 symbol
		return 1
	}
	var lenght uint = 0
	if ch.Checked {
		PrintDrawer(0, 0, *st, dr, []rune(ch.pair[0]))
		lenght = uint(len(ch.pair[0]))
	} else {
		PrintDrawer(0, 0, *st, dr, []rune(ch.pair[1]))
		lenght = uint(len(ch.pair[1]))
	}
	dr(0, lenght, TextStyle, ' ')
	height = ch.Text.Render(width-lenght-1, DrawerLimit(
		dr,
		0, lenght+1,
		0, maxSize,
		lenght+1, width,
	))
	if height < 2 {
		height = 1
	}
	if ch.Text.compress {
		width = ch.Text.width + lenght + 1
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

type InputBox struct {
	Text
}

var Cursor rune = '_'

func (in *InputBox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		in.StoreSize(width, height)
	}()
	// set test property
	st := &InputBoxStyle
	if in.focus {
		st = &InputBoxFocusStyle
	}
	in.Text.style = st
	in.Text.addCursor = true
	return in.Text.Render(width, dr)
}

func (in *InputBox) Event(ev tcell.Event) {
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
	rootable
	frame Frame
	open  bool
	cb    CheckBox
	init  bool
}

func (c *CollapsingHeader) Focus(focus bool) {
	c.frame.Focus(focus)
}

func (c *CollapsingHeader) SetText(str string) {
	c.cb.SetText(str)
	c.cb.SetMaxLines(DefaultMaxTextLines)
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
		c.cb.Compress()
		c.frame.Header = &c.cb
		c.init = true
	}
	if c.open {
		c.frame.root = c.root
	} else {
		c.frame.root = nil
	}
	return c.frame.Render(width, dr)
}

func (c *CollapsingHeader) StoreSize(width, height uint) {
	c.frame.StoreSize(width, height)
}

func (c CollapsingHeader) GetSize() (width, height uint) {
	return c.frame.GetSize()
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
	ContainerVerticalFix
	Splitter func(width uint, size int) (elementWidth []int)

	nodes    []listNode
	compress bool
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

func (l *ListH) Compress() {
	l.compress = true
}

func (l *ListH) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		l.StoreSize(width, height)
	}()
	if len(l.nodes) == 0 {
		return
	}
	if l.nodes[len(l.nodes)-1].to != int(width) {
		if l.compress {
			for i := range l.nodes {
				// initialize sizes of widgets
				l.nodes[i].w.Render(width, NilDrawer)
			}
			l.nodes[0].from = 0
			for i := range l.nodes {
				if c, ok := l.nodes[i].w.(Compressable); ok {
					c.Compress()
				} else {
					panic(fmt.Errorf("add Compressable widget: %T", l.nodes[i].w))
				}
				w, _ := l.nodes[i].w.GetSize()
				l.nodes[i].to = l.nodes[i].from + int(w)
				if i+1 != len(l.nodes) {
					l.nodes[i+1].from = l.nodes[i].to + 1
				}
			}
			l.nodes[len(l.nodes)-1].to = int(width)
		} else {
			// use splitter
			added := false
			if l.Splitter != nil {
				ws := l.Splitter(width, len(l.nodes))
				if len(ws) == len(l.nodes) {
					summ := 0
					for i := range ws {
						summ += ws[i]
					}
					summ += len(l.nodes) - 1 // borders
					if summ == int(width) {
						summ = 0
						for i := range ws {
							l.nodes[i].from = summ
							summ += ws[i]
							l.nodes[i].to = summ
							summ++
						}
						added = true
					}
				}
			}
			if !added {
				// width of each element
				// gap 1 symbol between widgets
				dw := int(float32(width-uint(len(l.nodes)-1)) / float32(len(l.nodes)))
				// calculate widths
				for i := range l.nodes {
					l.nodes[i].from = i * (dw + 1)
					l.nodes[i].to = l.nodes[i].from + dw
				}
			}
			if 0 < len(l.nodes) {
				l.nodes[len(l.nodes)-1].to = int(width)
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
			if width < col {
				return
			}
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
	l.ContainerVerticalFix.SetHeight(hmax)
	if len(l.nodes) == 0 {
		return
	}
	l.nodes[len(l.nodes)-1].to = -1
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

func (l *ListH) Size() int {
	return len(l.nodes)
}

func (l *ListH) Clear() {
	l.nodes = nil
}

///////////////////////////////////////////////////////////////////////////////

// ComboBox example
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
type ComboBox struct {
	ch       CollapsingHeader
	rg       RadioGroup
	ts       []string
	OnChange func()
}

func (c *ComboBox) Add(ts ...string) {
	for _, s := range ts {
		c.ts = append(c.ts, s)
		c.rg.Add(TextStatic(s))
	}
	if f := c.OnChange; f != nil {
		f()
	}
	if f := c.rg.OnChange; f != nil {
		f()
	}
}

func (c *ComboBox) Clear() {
	c.rg.Clear()
	c.ts = []string{}
	c.OnChange = nil
}

func (c *ComboBox) SetPos(pos uint) {
	c.rg.SetPos(pos)
	if f := c.OnChange; f != nil {
		f()
	}
	if f := c.rg.OnChange; f != nil {
		f()
	}
}

func (c *ComboBox) GetPos() uint {
	return c.rg.GetPos()
}

func (c *ComboBox) checkUpdater() {
	if c.rg.OnChange != nil {
		return
	}
	c.rg.OnChange = func() {
		c.ch.SetText("")
		if len(c.ts) == 0 {
			// empty list
			return
		}
		// c.rg.pos = -1; int(c.rg.pos) = 18446744073709551615
		// So, use `uint(len(c.ts)) <= c.rg.pos`
		if uint(len(c.ts)) <= c.rg.pos {
			// outside of range - this is strange
			// try to analyze your code
			if 0 < len(c.ts) {
				c.rg.pos = uint(len(c.ts) - 1)
			} else {
				return
			}
		}
		c.ch.SetText(c.ts[c.rg.pos])
		if f := c.OnChange; f != nil {
			f()
		}
	}
	c.rg.OnChange()
}

func (c *ComboBox) Render(width uint, dr Drawer) (height uint) {
	defer func() {
		c.StoreSize(width, height)
	}()
	if width < 4 {
		return 1
	}
	if c.ch.root == nil {
		c.ch.root = &c.rg
	}
	c.checkUpdater()
	return c.ch.Render(width, dr)
}

func (c *ComboBox) StoreSize(width, height uint) {
	c.ch.StoreSize(width, height)
}

func (c ComboBox) GetSize() (width, height uint) {
	return c.ch.GetSize()
}

func (c *ComboBox) Focus(focus bool) { c.ch.Focus(focus) }
func (c *ComboBox) Event(ev tcell.Event) {
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
	OnChange    func()
	init        bool
	headerListH *ListH
	headerCombo *ComboBox
	combo       bool
	list        struct {
		names []string
		roots []Widget
	}
}

func (t *Tabs) Add(name string, root Widget) {
	if name == "" || root == nil {
		return
	}
	if !t.init {
		t.headerListH = new(ListH)
		t.headerCombo = new(ComboBox)
		t.UseCombo(false)
		t.init = true
	}
	t.list.names = append(t.list.names, name)
	t.list.roots = append(t.list.roots, root)
	{
		// buttons
		var btn Button
		btn.SetText(name)
		btn.OnClick = func() {
			t.Frame.root = root
			if f := t.OnChange; f != nil {
				f()
			}
		}
		btn.Compress()
		t.headerListH.Add(&btn)
		btn.OnClick()
	}
	{
		// combo
		t.headerCombo.Add(t.list.names[len(t.list.names)-1])
		t.headerCombo.OnChange = func() {
			pos := t.headerCombo.GetPos()
			if int(pos) < len(t.list.roots) {
				t.Frame.root = t.list.roots[pos]
			}
			if f := t.OnChange; f != nil {
				f()
			}
		}
		if 0 < len(t.list.names) {
			t.headerCombo.SetPos(0)
		}
	}
}

func (t *Tabs) UseCombo(combo bool) {
	// convert
	defer func() {
		t.combo = combo
	}()
	if combo {
		// from ListH to ComboBox
		t.Frame.Header = t.headerCombo
		return
	}
	// from ComboBox to ListH
	t.Frame.Header = t.headerListH
	t.headerListH.Compress()
}

///////////////////////////////////////////////////////////////////////////////

type Stack struct {
	widgets []WidgetVerticalFix
}

func (s *Stack) Push(w WidgetVerticalFix) {
	s.widgets = append(s.widgets, w)
}
func (s *Stack) Pop() {
	if len(s.widgets) == 0 {
		return
	}
	s.widgets = s.widgets[:len(s.widgets)-1]
}

func (s *Stack) present() WidgetVerticalFix {
	if len(s.widgets) == 0 {
		sc := new(Scroll)
		sc.SetRoot(TextStatic("stack is empty"))
		return sc
	}
	return s.widgets[len(s.widgets)-1]
}
func (s *Stack) Focus(focus bool) {
	s.present().Focus(focus)
}
func (s *Stack) Render(width uint, dr Drawer) (height uint) {
	return s.present().Render(width, dr)
}
func (s *Stack) Event(ev tcell.Event) {
	s.present().Event(ev)
}
func (s *Stack) StoreSize(width, height uint) {
	s.present().StoreSize(width, height)
}
func (s *Stack) GetSize() (width, height uint) {
	return s.present().GetSize()
}
func (s *Stack) SetHeight(hmax uint) {
	s.present().SetHeight(hmax)
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
		tr.StoreSize(width, height)
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

func Demo() (demos []Widget) {
	var (
		scroll Scroll
		list   List
	)
	defer func() {
		for i := range list.nodes {
			demos = append(demos, list.nodes[i].w)
		}
	}()

	scroll.SetRoot(&list)
	{
		var listh ListH
		list.Add(&listh)

		{
			var frame Frame
			listh.Add(&frame)
			frame.Header = TextStatic("Checkbox test")
			var list List
			frame.SetRoot(&list)
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
			list.Add(new(Separator))
			{
				var viewer Viewer
				viewer.SetText(`In according to https://en.wikipedia.org/wiki/Representational_systems_(NLP)
According to Bandler and Grinder our chosen words, phrases and sentences are indicative of our referencing of each of the representational systems.[4] So for example the words "black", "clear", "spiral" and "image" reference the visual representation system; similarly the words "tinkling", "silent", "squeal" and "blast" reference the auditory representation system.[4] Bandler and Grinder also propose that ostensibly metaphorical or figurative language indicates a reference to a representational system such that it is actually literal. For example, the comment "I see what you're saying" is taken to indicate a visual representation.[5]`)

				viewer.SetColorize([]Colorize{
					TypicalColorize(
						[]string{"see", "visual", "black", "white", "image", "indicate"},
						Style(tcell.ColorWhite, tcell.ColorGreen),
					),
					TypicalColorize(
						[]string{"bandler", "i", "you", "grinder"},
						Style(tcell.ColorDeepPink, tcell.ColorYellow),
					),
					TypicalColorize(
						[]string{"silent", "saying"},
						Style(tcell.ColorBlack, tcell.ColorBlue),
					),
					TypicalColorize(
						[]string{"or", "for example", "also", "is taken to",
							"in according to", "According to", "and", "to"},
						Style(tcell.ColorBlack, tcell.ColorDeepPink),
					),
				}...)
				list.Add(&viewer)
			}
		}

		{
			var log Text
			log.SetLinesLimit(2)
			var frame Frame
			listh.Add(&frame)
			frame.Header = TextStatic("Radio button test")
			var list List
			frame.SetRoot(&list)

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
				var inp InputBox
				inp.SetText("123456789")
				l.Add(&inp)
				ch.SetRoot(&l)
				rg.Add(&ch)
			}
			{
				var ch CollapsingHeader
				ch.SetText("CollapsingHeader example")
				ch.SetRoot(TextStatic("Hello inside"))
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
		var listh ListH

		{
			var frame Frame
			listh.Add(&frame)
			frame.Header = TextStatic("Button test")

			var list List
			frame.SetRoot(&list)

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
		{
			var frame Frame
			listh.Add(&frame)
			frame.Header = TextStatic("Image")
			data := make([][]Cell, 4)
			for i := range data {
				data[i] = make([]Cell, 15)
			}
			for i := range data {
				for j := range data[i] {
					var c Cell
					if (i+j)%2 == 0 {
						c.R = rune('F')
					} else {
						c.R = rune('Q')
					}
					if (i+j)%3 == 0 {
						c.S = Style(tcell.ColorBlack, tcell.ColorRed)
					} else {
						c.S = Style(tcell.ColorYellow, tcell.ColorGreen)
					}
					data[i][j] = c
				}
			}
			var img Image
			img.SetImage(data)
			frame.SetRoot(&img)
		}

		list.Add(&listh)

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
		frame.SetText("InputBox test")
		var list List
		frame.SetRoot(&list)

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
			var text InputBox
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

		ch.SetRoot(&il)

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
			var text InputBox
			if i%2 == 0 {
				text.SetText(fmt.Sprintf("== %d ==", i))
			}
			list.Add(&text)
			t.Add(fmt.Sprintf("tab %02d", i), &list)
		}
		list.Add(&t)
	}
	{
		var c ComboBox
		c.Add([]string{"A", "BB", "CCC"}...)
		list.Add(&c)
	}

	var menu Menu
	menu.SetRoot(&scroll)
	menu.AddButton("File", nil)
	{
		var sub Menu
		for i := 0; i < 30; i++ {
			name := fmt.Sprintf("Text%02d", i)
			if i%3 == 0 {
				sub.AddButton(name, func() {
					debugs = append(debugs, fmt.Sprintln("Click:"+name))
				})
			} else {
				sub.AddText(name)
			}
			if 0 < i && i%5 == 0 {
				sub.AddText("long long long long long long long long long long long long")
			}
			if i%4 == 0 {
				name += "Sub"
				var ss Menu
				for k := 0; k < 5; k++ {
					ss.AddButton(name, func() {
						debugs = append(debugs, fmt.Sprintln("Click Sub:"+name))
					})
				}
				sub.AddMenu(fmt.Sprintf("Submenu%02d", i), &ss)
			}
		}
		menu.AddMenu("Edit", &sub)
	}
	{
		var sub Menu
		for i := 0; i < 5; i++ {
			name := fmt.Sprintf("SecondText%02d", i)
			sub.AddText(name)
		}
		sub.AddButton("long long long long long long long long long long long long", func() {})
		menu.AddMenu("View", &sub)
	}
	// 	{
	// 		var cb CheckBox
	// 		cb.SetText("Line")
	// 		menu.Add(&cb)
	// 	}
	menu.AddButton("Help", nil)

	demos = append(demos, &menu)
	return
}

///////////////////////////////////////////////////////////////////////////////

type container struct {
	focus  bool
	width  uint
	height uint
}

// Focus modify focus state of widget
func (c *container) Focus(focus bool) {
	c.focus = focus
}

// StoreSize store widget size
func (c *container) StoreSize(width, height uint) {
	c.width = width
	c.height = height
}

// GetSize return for widget sizes
func (c container) GetSize() (width, height uint) {
	return c.width, c.height
}

// Event for widget actions
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

type ContainerVerticalFix struct {
	container
	hmax     uint
	addlimit bool
}

func (c *ContainerVerticalFix) SetHeight(hmax uint) {
	if maxSize < hmax {
		panic(fmt.Errorf("SetHeight: too big size: %d", hmax))
	}
	c.hmax = hmax
	c.addlimit = true
}

func (c *ContainerVerticalFix) GetLimit() (addlimit bool, hmax uint) {
	return c.addlimit, c.hmax
}

///////////////////////////////////////////////////////////////////////////////

type Compressable interface {
	Compress()
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

	if sc, ok := root.(*Screen); ok {
		sc.Fill(screen.Fill)
	}

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
				if runtime.GOOS == "windows" {
					if p, ok := ev.(*tcell.EventMouse); ok {
						bm := p.Buttons()
						if bm == tcell.Button1 || bm == tcell.Button2 || bm == tcell.Button3 {
							time.Sleep(time.Millisecond * 500) // sleep for Windows
						}
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
