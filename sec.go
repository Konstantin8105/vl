package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Konstantin8105/tf"
	"github.com/gdamore/tcell/v2"
)

var StyleDefault = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorWhite)

// var StyleFocus = tcell.StyleDefault.
// 	Foreground(tcell.ColorBlack).
// 	Background(tcell.ColorYellow)

var StyleButton = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorBlue)

var StyleButtonFocus = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorRed)

var StyleInput = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorYellow)

var StyleInputFocus = tcell.StyleDefault.
	Foreground(tcell.ColorBlack).
	Background(tcell.ColorRed)

const MinWidth = 50

type Widget interface {
	Draw(Width int, dr Drawer) (Height int)
	Event(Ev tcell.Event)
	Focus(focus bool)
}

// Template for widgets:
//
//	func (...) Focus(focus bool) {}
//	func (...) Draw(width int, dr Drawer) (height int) {}
//	func (...) Event(ev tcell.Event) {}

type HorizontalBox struct {
	// distance from left side to border
	// between left and right widgets
	Border uint

	// widgets
	Left, Right Widget
}

func (hb *HorizontalBox) Focus(focus bool) {}
func (hb *HorizontalBox) Draw(width int, dr Drawer) (height int) {
	if 0 < hb.Border && hb.Left != nil {
		height = hb.Left.Draw(int(hb.Border), dr)
	}
	if hb.Right != nil && int(hb.Border) < width {
		draw := func(row, col int, s tcell.Style, r rune) {
			dr(row, col+int(hb.Border), s, r)
		}
		h2 := hb.Right.Draw(width-int(hb.Border), draw)
		if height < h2 {
			height = h2
		}
	}
	return
}
func (hb *HorizontalBox) Event(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventKey:
	case *tcell.EventMouse:
		col, row := ev.Position()
		if int(hb.Border) < col && hb.Right != nil {
			// to right
			col -= int(hb.Border)
			nev := tcell.NewEventMouse(col, row, ev.Buttons(), ev.Modifiers())
			hb.Right.Event(nev)
			DebugInfo = fmt.Sprintf("right: %d  %d", col, hb.Border)
			return
		}
		if col < int(hb.Border) && hb.Right != nil {
			// to left
			nev := tcell.NewEventMouse(col, row, ev.Buttons(), ev.Modifiers())
			hb.Left.Event(nev)
			DebugInfo = "left"
			return
		}
	}
}

type CollapsingHeader struct {
	b    Button
	open bool
	ws   []Widget
}

func (ch *CollapsingHeader) SetText(str string) {
	ch.b.SetText(str)
}

func (ch *CollapsingHeader) Add(w Widget) {
	ch.ws = append(ch.ws, w)
}

// ignore any actions
func (ch *CollapsingHeader) Focus(focus bool) {}

func (ch *CollapsingHeader) Draw(width int, dr Drawer) (height int) {
	// TODO : const (
	// TODO : 	OpenIndicator  = "Open"
	// TODO : 	CloseIndicator = "Close"
	// TODO : )
	// header
	height += ch.b.Draw(width, dr)
	// draw other widgets
	if !ch.open { // is close?
		return
	}
	// open collapsing header
	for i := range ch.ws {
		height += ch.ws[i].Draw(width, dr)
	}
	return
}

// ignore any actions
func (ch *CollapsingHeader) Event(ev tcell.Event) {}

type Input struct {
	text  tf.TextField
	focus bool
}

func (i *Input) Focus(focus bool) {
	i.focus = focus
}

