//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Konstantin8105/vl"
	"github.com/gdamore/tcell/v2"
)

func main() {
	var err error
	input := 0
	switch input {
	case 0:
		action := make(chan func(), 10)
		root := vl.Demo()[0]
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 1:
		action := make(chan func(), 10)
		scroll := new(vl.Scroll)
		list := new(vl.List)
		scroll.SetRoot(list)
		for i := range 1000 {
			str := fmt.Sprintf("%d MouseFlags are options to modify the handling", i)
			list.Add(vl.TextStatic(str))
		}
		root := scroll
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 2:
		action := make(chan func(), 10)
		c := new(vl.ComboBox)
		c.Add(strings.Repeat("fd gsdfg fdg sdfg", 5))
		for i := range 5 {
			// c.Add(strings.Repeat("AB", 100/(i+1)))
			c.Add(strings.Repeat("AB", (i+1)*20))
		}
		root := c
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 3:
		action := make(chan func(), 10)
		rg := new(vl.RadioGroup)
		rg.AddText(strings.Repeat("fd gsdfg fdg sdfg", 5))
		for i := range 5 {
			rg.AddText(strings.Repeat("AB", (i+1)*20))
		}
		root := rg
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 4:
		action := make(chan func(), 10)
		txt := vl.TextStatic(strings.Repeat("ABCDabcd", 500000))
		scroll := new(vl.Scroll)
		scroll.SetRoot(txt)
		root := scroll
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 5:
		action := make(chan func(), 10)
		fr := new(vl.Frame)
		// fr.NoBorder = true
		fr.Header = vl.TextStatic(strings.Repeat("AWSDEQASWED", 5))
		{
			sc := new(vl.Scroll)
			list := new(vl.List)
			for i := range 100 {
				list.Add(vl.TextStatic(fmt.Sprintf("ABCDabcd:%d", i)))
			}
			sc.SetRoot(list)
			fr.SetRoot(sc)
		}
		root := fr
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 6:
		action := make(chan func(), 10)
		list := new(vl.List)
		for range 10 {
			ch := new(vl.CollapsingHeader)
			ch.SetText("asdas d asd as d\n sdf sdf")
			sl := new(vl.List)
			for i := range 100 {
				sl.Add(vl.TextStatic(fmt.Sprintf("ABCDabcd:%d", i)))
			}
			// ch.BorderIfClosed(false)
			ch.SetRoot(sl)
			list.Add(ch)
		}
		tabs := new(vl.Tabs)
		tabs.UseCombo(true)

		sc := new(vl.Scroll)
		sc.SetRoot(list)
		tabs.Add("BBB", sc)
		root := tabs
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	case 7:
		list := new(vl.List)
		for i := range 10 {
			ch := new(vl.CollapsingHeader)
			ch.SetText(fmt.Sprintf("Name %d", i))
			ch.SetRoot(vl.TextStatic(strings.Repeat("Body big", 100)))
			list.Add(ch)
		}
		scroll := new(vl.Scroll)
		scroll.SetRoot(list)
		tabs := new(vl.Tabs)
		tabs.Add("Prompts", scroll)
		root := tabs
		action := make(chan func(), 10)
		err = vl.Run(root, action, nil, tcell.KeyCtrlC)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
