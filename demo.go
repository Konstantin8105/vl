//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/Konstantin8105/vl"
	"github.com/gdamore/tcell/v2"
)

func main() {
	err := vl.Run(vl.Demo(), tcell.KeyCtrlC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
