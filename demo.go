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
		for i := 0; i < 1000; i++ {
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
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
