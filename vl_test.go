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
	texts = []string{"", "Lorem", `Название языка, выбранное компанией Google, практически совпадает с названием языка программирования Go!, созданного Ф. Джи. МакКейбом и К. Л. Кларком в 2003 году[9]. Обсуждение названия ведётся на странице, посвящённой Go[9].
На домашней странице языка и вообще в Интернет-публикациях часто используется альтернативное название — «golang»`}
)

var roots = []Widget{
	nil,
}

func init() {
	for ti := range texts {
		roots = append(roots, TextStatic(texts[ti]))
	}
}

func Test(t *testing.T) {
	for si := range sizes {
		for ri := range roots {
			name := fmt.Sprintf("%03d-%03d", sizes[si], ri)
			t.Run(name, func(t *testing.T) {
				check(t, name, si, roots[ri])
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

	var db Buffer
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

	fmt.Fprintf(&buf, "%s", db)
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
