package main

import (
	"fmt"
	"os"

	"github.com/Konstantin8105/vl"
	"github.com/gdamore/tcell/v2"
)

func main() {
	action := make(chan func(), 10)
	demo := vl.Demo()
	root := demo[0]
	// start web server on port 8080
	fmt.Fprintln(os.Stdout, "Web demo at http://localhost:8080")
	err := vl.WebRun(":8080", root, action, nil, tcell.KeyCtrlC)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
