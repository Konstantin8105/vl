package vl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
				l.Add(nil)
				var fr Frame
				var chfr CheckBox
				chfr.SetText("Frame header")
				fr.Header = &chfr // TextStatic("Frame header")
				var secFr Frame
				secFr.Header = TextStatic("Second header with long multiline\nNo addition options")
				secFr.Root = TextStatic(texts[ti])
				fr.Root = &secFr
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

				var in Inputbox
				in.SetText("Some inputbox text")
				l.Add(&in)

				r.Root = &l
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
			screen.Root = rt
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
func Benchmark(b *testing.B) {
	var screen Screen
	r, _ := roots[len(roots)-1].generate()
	screen.Root = r
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
