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

const MinWidth = 50

type Widget interface {
	Draw(Width int, dr Drawer) (Height int)
	Event(Ev tcell.Event)
	Focus(focus bool)
}

type Button struct {
	text    Text
	focus bool
	OnClick func()
}

// ignore any actions
func (b *Button) Focus(focus bool) {
	if f := b.OnClick; f != nil && focus {
		f()
	}
	b.focus = focus
}

func (b *Button) Draw(width int, dr Drawer) (height int) {
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
		showRow(row + 1)
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

// ignore any actions
func (b *Button) Event(ev tcell.Event) {}

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
	he := func(row int) {
		if height < row {
			height = row
		}
	}
	draw := func(row, col uint, r rune) {
		dr(int(row), int(col), st, r)
		he(int(row))
	}
	if !t.text.NoUpdate {
		t.text.SetWidth(uint(width))
	}
	t.text.Render(draw, nil)
	height++
	return
}

// ignore any actions
func (t *Text) Event(ev tcell.Event) {}

type Scroll struct {
	offset int
	height int
	ws     []Widget
	focus  bool
	mouse  struct {
		coord Coordinate
		check bool
	}
}

func (sc *Scroll) Focus(focus bool) {
	sc.focus = focus
}

func (sc *Scroll) Draw(width int, dr Drawer) (height int) {
	draw := func(Row, Col int, Style tcell.Style, Rune rune) {
		dr(Row+height-sc.offset, Col, Style, Rune)
	}
	for i := range sc.ws {
		height += sc.ws[i].Draw(width, draw)
		if sc.mouse.check {
			if 0 <= sc.mouse.coord.Row+sc.offset &&
				sc.mouse.coord.Row+sc.offset < height &&
				0 <= sc.mouse.coord.Col &&
				sc.mouse.coord.Col <= width {
				sc.Focus(true)
				sc.ws[i].Focus(true)
				sc.mouse.check = false
			}
		}
	}
	if sc.mouse.check {
		sc.mouse.check = false
	}
	sc.height = height
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
			if maxOffset := sc.height - minViewLines; maxOffset < sc.offset {
				sc.offset = maxOffset
			}
		case tcell.Button1: // Left mouse
			sc.mouse.coord.Col, sc.mouse.coord.Row = ev.Position()
			sc.mouse.check = true
			sc.Focus(false)
			for i := range sc.ws {
				sc.ws[i].Focus(false)
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

	screen.EnableMouse()
	screen.EnablePaste() // ?

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
		for i := 0; i < 50; i++ {
			root.Add(TextStatic("Hello world\nMy dear friend"))
			{
				var counter int
				var b Button
				b.text.SetText(fmt.Sprintf("Button:%d", i))
				b.OnClick = func() {
					counter++
					b.text.SetText(fmt.Sprintf("Counter:%d", counter))
				}
				root.Add(&b)
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
