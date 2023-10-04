package vl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Konstantin8105/compare"
	"github.com/gdamore/tcell/v2"
)

const (
	testdata  = "testdata"
	errorRune = rune('#')
)

var (
	sizes = []uint{0, 1, 2, 7, 40}
	texts = []string{"", "Lorem", "Instead, they use ModAlt, even for events that could possibly have been distinguished from ModAlt."}
)

type Root struct {
	name     string
	generate func() (root Widget, action chan func())
}

var roots = []Root{
	{"nil", func() (Widget, chan func()) { return nil, nil }},
}

func init() {
	{
		ws := Demo()
		for i := range ws {
			i := i
			roots = append(roots, Root{
				name: fmt.Sprintf("Demo%03d", i),
				generate: func() (Widget, chan func()) {
					action := make(chan func(), 10)
					return ws[i], action
				},
			})
			break
		}
	}
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name:     fmt.Sprintf("justtext%03d", ti),
			generate: func() (Widget, chan func()) { return TextStatic(texts[ti]), nil },
		})
	}
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name: fmt.Sprintf("ScrollWithDoubleText%03d", ti),
			generate: func() (Widget, chan func()) {
				var (
					r Scroll
					l List
					b Button
				)
				b.SetText(texts[ti])
				b.OnClick = func() {}
				l.Add(&b)
				l.Add(TextStatic(texts[ti]))
				l.Add(new(Separator))
				l.Add(nil)
				var fr Frame
				var chfr CheckBox
				chfr.SetText("Frame header")
				fr.Header = &chfr // TextStatic("Frame header")
				var secFr Frame
				secFr.Header = TextStatic("Second header with long multiline\nNo addition options")
				secFr.SetRoot(TextStatic(texts[ti]))
				fr.SetRoot(&secFr)
				l.Add(&fr)
				l.Add(&b)

				var rg RadioGroup
				rg.AddText([]string{"one", "two", "three"}...)
				l.Add(&rg)

				var ch CheckBox
				ch.SetText("checkbox 1")
				ch.Checked = true
				l.Add(&ch)

				var ch2 CheckBox
				ch2.SetText("checkbox 2")
				ch2.Checked = false
				l.Add(&ch2)

				var in InputBox
				in.SetText("Some inputbox text")
				l.Add(&in)

				r.SetRoot(&l)
				return &r, nil
			},
		})
	}
}

func Test(t *testing.T) {
	run := func(si, ri int) {
		name := fmt.Sprintf("%03d-%03d-%s", sizes[si], ri, roots[ri].name)
		t.Run(name, func(t *testing.T) {
			rt, ac := roots[ri].generate()
			if _, ok := rt.(*Separator); ok {
				return
			}
			go func() {
				for {
					isbreak := false
					select {
					case f, ok := <-ac:
						if !ok {
							isbreak = true
						}
						f()
					}
					if isbreak {
						break
					}
				}
			}()
			var screen Screen
			screen.SetRoot(rt)
			check(t, name, si, screen)
		})
	}
	for si := range sizes {
		for ri := range roots {
			run(si, ri)
		}
	}
}