func (i *Input) Draw(width int, dr Drawer) (height int) {
	// default style
	st := StyleInput
	if i.focus {
		st = StyleInputFocus
	}
	// default line
	var br []int
	showRow := func(row int) {
		found := false
		for i := range br {
			if br[i] == row {
				found = true
			}
		}
		if found {
			return
		}
		// draw empty button
		for i := 0; i < width; i++ {
			dr(row, i, st, ' ')
		}
		br = append(br, row)
	}
	showRow(0)
	// draw
	draw := func(row, col uint, r rune) {
		for i := 0; i <= int(row); i++ {
			showRow(i)
		}
		dr(int(row), int(col), st, r)
	}
	cur := func(row, col uint) {
		for i := 0; i <= int(row); i++ {
			showRow(i)
		}
		dr(int(row), int(col), st, '*')
	}
	if !i.text.NoUpdate {
		i.text.SetWidth(uint(width))
	}
	height = int(i.text.Render(draw, cur))
	return
}

// ignore any actions
func (i *Input) Event(ev tcell.Event) {
	if !i.focus {
		return
	}
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		switch ev.Buttons() {
		case tcell.Button1: // Left mouse
			col, row := ev.Position()
			DebugInfo = fmt.Sprintf("INP %d %d", col, row)
			i.Focus(true)
		}
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyUp:
			i.text.CursorMoveUp()
		case tcell.KeyDown:
			i.text.CursorMoveDown()
		case tcell.KeyLeft:
			i.text.CursorMoveLeft()
		case tcell.KeyRight:
			i.text.CursorMoveRight()
		case tcell.KeyEnter:
			i.text.Insert('\n')
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			i.text.KeyBackspace()
		case tcell.KeyDelete:
			i.text.KeyDel()
		default:
			i.text.Insert(ev.Rune())
		}
	}
}

type Line struct{}

// ignore any actions
func (l *Line) Focus(focus bool) {}

func (l *Line) Draw(width int, dr Drawer) (height int) {
	for i := 0; i < width; i++ {
		dr(0, i, StyleDefault, '─')
	}
	return 1
}

// ignore any actions
func (l *Line) Event(ev tcell.Event) {}

type Button struct {
	height uint // templorary data

	text    Text
	focus   bool
	OnClick func()
}

func (b *Button) Focus(focus bool) {
	b.focus = focus
}

func (b *Button) SetText(str string) {
	b.text.SetText(str)
}

func (b *Button) Draw(width int, dr Drawer) (height int) {
	defer func() {
		b.height = uint(height)
	}()
	// default style
	st := StyleButton
	if b.focus {
		st = StyleButtonFocus
		defer func() {
			b.focus = false
		}()
	}
	// show button row
	var br []int
	showRow := func(row int) {
		found := false
		for i := range br {
			if br[i] == row {
				found = true
			}
		}
		if found {
			return
		}
		// draw empty button
		for i := 0; i < width; i++ {
			dr(row, i, st, ' ')
		}
		br = append(br, row)

	}
	// draw runes
	draw := func(row, col int, s tcell.Style, r rune) {
		// draw empty lines
		for i := 0; i <= row+1; i++ {
			showRow(i)
		}
		// draw symbol
		s = st
		dr(row+1, col+1, s, r)
	}
	height = b.text.Draw(width-2, draw)
	// borders
	for i := 0; i < width; i++ {
		dr(0, i, st, '─')
		dr(height+1, i, st, '─')
	}
	height += 2
	for i := 0; i < height; i++ {
		dr(i, 0, st, '│')
		dr(i, width, st, '│')
	}
	dr(0, 0, st, '┌')
	dr(height-1, 0, st, '└')
	dr(height-1, width, st, '┘')
	dr(0, width, st, '┐')
	return
}

func (b *Button) Event(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		switch ev.Buttons() {
		case tcell.Button1: // Left mouse
			//  click on button
			col, row := ev.Position()
			DebugInfo = fmt.Sprintf("Button1: row=%d col=%d height=%d", row, col, b.height)
			if col < 0 {
				return
			}
			if row < 0 {
				return
			}
			if int(b.height) < row {
				return
			}
			// on click
			DebugInfo = fmt.Sprintf("Before OnClick %v", b)
			if f := b.OnClick; f != nil {
				DebugInfo = fmt.Sprintf("OnClick %v", b)
				f()
			}
		}
	}
}

