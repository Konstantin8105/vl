package vl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

const (
	testdata  = "testdata"
	errorRune = rune('#')
)

var (
	sizes = []uint{0, 1, 2, 5, 7, 10, 30}
	texts = []string{"", "Lorem", "Instead, they use ModAlt, even for events that could possibly have been distinguished from ModAlt.", `Название языка, выбранное компанией Google, практически совпадает с названием языка программирования Go!, созданного Ф. Джи. МакКейбом и К. Л. Кларком в 2003 году[9]. Обсуждение названия ведётся на странице, посвящённой Go[9].
На домашней странице языка и вообще в Интернет-публикациях часто используется альтернативное название — «golang»`}
)

type Root struct {
	name     string
	generate func() Widget
}

var roots = []Root{
	{"nil", func() Widget { return nil }},
}

func init() {
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name:     fmt.Sprintf("justtext%03d", ti),
			generate: func() Widget { return TextStatic(texts[ti]) },
		})
	}
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name: fmt.Sprintf("ScrollWithText%03d", ti),
			generate: func() Widget {
				var r Scroll
				var l List
				l.Add(TextStatic(texts[ti]))
				r.Root = &l
				return &r
			},
		})
	}
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name: fmt.Sprintf("ScrollWithDoubleText%03d", ti),
			generate: func() Widget {
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
				fr.Header = TextStatic("Frame header")
				var secFr Frame
				secFr.Header = TextStatic("Second header with long multiline\nNo addition options")
				secFr.Root = TextStatic(texts[ti])
				fr.Root = &secFr
				l.Add(&fr)
				l.Add(&b)

				var rg RadioGroup
				rg.SetText([]string{"one", "two", "three"})
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
				return &r
			},
		})
	}
}

func Test(t *testing.T) {
	for si := range sizes {
		for ri := range roots {
			name := fmt.Sprintf("%03d-%03d-%s", sizes[si], ri, roots[ri].name)
			t.Run(name, func(t *testing.T) {
				check(t, name, si, roots[ri].generate())
			})
		}
	}
}

func check(t *testing.T, name string, si int, root Widget) {
	b := Screen{
		Width:  sizes[si],
		Height: sizes[si] / 2,
		Root:   root,
	}
	t.Logf("Screen size: width=%d height=%d", b.Width, b.Height)

	var buf bytes.Buffer

	// compare
	defer func() {
		var (
			actual   = buf.Bytes()
			filename = filepath.Join(testdata, name)
		)
		// for update test screens run in console:
		// UPDATE=true go test
		if os.Getenv("UPDATE") == "true" {
			if err := ioutil.WriteFile(filename, actual, 0644); err != nil {
				t.Fatalf("Cannot write snapshot to file: %v", err)
			}
		}
		// get expect result
		expect, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("Cannot read snapshot file: %v", err)
		}
		// compare
		if !bytes.Equal(actual, expect) {
			f2 := filename + ".new"
			if err := ioutil.WriteFile(f2, actual, 0644); err != nil {
				t.Fatalf("Cannot write snapshot to file new: %v", err)
			}
			t.Errorf("Snapshots is not same:\nActual:\n%s\nExpect:\n%s\nmeld %s %s",
				actual,
				expect,
				filename, f2,
			)
		}
	}()

	var db Buffer

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
	for i := 0; i < 18; i++ {
		for j := 0; j < 18; j++ {
			move = append(move, Event{
				name: fmt.Sprintf("Click%02d-%02d", i, j),
				ev:   tcell.NewEventMouse(i, j, tcell.Button1, tcell.ModNone),
			})
		}
	}

	for i := range move {
		fmt.Fprintf(&buf, "Move: %s\n", move[i].name)
		if e := move[i].ev; e != nil {
			b.Event(e)
		}
		b.Render(b.Width, db.Drawer)
		if db.ErrorRune() {
			t.Errorf("error rune")
		}
		if len(db.m) != int(b.Height) {
			t.Errorf("height is not valid: %d %d", len(db.m), int(b.Height))
		}
		for r := range db.m {
			if len(db.m[r]) != int(b.Width) {
				t.Errorf("width is not valid: %d %d", len(db.m[r]), int(b.Width))
			}
		}
		fmt.Fprintf(&buf, "%s", db)
	}
}

type Buffer struct {
	m [][]rune
}

func (b *Buffer) Drawer(row, col uint, s tcell.Style, r rune) {
	for i := len(b.m); i <= int(row); i++ {
		b.m = append(b.m, make([]rune, 0))
	}
	for i := len(b.m[row]); i <= int(col); i++ {
		b.m[row] = append(b.m[row], errorRune)
	}
	b.m[row][col] = r
}

func (b Buffer) String() string {
	var str string
	var w int
	for r := range b.m {
		str += fmt.Sprintf("%09d|", r+1)
		for c := range b.m[r] {
			str += string(b.m[r][c])
		}
		if width := len(b.m[r]); w < width {
			w = width
		}
		str += fmt.Sprintf("| width:%09d\n", len(b.m[r]))
	}
	str += fmt.Sprintf("rows  = %3d\n", len(b.m))
	str += fmt.Sprintf("width = %3d\n", w)
	return str
}

func (b Buffer) Text() string {
	var str string
	for r := range b.m {
		for c := range b.m[r] {
			str += string(b.m[r][c])
		}
		str += "\n"
	}
	return str
}

func (b Buffer) ErrorRune() bool {
	for r := range b.m {
		for c := range b.m[r] {
			if b.m[r][c] == errorRune {
				return true
			}
		}
	}
	return false
}