func check(t *testing.T, name string, si int, screen Screen) {
	width := sizes[si]
	height := sizes[si]

	var buf bytes.Buffer

	// compare
	defer func() {
		filename := filepath.Join(testdata, name)
		compare.Test(t, filename, buf.Bytes())
	}()

	// var db Buffer

	type Event struct {
		name string
		ev   tcell.Event
	}
	var move = []Event{
		{ // 0
			name: "none",
			ev:   nil,
		},
		{ // 1
			name: "WheelUp",
			ev:   tcell.NewEventMouse(0, 0, tcell.WheelUp, tcell.ModNone),
		},
		{ // 2
			name: "WheelDown",
			ev:   tcell.NewEventMouse(0, 0, tcell.WheelDown, tcell.ModNone),
		},
		{ // 3
			name: "Click",
			ev:   tcell.NewEventMouse(1, 1, tcell.Button1, tcell.ModNone),
		},
		{ // 4
			name: "InputRune",
			ev:   tcell.NewEventKey(0, 'W', tcell.ModNone),
		},
		{ // 5
			name: "Right",
			ev:   tcell.NewEventKey(tcell.KeyRight, ' ', tcell.ModNone),
		},
		{ // 6
			name: "Left",
			ev:   tcell.NewEventKey(tcell.KeyLeft, ' ', tcell.ModNone),
		},
	}

	for i := 0; i < 6; i++ {
		move = append(move, move[2])
	}
	for i := 0; i < 8; i++ {
		move = append(move, move[1])
	}
	for i := 0; i < 2; i++ {
		move = append(move, move[3], move[4], move[4], move[2])
	}
	for i := 0; i < 2; i++ {
		move = append(move, move[5], move[6])
	}
	for i := -2; i < int(sizes[si]+1); i++ {
		for j := -2; j < int(sizes[si]+1); j += 5 {
			move = append(move, Event{
				name: fmt.Sprintf("Click%02d-%02d", i, j),
				ev:   tcell.NewEventMouse(i, j, tcell.Button1, tcell.ModNone),
			})
			move = append(move, move[4], move[5], move[6])
		}
	}

	// move = move[:1] // TODO remove

	cells := new([][]Cell)

	for i := range move {
		fmt.Fprintf(&buf, "Pos %04d. Move: %s\n", i, move[i].name)
		if e := move[i].ev; e != nil {
			screen.Event(e)
		}
		screen.SetHeight(height)
		screen.GetContents(width, cells)
		if len(*cells) != int(height) {
			t.Fatalf("height is not valid: %d %d", len(*cells), int(height))
		}
		for r := range *cells {
			if len((*cells)[r]) != int(width) {
				t.Errorf("width is not valid: %d %d", len((*cells)[r]), int(width))
			}
		}
		fmt.Fprintf(&buf, "%s", Convert(*cells))
	}
}

