//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/Konstantin8105/vl"
	"github.com/gdamore/tcell/v2"
)

func main() {
	if true {
		// vl.SpecificSymbol(false)
		action := make(chan func(), 10)
		root := vl.Demo()[0]
		err := vl.Run(root, action, nil, tcell.KeyCtrlC)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
		return
	}
	action := make(chan func(), 10)
	scroll := new(vl.Scroll)
	list := new(vl.List)
	scroll.SetRoot(list)
	for i := 0; i < 1000; i++ {
		str := fmt.Sprintf("%d MouseFlags are options to modify the handling", i)
		list.Add(vl.TextStatic(str))
	}
	root := scroll
	err := vl.Run(root, action, nil, tcell.KeyCtrlC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	return

}