type Text struct {
	text tf.TextField
}

func TextStatic(str string) *Text {
	t := new(Text)
	t.text.Text = []rune(str)
	return t
}

func (t *Text) SetText(str string) {
	t.text.Text = []rune(str)
	t.text.NoUpdate = false
}

// ignore any actions
func (t *Text) Focus(focus bool) {}

func (t *Text) Draw(width int, dr Drawer) (height int) {
	var st tcell.Style = StyleDefault
	draw := func(row, col uint, r rune) {
		dr(int(row), int(col), st, r)
	}
	if !t.text.NoUpdate {
		t.text.SetWidth(uint(width))
	}
	height = int(t.text.Render(draw, nil))
	return
}

// ignore any actions
func (t *Text) Event(ev tcell.Event) {}

type Scroll struct {
	offset  int
	heights []uint
	ws      []Widget
	focus   bool
}

func (sc Scroll) heightSumm() uint {
	var s uint = 0
	for _, h := range sc.heights {
		s += h
	}
	return s
}

func (sc *Scroll) Focus(focus bool) {
	sc.focus = focus
}

func (sc *Scroll) Draw(width int, dr Drawer) (height int) {
	draw := func(Row, Col int, Style tcell.Style, Rune rune) {
		dr(Row+height-sc.offset, Col, Style, Rune)
	}
	if len(sc.heights) != len(sc.ws) {
		sc.heights = make([]uint, len(sc.ws))
	}
	for i := range sc.ws {
		h := sc.ws[i].Draw(width, draw)
		height += h
		sc.heights[i] = uint(h)
	}
	return
}

func (sc *Scroll) Event(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventMouse:
		switch ev.Buttons() {
		case tcell.WheelUp:
			sc.offset--
			if sc.offset < 0 {
				sc.offset = 0
			}
		case tcell.WheelDown:
			sc.offset++
			const minViewLines = 2 // constant
			if maxOffset := int(sc.heightSumm()) - minViewLines; maxOffset < sc.offset {
				sc.offset = maxOffset
			}
		case tcell.Button1: // Left mouse
			col, row := ev.Position() // TODO compare col
			var hlast, hnew int
			hlast += sc.offset
			for i := range sc.heights {
				hlast = hnew
				hnew += int(sc.heights[i])
				if hlast < row && row < hnew {
					sc.Focus(true)
					if sc.ws[i] != nil {
						sc.ws[i].Focus(true)
						sc.ws[i].Event(tcell.NewEventMouse(
							col, row-hlast, ev.Buttons(), ev.Modifiers()))
					}
					return
				}
			}
		}
	}
	for i := range sc.ws {
		sc.ws[i].Event(ev)
	}
}

func (sc *Scroll) Add(w Widget) {
	sc.ws = append(sc.ws, w)
}

type Coordinate struct{ Row, Col int }

type Drawer = func(Row, Col int, Style tcell.Style, Rune rune)

type App struct {
	mu             sync.Mutex
	quit           bool
	quitKeys       []tcell.Key
	screen         tcell.Screen
	root           Widget
	TimeFrameSleep time.Duration
}

func (app *App) Init() (err error) {
	tcell.SetEncodingFallback(tcell.EncodingFallbackUTF8)
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err = screen.Init(); err != nil {
		return err
	}
	defer func() {
		app.screen = screen
	}()

	screen.EnableMouse(tcell.MouseButtonEvents) // Click event only
	screen.EnablePaste()                        // ?

	screen.SetStyle(StyleDefault)
	screen.Clear()

	go func() {
		for {
			ev := screen.PollEvent()
			switch ev.(type) {
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				for i := range app.quitKeys {
					if app.quitKeys[i] == ev.(*tcell.EventKey).Key() {
						app.quit = true
					}
				}
			}
			if ev != nil && app.root != nil {
				app.mu.Lock()
				app.root.Event(ev)
				app.mu.Unlock()
			}
		}
	}()
	return
}