func TestRun(t *testing.T) {
	simulation = true
	defer func() {
		simulation = false
	}()
	t.Run("exit by key", func(t *testing.T) {
		action := make(chan func(), 10)
		root := Demo()[0]
		go func() {
			<-time.After(time.Millisecond * 200)
			screen.(tcell.SimulationScreen).InjectKey(tcell.KeyCtrlC, ' ', tcell.ModNone)
		}()
		err := Run(root, action, nil, tcell.KeyCtrlC)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
	t.Run("exit by channel", func(t *testing.T) {
		qu := make(chan struct{})
		action := make(chan func(), 10)
		root := Demo()[0]
		go func() {
			<-time.After(time.Millisecond * 200)
			var closed struct{}
			qu <- closed
		}()
		err := Run(root, action, qu)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
	t.Run("exit by close channel", func(t *testing.T) {
		qu := make(chan struct{})
		action := make(chan func(), 10)
		root := Demo()[0]
		go func() {
			<-time.After(time.Millisecond * 200)
			close(qu)
		}()
		err := Run(root, action, qu)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
}

// goos: linux
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: Intel(R) Xeon(R) CPU E3-1240 V2 @ 3.40GHz
// Benchmark-4   	   15679	     72798 ns/op	     505 B/op	      19 allocs/op
// Benchmark-4   	   16251	     73508 ns/op	     537 B/op	      19 allocs/op
// Benchmark-4   	   16137	     73325 ns/op	     537 B/op	      19 allocs/op
// Benchmark-4   	   16294	     73189 ns/op	     537 B/op	      19 allocs/op
// Benchmark-4   	   16347	     73162 ns/op	     537 B/op	      19 allocs/op
// Benchmark-4   	   16138	     74406 ns/op	     537 B/op	      19 allocs/op
// Benchmark/Size020-4         	   39470	     29108 ns/op	     536 B/op	      19 allocs/op
// Benchmark/Size040-4         	   33872	     35148 ns/op	     536 B/op	      19 allocs/op
// Benchmark/Size080-4         	   20626	     56924 ns/op	     536 B/op	      19 allocs/op
//
// Benchmark/Size020-4         	   23887	     52026 ns/op	    1424 B/op	      50 allocs/op
// Benchmark/Size040-4         	   21730	     55157 ns/op	    1424 B/op	      50 allocs/op
// Benchmark/Size080-4         	   15016	     81002 ns/op	    1425 B/op	      50 allocs/op
//
// Benchmark/Size020-4         	   23868	     50256 ns/op	    1696 B/op	      40 allocs/op
// Benchmark/Size040-4         	   20026	     54741 ns/op	    1696 B/op	      40 allocs/op
// Benchmark/Size080-4         	   14938	     78226 ns/op	    1697 B/op	      40 allocs/op
//
// Benchmark/Size020-4         	   22135	     51894 ns/op	    2080 B/op	      46 allocs/op
// Benchmark/Size040-4         	   19011	     62061 ns/op	    2080 B/op	      46 allocs/op
// Benchmark/Size080-4         	   14752	     88859 ns/op	    2082 B/op	      46 allocs/op
func Benchmark(b *testing.B) {
	var screen Screen
	r, _ := roots[len(roots)-1].generate()
	screen.SetRoot(r)
	for _, size := range []uint{20, 40, 80} {
		b.Run(fmt.Sprintf("Size%03d", size), func(b *testing.B) {
			screen.SetHeight(size)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = screen.Render(size, NilDrawer)
			}
		})
	}
}

func TestAscii(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !utf8.Valid(content) {
			t.Fatalf("utf8 invalid")
		}
		runes := []rune(string(content))
		for _, r := range runes {
			ir := int(r)
			if 32 <= ir && ir <= 127 {
				continue
			}
			if ir == int('\n') {
				continue
			}
			if ir == int('\t') {
				continue
			}
			t.Errorf("find unicode: `%s`", string(r))
		}
	}
}

func TestWidget(t *testing.T) {
	list := func() []Widget {
		return []Widget{
			new(Separator),
			new(Text),
			new(Scroll),
			new(List),
			new(Menu),
			new(Button),
			new(Frame),
			new(RadioGroup),
			new(CheckBox),
			new(InputBox),
			new(CollapsingHeader),
			new(ListH),
			new(ComboBox),
			new(Tabs),
			new(Tree),
		}
	}
	getName := func(w Widget) string {
		name := fmt.Sprintf("%T", w)
		name = strings.ReplaceAll(name, "*vl.", "")
		return name
	}
	type tcase struct {
		name string
		w    Widget
	}
	var tcs []tcase
	for _, w := range list() {
		tcs = append(tcs, tcase{name: getName(w), w: w})
	}
	for it := range texts {
		for _, w := range list() {
			c, ok := w.(interface {
				SetText(string)
				GetText() string
			})
			if !ok {
				continue
			}
			c.SetText(texts[it])
			name := fmt.Sprintf("%s-SetText%02d", getName(w), it)
			tcs = append(tcs, tcase{name: name, w: w})
			t.Run(getName(w)+"PrepareGetText", func(t *testing.T) {
				if texts[it] != c.GetText() {
					t.Errorf("not same")
				}
			})
		}
	}
	for _, w := range list() {
		if _, ok := w.(*RadioGroup); ok {
			continue
		}
		c, ok := w.(interface {
			Add(Widget)
		})
		if !ok {
			continue
		}
		c.Add(TextStatic("Second text"))
		name := fmt.Sprintf("%s-Add", getName(w))
		tcs = append(tcs, tcase{name: name, w: w})
		t.Run(getName(w)+"PrepareAdd", func(t *testing.T) {
			c, ok := w.(interface {
				Size() int
				Clear()
			})
			if !ok {
				t.Fatalf("Not enought function")
			}
			if c.Size() != 1 {
				t.Errorf("not valid size")
			}
		})
	}
	for _, w := range list() {
		if _, ok := w.(*RadioGroup); ok {
			continue
		}
		c, ok := w.(interface {
			Add(Widget)
		})
		if !ok {
			continue
		}
		c.Add(TextStatic("Second text"))
		var value int
		var btn Button
		btn.SetText("Under root")
		btn.OnClick = func() {
			value += 1
			btn.SetText(fmt.Sprintf("%s%d", btn.GetText(), value))
		}
		c.Add(&btn)
		name := fmt.Sprintf("%s-Add2", getName(w))
		tcs = append(tcs, tcase{name: name, w: w})
		t.Run(getName(w)+"PrepareAdd2", func(t *testing.T) {
			c, ok := w.(interface {
				Size() int
				Clear()
			})
			if !ok {
				t.Fatalf("Not enought function")
			}
			if c.Size() != 2 {
				t.Errorf("not valid size")
			}
		})
	}
	for _, w := range list() {
		c, ok := w.(interface {
			Compress()
		})
		if !ok {
			continue
		}
		c.Compress()
		name := fmt.Sprintf("%s-Compress", getName(w))
		tcs = append(tcs, tcase{name: name, w: w})
	}
	for _, w := range list() {
		c, ok := w.(interface {
			SetRoot(Widget)
		})
		if !ok {
			continue
		}
		var value int
		var btn Button
		btn.SetText("Under root")
		btn.OnClick = func() {
			value += 1
			btn.SetText(fmt.Sprintf("%s%d", btn.GetText(), value))
		}
		c.SetRoot(&btn)
		name := fmt.Sprintf("%s-SetRoot", getName(w))
		tcs = append(tcs, tcase{name: name, w: w})
	}
	for _, w := range list() {
		c, ok := w.(interface {
			SetRoot(Widget)
		})
		if !ok {
			continue
		}
		var rg RadioGroup
		rg.AddText("radio0", "radio1")
		c.SetRoot(&rg)
		name := fmt.Sprintf("%s-SetRootRadiGroup", getName(w))
		tcs = append(tcs, tcase{name: name, w: w})
	}
	for _, tc := range tcs {
		for _, size := range sizes {
			name := fmt.Sprintf("Widget-%02d-%s", size, tc.name)
			t.Run(name, func(t *testing.T) {
				cells := new([][]Cell)
				height := size
				width := size
				var screen Screen
				screen.SetRoot(tc.w)
				screen.SetHeight(size)

				// first shot
				screen.GetContents(width, cells)
				if len(*cells) != int(height) {
					t.Fatalf("height is not valid: %d %d", len(*cells), int(height))
				}
				for r := range *cells {
					if len((*cells)[r]) != int(width) {
						t.Errorf("width is not valid: %d %d", len((*cells)[r]), int(width))
					}
				}
				var buf bytes.Buffer
				fmt.Fprintf(&buf, "%s", Convert(*cells))

				// click on field
				var x, y uint
				var found bool
				for x = 0; x < width; x++ {
					for y = 0; y < height; y++ {
						if (*cells)[x][y].S == ButtonStyle ||
							(*cells)[x][y].S == InputBoxStyle {
							found = true
						}
						if found {
							break
						}
					}
					if found {
						break
					}
				}
				if found {
				} else {
					x, y = 1, 1
					t.Logf("not clicked")
				}
				for i := 0; i < 2; i++ {
					click := tcell.NewEventMouse(
						int(x), int(y),
						tcell.Button1, tcell.ModNone)
					screen.Event(click)
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))
				}

				// click left at field

				// resize
				{
					width += 4
					height += 4
					screen.SetHeight(size)
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))
				}
				{
					width -= 4
					height -= 4
					screen.SetHeight(size)
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))
				}

				// testing
				if width < 4 {
					return
				}
				// fmt.Println(buf.String())
				filename := filepath.Join(testdata, name)
				compare.Test(t, filename, buf.Bytes())
			})
		}
	}
}
