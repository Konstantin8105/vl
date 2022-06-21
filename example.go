//go:build ignore

package main

import (
	"time"

	"github.com/Konstantin8105/vl"
	"github.com/gdamore/tcell/v2"
)

func main() {
	texts := "Lolly pop"
	var (
		r vl.Scroll
		l vl.List
		b vl.Button
	)
	b.SetText(texts)
	b.OnClick = func() {}
	l.Add(&b)
	l.Add(vl.TextStatic(texts))
	l.Add(nil)
	var fr vl.Frame
	var chfr vl.CheckBox
	chfr.SetText("Frame header")
	fr.Header = &chfr
	var secFr vl.Frame
	secFr.Header = vl.TextStatic("Second header with long multiline\nNo addition options")
	secFr.Root = vl.TextStatic(texts)
	fr.Root = &secFr
	l.Add(&fr)

	var rg vl.RadioGroup
	rg.SetText([]string{"one", "two", "three"})
	l.Add(&rg)

	var ch vl.CheckBox
	ch.SetText("checkbox 1")
	ch.Checked = true
	l.Add(&ch)

	var ch2 vl.CheckBox
	ch2.SetText("checkbox 2")
	ch2.Checked = false
	l.Add(&ch2)

	var in vl.Inputbox
	in.SetText("Some inputbox text")
	l.Add(&in)

	r.Root = &l

	var stop chan bool

	go func() {
		<-time.After(time.Second * 1)
		stop <- true
	}()

	vl.Run(&r, stop, tcell.KeyCtrlC)
}