func (app *App) QuitKeys(ks ...tcell.Key) (err error) {
	app.quitKeys = ks
	return
}

func (app *App) SetRoot(root Widget) (err error) {
	app.root = root
	return
}

func (app *App) Run() (err error) {
	for {
		// application is quit
		if app.quit {
			break
		}
		// Sleep between frames updates
		// 50 ms :  20 fps
		//  5 ms : 150 fps
		//  1 ms : 500 fps
		if app.TimeFrameSleep <= 0 {
			<-time.After(time.Millisecond * 50)
		} else {
			<-time.After(app.TimeFrameSleep)
		}
		// clear screen
		app.screen.Clear()
		app.mu.Lock()
		// draw root widget
		if width, height := app.screen.Size(); 0 < width && 0 < height {
			const widthOffset = 1 // for avoid terminal collisions
			// root wigdets
			rootH := app.root.Draw(width-widthOffset, func(row, col int, st tcell.Style, r rune) {
				// zero initial offset
				offset := Coordinate{Row: 0, Col: 0}
				row += offset.Row
				col += offset.Col
				if row < 0 || height < row {
					return
				}
				if col < 0 || width < col {
					return
				}
				app.screen.SetCell(col, row, st, r)
			})
			// ignore height of root widget height
			_ = rootH
		}
		// show screen result
		app.mu.Unlock()
		app.screen.Show()
	}
	return
}

func (app *App) Stop() (err error) {
	app.screen.Fini()
	return
}

var DebugInfo string

func main() {
	var app App
	if err := app.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Init: %v", err)
		os.Exit(1)
	}

	if err := app.QuitKeys(tcell.KeyCtrlC); err != nil {
		fmt.Fprintf(os.Stderr, "Init: %v", err)
		os.Exit(1)
	}

	// create a widget
	var root Scroll
	// add widgets
	{
		// for debug only
		{
			var dt Text
			go func() {
				for {
					<-time.After(time.Millisecond * 10)
					dt.SetText(DebugInfo)
				}
			}()
			root.Add(&dt)
		}
		for i := 0; i < 50; i++ {
			root.Add(TextStatic("Hello world\nMy dear friend"))
			{
				hb := HorizontalBox{
					Border: 20,
					Left:   TextStatic("Some think"),
					Right:  TextStatic("New part"),
				}
				root.Add(&hb)
			}
			{
				var counter int
				var b Button
				b.text.SetText(fmt.Sprintf("Button:%d", i))
				b.OnClick = func() {
					counter++
					b.SetText(fmt.Sprintf("Counter:%d", counter))
				}
				hb := HorizontalBox{
					Border: 15,
					Left:   TextStatic("Very very very long text"),
					Right:  &b,
				}
				root.Add(&hb)
			}
			{
				var counter int
				var b Button
				b.text.SetText(fmt.Sprintf("Button:%d", i))
				b.OnClick = func() {
					counter++
					b.SetText(fmt.Sprintf("Counter:%d", counter))
				}
				root.Add(&b)
			}
			{
				var inp Input
				//inp.text.SetText(fmt.Sprintf("Input:%d", i))
				root.Add(&inp)
			}
			{
				var dt Text
				go func(i int) {
					for {
						<-time.After(time.Millisecond * 10)
						dt.SetText(fmt.Sprintf("%03d: %016d", 2*i,
							time.Now().Nanosecond()))
					}
				}(i)
				root.Add(&dt)
			}
			root.Add(new(Line))
		}
	}

	if err := app.SetRoot(&root); err != nil {
		fmt.Fprintf(os.Stderr, "Run: %v", err)
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}

		if err := app.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Stop: %v", err)
			os.Exit(1)
		}
	}()

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Run: %v", err)
		os.Exit(1)
	}
}
