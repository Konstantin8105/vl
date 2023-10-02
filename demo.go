//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/Konstantin8105/vl"
	"github.com/gdamore/tcell/v2"
)

func main() {
	action := make(chan func(), 10)
	root := vl.Demo()[0]
	err := vl.Run(root, action, nil, tcell.KeyCtrlC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
