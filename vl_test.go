package vl

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Konstantin8105/compare"
	"github.com/Konstantin8105/snippet"
	"github.com/gdamore/tcell/v2"
)

const (
	testdata = "testdata"
	// errorRune = rune('#')
)

var (
	sizes = []uint{0, 1, 2, 7, 40}
	texts = []string{"", "Lorem", "Instead, they use ModAlt, even for events that could possibly have been distinguished from ModAlt."}
)

type Root struct {
	name     string
	generate func() (root Widget) // , action chan func())
}

var roots = []Root{
	{"nil", func() Widget { return nil }},
}

func init() {
	{
		ws := Demo()
		for i := range ws {
			i := i
			roots = append(roots, Root{
				name: fmt.Sprintf("Demo%03d", i),
				generate: func() Widget {
					// action := make(chan func(), 10)
					return ws[i] // , action
				},
			})
			break
		}
	}
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name: fmt.Sprintf("justtext%03d", ti),
			generate: func() Widget { //, chan func()) {
				return TextStatic(texts[ti])
			}, // , nil },
		})
	}
	for ti := range texts {
		ti := ti
		roots = append(roots, Root{
			name: fmt.Sprintf("ScrollWithDoubleText%03d", ti),
			generate: func() Widget { // , chan func()) {
				var (
					r Scroll
					l List
					b Button
				)
				b.SetText(texts[ti])
				b.OnClick = func() {}
				l.Add(&b)
				l.Add(TextStatic(texts[ti]))
				l.Add(new(Separator))
				l.Add(nil)
				var fr Frame
				var chfr CheckBox
				chfr.SetText("Frame header")
				fr.Header = &chfr // TextStatic("Frame header")
				var secFr Frame
				secFr.Header = TextStatic("Second header with long multiline\nNo addition options")
				secFr.SetRoot(TextStatic(texts[ti]))
				fr.SetRoot(&secFr)
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

				var in InputBox
				in.SetText("Some inputbox text")
				l.Add(&in)

				r.SetRoot(&l)
				return &r // , nil
			},
		})
	}
}

func Test(t *testing.T) {
	run := func(si, ri int) {
		name := fmt.Sprintf("%03d-%03d-%s", sizes[si], ri, roots[ri].name)
		t.Run(name, func(t *testing.T) {
			rt := roots[ri].generate()
			if _, ok := rt.(*Separator); ok {
				return
			}
			// go func() {
			// 	for {
			// 		isbreak := false
			// 		select {
			// 		case f, ok := <-ac:
			// 			if !ok {
			// 				isbreak = true
			// 			}
			// 			f()
			// 		}
			// 		if isbreak {
			// 			break
			// 		}
			// 	}
			// }()
			var screen Screen
			screen.SetRoot(rt)
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
		if width < 4 {
			return
		}
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

	for range 6 {
		move = append(move, move[2])
	}
	for range 8 {
		move = append(move, move[1])
	}
	for range 2 {
		move = append(move, move[3], move[4], move[4], move[2])
	}
	for range 2 {
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
// Benchmark/Size020-4         	   22135	     51894 ns/op	    2080 B/op	      46 allocs/op
// Benchmark/Size040-4         	   19011	     62061 ns/op	    2080 B/op	      46 allocs/op
// Benchmark/Size080-4         	   14752	     88859 ns/op	    2082 B/op	      46 allocs/op
//
// Benchmark/Size020-4         	   16173	     76793 ns/op	    2145 B/op	      47 allocs/op
// Benchmark/Size040-4         	   13602	     87016 ns/op	    2145 B/op	      47 allocs/op
// Benchmark/Size080-4         	    9872	    119570 ns/op	    2145 B/op	      47 allocs/op
//
// Benchmark/Size020-4         	   18955	     63914 ns/op	    2145 B/op	      47 allocs/op
// Benchmark/Size040-4         	   15909	     74452 ns/op	    2145 B/op	      47 allocs/op
// Benchmark/Size080-4         	   10000	    108412 ns/op	    2146 B/op	      47 allocs/op
//
// Benchmark/Size020-8         	    6565	    207867 ns/op	    2810 B/op	      67 allocs/op
// Benchmark/Size040-8         	    6475	    210451 ns/op	    2681 B/op	      63 allocs/op
// Benchmark/Size080-8         	    4423	    271537 ns/op	    2683 B/op	      63 allocs/op
//
// Benchmark/Size020-8         	    5998	    197053 ns/op	    2051 B/op	      45 allocs/op
// Benchmark/Size040-8         	    5430	    207056 ns/op	    2052 B/op	      45 allocs/op
// Benchmark/Size080-8         	    4108	    253947 ns/op	    2050 B/op	      45 allocs/op
//
// Benchmark/Size020-8         	    6180	    194175 ns/op	    2052 B/op	      45 allocs/op
// Benchmark/Size040-8         	    4893	    206666 ns/op	    2053 B/op	      45 allocs/op
// Benchmark/Size080-8         	    3670	    281945 ns/op	    2050 B/op	      45 allocs/op
// Benchmark/Separato-8        	   23418	     54807 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Text-8            	   21466	     53051 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Scroll-8          	   22903	     52856 ns/op	      32 B/op	       1 allocs/op
// Benchmark/List-8            	   23420	     51074 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Menu-8            	   22417	     51086 ns/op	      64 B/op	       2 allocs/op
// Benchmark/Button-8          	   22557	     54279 ns/op	      96 B/op	       2 allocs/op
// Benchmark/Frame-8           	   19860	     58808 ns/op	      64 B/op	       2 allocs/op
// Benchmark/RadioGro-8        	   22983	     52854 ns/op	      32 B/op	       1 allocs/op
// Benchmark/CheckBox-8        	   21762	     53880 ns/op	      32 B/op	       1 allocs/op
// Benchmark/InputBox-8        	   21427	     53540 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Collapsi-8        	   19938	     60839 ns/op	      88 B/op	       3 allocs/op
// Benchmark/ListH-8           	   22177	     51075 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ComboBox-8        	   19702	     59992 ns/op	      88 B/op	       3 allocs/op
// Benchmark/Tabs-8            	   20982	     58993 ns/op	      64 B/op	       2 allocs/op
// Benchmark/Tree-8            	   22326	     51231 ns/op	      40 B/op	       2 allocs/op
//
// Benchmark/Size020-4         	   10000	    118478 ns/op	    1795 B/op	      41 allocs/op
// Benchmark/Size040-4         	    8124	    130859 ns/op	    1795 B/op	      41 allocs/op
// Benchmark/Size080-4         	    6294	    172075 ns/op	    1794 B/op	      41 allocs/op
// Benchmark/Separato-4        	   31983	     38224 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Text-4            	   30786	     41011 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Scroll-4          	   31495	     39544 ns/op	      32 B/op	       1 allocs/op
// Benchmark/List-4            	   30926	     38390 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Menu-4            	   30685	     38228 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Button-4          	   28232	     40346 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Frame-4           	   28732	     42876 ns/op	      64 B/op	       2 allocs/op
// Benchmark/RadioGro-4        	   31024	     38564 ns/op	      32 B/op	       1 allocs/op
// Benchmark/CheckBox-4        	   29526	     39660 ns/op	      32 B/op	       1 allocs/op
// Benchmark/InputBox-4        	   29588	     38979 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Collapsi-4        	   27289	     43714 ns/op	      88 B/op	       3 allocs/op
// Benchmark/ListH-4           	   31335	     39069 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ComboBox-4        	   27135	     43895 ns/op	      88 B/op	       3 allocs/op
// Benchmark/Tabs-4            	   28226	     43049 ns/op	      64 B/op	       2 allocs/op
// Benchmark/Tree-4            	   30840	     37779 ns/op	      40 B/op	       2 allocs/op
//
// Benchmark/Size020-4         	    8266	    131666 ns/op	    2434 B/op	      41 allocs/op
// Benchmark/Size040-4         	    9135	    145425 ns/op	    2434 B/op	      41 allocs/op
// Benchmark/Size080-4         	    5845	    180644 ns/op	    2438 B/op	      41 allocs/op
// Benchmark/Separato-4        	   27798	     44193 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Text-4            	   29014	     43097 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Scroll-4          	   28222	     40916 ns/op	      32 B/op	       1 allocs/op
// Benchmark/List-4            	   28788	     40608 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Menu-4            	   30699	     40568 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Button-4          	   29787	     42412 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Frame-4           	   27374	     46007 ns/op	      64 B/op	       2 allocs/op
// Benchmark/RadioGro-4        	   29445	     41844 ns/op	      32 B/op	       1 allocs/op
// Benchmark/CheckBox-4        	   29716	     40744 ns/op	      32 B/op	       1 allocs/op
// Benchmark/InputBox-4        	   28488	     41075 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Collapsi-4        	   24603	     47008 ns/op	     128 B/op	       3 allocs/op
// Benchmark/ListH-4           	   28605	     39280 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ComboBox-4        	   25687	     46629 ns/op	     128 B/op	       3 allocs/op
// Benchmark/Tabs-4            	   27115	     45441 ns/op	      64 B/op	       2 allocs/op
// Benchmark/Tree-4            	   28879	     40285 ns/op	      40 B/op	       2 allocs/op
// Benchmark/Viewer-4          	   28900	     42974 ns/op	      32 B/op	       1 allocs/op
//
// cpu: Intel(R) Xeon(R) CPU E3-1240 V2 @ 3.40GHz
// Benchmark/ViewerP-4         	     966	   1071589 ns/op	  690830 B/op	    3042 allocs/op
// Benchmark/ViewerP-4         	    1597	    765777 ns/op	  684385 B/op	    3035 allocs/op
// Benchmark/ViewerP-4         	    1626	    627755 ns/op	  490629 B/op	    3020 allocs/op
//
// cpu: Intel(R) Xeon(R) CPU E3-1240 V2 @ 3.40GHz
// Benchmark/Size020-4     	    9613	    119436 ns/op	    2435 B/op	      41 allocs/op
// Benchmark/Size040-4     	    9499	    122297 ns/op	    2435 B/op	      41 allocs/op
// Benchmark/Size080-4     	    7232	    150581 ns/op	    2434 B/op	      41 allocs/op
// Benchmark/Separato-4    	16031305	        66.11 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Text-4        	  866361	      1298 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Scroll-4      	14891812	        80.20 ns/op	      32 B/op	       1 allocs/op
// Benchmark/List-4        	15000658	        77.54 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Menu-4        	13583463	        78.01 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Button-4      	  777271	      1564 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Frame-4       	  241263	      4729 ns/op	      64 B/op	       2 allocs/op
// Benchmark/RadioGro-4    	16075549	        82.66 ns/op	      32 B/op	       1 allocs/op
// Benchmark/CheckBox-4    	  734319	      1652 ns/op	      32 B/op	       1 allocs/op
// Benchmark/InputBox-4    	 1000000	      1117 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Collapsi-4    	  173040	      6746 ns/op	     128 B/op	       3 allocs/op
// Benchmark/ListH-4       	16226559	        69.16 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ComboBox-4    	  181116	      6865 ns/op	     128 B/op	       3 allocs/op
// Benchmark/Tabs-4        	  253702	      4817 ns/op	      64 B/op	       2 allocs/op
// Benchmark/Tree-4        	10501458	       118.6 ns/op	      40 B/op	       2 allocs/op
// Benchmark/Viewer-4      	  940558	      1306 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Image-4       	 5806833	       212.3 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ViewerP-4     	    2143	    674362 ns/op	  490648 B/op	    3020 allocs/op
//
// Benchmark/Size020-4     	   13375	     88755 ns/op	    2433 B/op	      41 allocs/op
// Benchmark/Size040-4     	   12421	     95885 ns/op	    2432 B/op	      41 allocs/op
// Benchmark/Size080-4     	    9189	    127487 ns/op	    2436 B/op	      41 allocs/op
// Benchmark/Separato-4    	15811003	        67.50 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Text-4        	  813781	      1270 ns/op	      32 B/op	       1 allocs/op
// Benchmark/staticTe-4    	 5587471	       211.6 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Scroll-4      	14879220	        68.29 ns/op	      32 B/op	       1 allocs/op
// Benchmark/List-4        	15916612	        66.91 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Menu-4        	14948356	        75.95 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Button-4      	  766341	      1442 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Frame-4       	  256839	      4517 ns/op	      64 B/op	       2 allocs/op
// Benchmark/RadioGro-4    	15119870	        75.77 ns/op	      32 B/op	       1 allocs/op
// Benchmark/CheckBox-4    	  748864	      1625 ns/op	      32 B/op	       1 allocs/op
// Benchmark/InputBox-4    	 1000000	      1047 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Collapsi-4    	  184360	      6437 ns/op	     128 B/op	       3 allocs/op
// Benchmark/ListH-4       	18375650	        67.37 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ComboBox-4    	  185797	      6642 ns/op	     128 B/op	       3 allocs/op
// Benchmark/Tabs-4        	  236923	      4631 ns/op	      64 B/op	       2 allocs/op
// Benchmark/Tree-4        	10791748	       110.7 ns/op	      40 B/op	       2 allocs/op
// Benchmark/Viewer-4      	 1000000	      1199 ns/op	      32 B/op	       1 allocs/op
// Benchmark/Image-4       	 5850693	       197.7 ns/op	      32 B/op	       1 allocs/op
// Benchmark/ViewerP-4     	    2146	    656160 ns/op	  490612 B/op	    3020 allocs/op
// Benchmark/ViewerA-4     	   49357	     24009 ns/op	      32 B/op	       1 allocs/op
//
// Benchmark/ViewerP-4     	     874	   1288535 ns/op	  619009 B/op	    6625 allocs/op
// Benchmark/ViewerA-4     	   47877	     23791 ns/op	      32 B/op	       1 allocs/op
//
// Benchmark/ViewerP-4     	     969	   1125126 ns/op	  618942 B/op	    6625 allocs/op
// Benchmark/ViewerP-4     	    1020	   1190187 ns/op	  619066 B/op	    6630 allocs/op
// Benchmark/ViewerP-4     	     985	   1119502 ns/op	  619071 B/op	    6629 allocs/op
// Benchmark/ViewerP-4     	     993	   1268332 ns/op	  619076 B/op	    6629 allocs/op
// Benchmark/ViewerP-4     	    1036	   1094960 ns/op	  629146 B/op	    6626 allocs/op
// Benchmark/ViewerP-4     	     954	   1246085 ns/op	  629359 B/op	    6632 allocs/op
//
// goos: windows
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: Intel(R) Xeon(R) CPU E5-2660 v4 @ 2.00GHz
// Benchmark/Size020-28               13617             85129 ns/op            2256 B/op         47 allocs/op
// Benchmark/Size040-28                9783            126617 ns/op            2256 B/op         47 allocs/op
// Benchmark/Size080-28                6920            169185 ns/op            2256 B/op         47 allocs/op
// Benchmark/Separato-28            8624926               133.0 ns/op            48 B/op          1 allocs/op
// Benchmark/Text-28                 509203              2397 ns/op              48 B/op          1 allocs/op
// Benchmark/Static-28              3357193               360.0 ns/op            48 B/op          1 allocs/op
// Benchmark/Scroll-28             12647514               141.7 ns/op            48 B/op          1 allocs/op
// Benchmark/List-28                7370029               138.1 ns/op            48 B/op          1 allocs/op
// Benchmark/Menu-28                9520590               122.2 ns/op            48 B/op          1 allocs/op
// Benchmark/Button-28               362455              3075 ns/op              96 B/op          2 allocs/op
// Benchmark/Frame-28                162360              7140 ns/op              48 B/op          1 allocs/op
// Benchmark/RadioGro-28            8894574               138.4 ns/op            48 B/op          1 allocs/op
// Benchmark/CheckBox-28             407659              3050 ns/op              96 B/op          2 allocs/op
// Benchmark/InputBox-28             565615              2035 ns/op              48 B/op          1 allocs/op
// Benchmark/Collapsi-28             136394              8376 ns/op             192 B/op          4 allocs/op
// Benchmark/ListH-28              11462379               134.6 ns/op            48 B/op          1 allocs/op
// Benchmark/ComboBox-28             152752              8159 ns/op             192 B/op          4 allocs/op
// Benchmark/Tabs-28                 164522              7200 ns/op              48 B/op          1 allocs/op
// Benchmark/Tree-28                8258251               134.5 ns/op            48 B/op          1 allocs/op
// Benchmark/Viewer-28               569508              2014 ns/op              48 B/op          1 allocs/op
// Benchmark/Image-28               3527438               339.8 ns/op            48 B/op          1 allocs/op
// Benchmark/FrameNoB-28             501481              2355 ns/op              48 B/op          1 allocs/op
// Benchmark/Collapsi#01-28          360709              3358 ns/op             192 B/op          4 allocs/op
// Benchmark/ViewerP-28                 856           2074056 ns/op          711721 B/op       6632 allocs/op
// Benchmark/ViewerA-28               30216             39712 ns/op              48 B/op          1 allocs/op
func Benchmark(b *testing.B) {
	var screen Screen
	r := roots[len(roots)-1].generate()
	screen.SetRoot(r)
	screen.Fill(func(rune, tcell.Style) {}) // for avoid perfomance for reset screen
	for _, size := range []uint{20, 40, 80} {
		b.Run(fmt.Sprintf("Size%03d", size), func(b *testing.B) {
			screen.SetHeight(size)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				_ = screen.Render(size, NilDrawer)
			}
		})
	}
	size := uint(100)
	for _, w := range list() {
		screen.SetHeight(size)
		b.ResetTimer()
		screen.SetRoot(w)
		name := getName(w)
		if 8 < len(name) {
			name = name[:8]
		}
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_ = screen.Render(size, NilDrawer)
			}
		})
	}
	b.Run("ViewerP", func(b *testing.B) {
		v := new(Viewer)
		v.SetText(strings.Repeat(texts[len(texts)-1], 40))
		v.SetColorize(TypicalColorize(
			strings.Fields(strings.Repeat(texts[len(texts)-1], 40)),
			InputBoxStyle))
		screen.SetRoot(v)
		screen.SetHeight(size)
		width := uint(20)
		g := false
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			if g {
				width++
			} else {
				width--
			}
			g = !g
			_ = screen.Render(width, NilDrawer)
		}
	})
	b.Run("ViewerA", func(b *testing.B) {
		txt := strings.Repeat(texts[len(texts)-1], 40)
		width := uint(20)
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			v := new(Viewer)
			v.SetText(txt)
			_ = screen.Render(width, NilDrawer)
			_ = v
		}
	})
}

// goos: linux
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: Intel(R) Xeon(R) CPU E3-1240 V2 @ 3.40GHz
// BenchmarkTextScroll/render-4         	     172	   6684900 ns/op	   64112 B/op	    1002 allocs/op
// BenchmarkTextScroll/moving-4         	     171	   6726420 ns/op	   64123 B/op	    1002 allocs/op
//
// BenchmarkTextScroll/render-4         	     171	   6772686 ns/op	   64113 B/op	    1002 allocs/op
// BenchmarkTextScroll/moving-4         	     170	   6887713 ns/op	   64064 B/op	    1002 allocs/op
//
// BenchmarkTextScroll/render-4         	     344	   3378204 ns/op	   64078 B/op	    1002 allocs/op
// BenchmarkTextScroll/moving-4         	     342	   3338545 ns/op	   64127 B/op	    1002 allocs/op
// BenchmarkTextScroll/static-4         	     703	   1612608 ns/op	      64 B/op	       2 allocs/op
//
// PS V:\go\vl> go test -run=Bench  -bench=BenchmarkTextScroll -benchmem -cpuprofile cpu.out -memprofile mem.out
// goos: windows
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: Intel(R) Xeon(R) CPU E5-2660 v4 @ 2.00GHz
// BenchmarkTextScroll/render-28                224           5268533 ns/op           64087 B/op       1002 allocs/op
// BenchmarkTextScroll/moving-28                226           5099225 ns/op           64064 B/op       1002 allocs/op
// BenchmarkTextScroll/static-28                492           2354826 ns/op              64 B/op          2 allocs/op
//
// goos: windows
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: Intel(R) Xeon(R) CPU E5-2660 v4 @ 2.00GHz
// BenchmarkTextScroll/render-28                442           2506285 ns/op       48108 B/op       1002 allocs/op
// BenchmarkTextScroll/moving-28                468           2572049 ns/op       48096 B/op       1002 allocs/op
// BenchmarkTextScroll/static-28                801           1587073 ns/op          96 B/op          2 allocs/op
//
// goos: windows
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: Intel(R) Xeon(R) CPU E5-2660 v4 @ 2.00GHz
// BenchmarkTextScroll/render-28               3397            296172 ns/op           48111 B/op       1002 allocs/op
// BenchmarkTextScroll/moving-28               4251            287509 ns/op           48096 B/op       1002 allocs/op
// BenchmarkTextScroll/static-28              10000            120654 ns/op              96 B/op          2 allocs/op
//
// goos: windows
// goarch: amd64
// pkg: github.com/Konstantin8105/vl
// cpu: AMD Ryzen 7 8745HS w/ Radeon 780M Graphics
// BenchmarkTextScroll/render-16               8670            127837 ns/op           48099 B/op       1002 allocs/op
// BenchmarkTextScroll/moving-16              10000            120478 ns/op           48096 B/op       1002 allocs/op
// BenchmarkTextScroll/static-16              25698             48872 ns/op              96 B/op          2 allocs/op
func BenchmarkTextScroll(b *testing.B) {
	var screen Screen
	screen.Fill(func(rune, tcell.Style) {}) // for avoid perfomance for reset screen
	scroll := new(Scroll)
	list := new(List)
	scroll.SetRoot(list)
	for range 1000 {
		list.Add(TextStatic(texts[len(texts)-1]))
	}
	screen.SetRoot(scroll)
	var size, width uint
	size, width = 40, 40
	screen.SetHeight(size)
	up := tcell.NewEventKey(tcell.KeyPgUp, ' ', tcell.ModNone)
	down := tcell.NewEventKey(tcell.KeyPgDn, ' ', tcell.ModNone)
	// test
	b.Run("render", func(b *testing.B) {
		screen.Event(down)
		for n := 0; n < b.N; n++ {
			_ = screen.Render(width, NilDrawer)
		}
	})
	b.Run("moving", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			for range 5 {
				screen.Event(down)
			}
			_ = screen.Render(size, NilDrawer)
			for range 5 {
				screen.Event(up)
			}
		}
	})
	stList := new(Static)
	stList.SetRoot(list)
	scroll.SetRoot(stList)
	b.Run("static", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			for range 5 {
				screen.Event(down)
			}
			_ = screen.Render(size, NilDrawer)
			for range 5 {
				screen.Event(up)
			}
		}
	})
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
		for _, r := range string(content) {
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
			if ir == int('\r') {
				continue
			}
			t.Errorf("find unicode: `%s` - %d", string(r), int(r))
		}
	}
}

func list() []Widget {
	return []Widget{
		new(Separator),
		func() Widget {
			t := new(Text)
			t.SetText("Hello, World")
			return t
		}(),
		func() Widget {
			return TextStatic("Hello, World")
		}(),
		new(Scroll),
		new(List),
		new(Menu),
		new(Button),
		new(Frame),
		new(RadioGroup),
		new(CheckBox),
		new(InputBox),
		new(CollapsingHeader),
		new(ListH),
		new(ComboBox),
		new(Tabs),
		new(Tree),
		func() Widget {
			v := new(Viewer)
			v.SetText("Hello, World")
			return v
		}(),
		func() Widget {
			img := new(Image)
			img.SetImage([][]Cell{
				{
					{S: TextStyle, R: 'H'},
					{S: TextStyle, R: 'e'},
					{S: TextStyle, R: 'l'},
					{S: TextStyle, R: 'l'},
					{S: TextStyle, R: 'o'},
					{S: TextStyle, R: ','},
					{S: TextStyle, R: ' '},
					{S: TextStyle, R: 'W'},
					{S: TextStyle, R: 'o'},
					{S: TextStyle, R: 'r'},
					{S: TextStyle, R: 'l'},
					{S: TextStyle, R: 'd'},
				},
			})
			return img
		}(),
		func() Widget {
			f := new(Frame)
			f.NoBorder = true
			return f
		}(),
		func() Widget {
			c := new(CollapsingHeader)
			c.BorderIfClosed(false)
			return c
		}(),
	}
}

// func TestPanic(t *testing.T) {
// 	for _, w := range list() {
// 		t.Run(getName(w), func(t *testing.T) {
// 			w.Render(10, func(row, col uint, s tcell.Style, r rune) {
// 			})
// 		})
// 	}
// }

func getName(w Widget) string {
	name := fmt.Sprintf("%T", w)
	name = strings.ReplaceAll(name, "*vl.", "")
	if fr, ok := w.(*Frame); ok && fr.NoBorder {
		name += "NoBorder"
	}
	if c, ok := w.(*CollapsingHeader); ok {
		if c.noBorderClosed {
			name += "NoBorder"
		}
	}
	return name
}

func TestWidget(t *testing.T) {
	type tcase struct {
		name string
		w    Widget
	}
	widgets := func() (tcs []tcase) {
		for _, w := range list() {
			tcs = append(tcs, tcase{name: getName(w), w: w})
		}
		for it := range texts {
			for _, w := range list() {
				c, ok := w.(interface {
					SetText(string)
					GetText() string
				})
				if !ok {
					continue
				}
				c.SetText(texts[it])
				name := fmt.Sprintf("%s-SetText%02d", getName(w), it)
				tcs = append(tcs, tcase{name: name, w: w})
				t.Run(fmt.Sprintf("%sPrerareGetText%d", getName(w), it), func(t *testing.T) {
					if texts[it] != c.GetText() {
						t.Errorf("not same")
					}
				})
			}
			for _, w := range list() {
				c, ok := w.(interface {
					Compress()
					SetText(string)
				})
				if !ok {
					continue
				}
				c.Compress()
				c.SetText(texts[it])
				name := fmt.Sprintf("%s-CompressText%02d", getName(w), it)
				tcs = append(tcs, tcase{name: name, w: w})
			}
		}
		for _, w := range list() {
			if _, ok := w.(*RadioGroup); ok {
				continue
			}
			c, ok := w.(interface {
				Add(Widget)
			})
			if !ok {
				continue
			}
			c.Add(TextStatic("Second text"))
			name := fmt.Sprintf("%s-Add", getName(w))
			tcs = append(tcs, tcase{name: name, w: w})
			t.Run(getName(w)+"PrepareAdd", func(t *testing.T) {
				c, ok := w.(interface {
					Size() int
					Clear()
				})
				if !ok {
					t.Fatalf("Not enought function")
				}
				if c.Size() != 1 {
					t.Errorf("not valid size")
				}
			})
		}
		for _, w := range list() {
			if _, ok := w.(*RadioGroup); ok {
				continue
			}
			c, ok := w.(interface {
				Add(Widget)
			})
			if !ok {
				continue
			}
			c.Add(TextStatic("Second text"))
			var value int
			var btn Button
			btn.SetText("A) Under root")
			btn.OnClick = func() {
				value++
				btn.SetText(fmt.Sprintf("%s%d", btn.GetText(), value))
			}
			c.Add(&btn)
			name := fmt.Sprintf("%s-Add2", getName(w))
			tcs = append(tcs, tcase{name: name, w: w})
			t.Run(getName(w)+"PrepareAdd2", func(t *testing.T) {
				c, ok := w.(interface {
					Size() int
					Clear()
				})
				if !ok {
					t.Fatalf("Not enought function")
				}
				if c.Size() != 2 {
					t.Errorf("not valid size")
				}
			})
		}
		for _, w := range list() {
			c, ok := w.(interface {
				Compress()
			})
			if !ok {
				continue
			}
			c.Compress()
			name := fmt.Sprintf("%s-Compress", getName(w))
			tcs = append(tcs, tcase{name: name, w: w})
		}
		for _, w := range list() {
			c, ok := w.(interface {
				Compress()
				Add(Widget)
			})
			if !ok {
				continue
			}
			c.Compress()
			c.Add(TextStatic("Second text"))
			var value int
			var btn Button
			btn.SetText("B) Under root")
			btn.OnClick = func() {
				value++
				btn.SetText(fmt.Sprintf("%s%d", btn.GetText(), value))
			}
			c.Add(&btn)
			name := fmt.Sprintf("%s-CompressAdd2", getName(w))
			tcs = append(tcs, tcase{name: name, w: w})
		}
		for _, w := range list() {
			c, ok := w.(interface {
				SetRoot(Widget)
			})
			if !ok {
				continue
			}
			var value int
			var btn Button
			btn.SetText("C) Under root")
			btn.OnClick = func() {
				value++
				btn.SetText(fmt.Sprintf("%s%d", btn.GetText(), value))
			}
			c.SetRoot(&btn)
			name := fmt.Sprintf("%s-SetRoot", getName(w))
			tcs = append(tcs, tcase{name: name, w: w})
		}
		for _, w := range list() {
			c, ok := w.(interface {
				SetRoot(Widget)
			})
			if !ok {
				continue
			}
			var rg RadioGroup
			rg.AddText("radio0", "radio1")
			c.SetRoot(&rg)
			name := fmt.Sprintf("%s-SetRootRadiGroup", getName(w))
			tcs = append(tcs, tcase{name: name, w: w})
		}
		return
	}
	for _, size := range sizes {
		height := size
		width := size
		if 10 < height {
			height = 10
		}
		for _, tc := range widgets() {
			name := fmt.Sprintf("Widget-%02d-%s", size, tc.name)
			t.Run(name, func(t *testing.T) {
				cells := new([][]Cell)
				var screen Screen
				screen.SetRoot(tc.w)
				screen.SetHeight(height)

				// first shot
				screen.GetContents(width, cells)
				var buf bytes.Buffer
				fmt.Fprintf(&buf, "%s", Convert(*cells))

				// click on field
				for i := 0; i < 2; i++ {
					col, row, ok := findClick(cells, width, height)
					if !ok {
						col, row = 0, 1
						t.Logf("not clicked")
					}
					fmt.Fprintf(&buf, "Click%02d %d, %d\n", i, col, row)
					click := tcell.NewEventMouse(
						int(col), int(row),
						tcell.Button1, tcell.ModNone)
					screen.Event(click)
					screen.GetContents(width, cells)
					if int(row) < len(*cells) {
						if int(col) < len((*cells)[row]) {
							(*cells)[row][col].R = 'V' // click indicator
						}
					}
					fmt.Fprintf(&buf, "%s", Convert(*cells))
				}

				// click left at field

				// resize
				{
					fmt.Fprintf(&buf, "Size more\n")
					width += 4
					height += 4
					screen.SetHeight(height)
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))
				}
				{
					fmt.Fprintf(&buf, "Size less\n")
					width -= 4
					height -= 4
					screen.SetHeight(height)
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))
				}

				// testing
				if width < 4 {
					return
				}
				filename := filepath.Join(testdata, name)
				compare.Test(t, filename, buf.Bytes())
			})
		}
	}
}

func findClick(cells *[][]Cell, width, height uint) (col, row uint, found bool) {
	if len(*cells) != int(height) {
		panic(fmt.Errorf("Height %d != %d", len(*cells), height))
	}
	if height == 0 {
		return
	}
	if len((*cells)[0]) != int(width) {
		panic(fmt.Errorf("Width %d != %d", len((*cells)[0]), width))
	}
	if width == 0 {
		return
	}
	for row = range height {
		for col = range width {
			if (*cells)[row][col].S == ButtonStyle ||
				(*cells)[row][col].S == InputBoxStyle {
				found = true
				return
			}
		}
	}
	for row = range height {
		for col = range width {
			if (*cells)[row][col].S == ButtonFocusStyle ||
				(*cells)[row][col].S == InputBoxFocusStyle {
				found = true
				return
			}
		}
	}
	return
}

func TestMenuList(t *testing.T) {
	txts := [][]string{
		{},
		{"One"},
		{"One", "Two"},
		{"One", "Long long text", "Tree"},
		{"Long long text 1", "Long long text 2"},
	}
	{
		var ls []string
		for i := range 10 {
			ls = append(ls, fmt.Sprintf("Long long text %d", i))
		}
		txts = append(txts, ls)
	}

	var main Menu
	var screen Screen
	for _, col := range []uint{5, 10, 20, 25} {
		for it := range txts {
			for _, size := range sizes {
				submenu := Menu{
					parent: &main,
					offset: Offset{
						row: 2,
						col: col,
					},
				}
				submenu.Focus(true)
				for k, t := range txts[it] {
					if k%2 == 0 {
						var sub Menu
						sub.AddButton(t, nil)
						submenu.AddMenu(t, &sub)
					} else {
						submenu.AddText(t)
					}
				}
				width, height := size, size
				if 10 < height {
					height = 10
				}
				name := fmt.Sprintf("MenuList-%02d-%02d-COL%02d", it, size, col)
				t.Run(name, func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("%v\n%s", r, string(debug.Stack()))
						}
					}()
					screen.SetRoot(&submenu)
					screen.SetHeight(height)

					cells := new([][]Cell)
					var buf bytes.Buffer
					submenu.opened = true // TODO ???
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))

					// click on field
					col, row, ok := findClick(cells, width, height)
					if !ok {
						col, row = 0, 1
						t.Logf("not clicked")
					}
					for i := range 2 {
						fmt.Fprintf(&buf, "Click%02d %d, %d\n", i, col, row)
						click := tcell.NewEventMouse(
							int(col), int(row),
							tcell.Button1, tcell.ModNone)
						screen.Event(click)
						submenu.opened = true // TODO ???
						screen.GetContents(width, cells)
						fmt.Fprintf(&buf, "%s", Convert(*cells))
					}

					// testing
					if size < 4 {
						return
					}
					filename := filepath.Join(testdata, name)
					compare.Test(t, filename, buf.Bytes())
				})
			}
		}
	}
}

func TestViewer(t *testing.T) {
	var vr Viewer
	vr.SetText("Instead, they use ModAlt, even for events that could possibly have been distinguished from ModAlt.\n\nInstead, they use ModAlt, even for events that could possibly have been distinguished from ModAlt.")
	vr.SetHeight(5)
	vr.render(10)
	// view text
	t.Logf("Text lines:")
	for i := range vr.data {
		var line string
		for j := range vr.data[i] {
			line += string(vr.data[i][j].R)
		}
		t.Logf("%04d %s\n", i, line)
	}
	// view datas
	// for i := range vr.linePos {
	// 	t.Logf("%v\t", vr.linePos[i])
	// }
	// moving
	vr.NextPage()
	vr.NextPage()
	for _, step := range []func(){
		vr.NextPage, vr.PrevPage,
		vr.NextPage, vr.PrevPage,
		vr.NextPage, vr.PrevPage,
		vr.NextPage, vr.PrevPage,
		vr.NextPage, vr.PrevPage,
		vr.NextPage, vr.PrevPage,
		vr.NextPage, vr.PrevPage,
	} {
		step()
		t.Logf("Position = %d", vr.position)
	}
}

func TestListHSplitter(t *testing.T) {
	lh := new(ListH)
	lh.Add(TextStatic("1111111111"))
	lh.Add(TextStatic("2222222222"))
	lh.Add(TextStatic("3333333333"))
	var screen Screen
	screen.SetRoot(lh)
	screen.SetHeight(5)

	var buf bytes.Buffer
	cells := new([][]Cell)
	for _, f := range []func(uint, int) []int{
		nil,
		func(width uint, size int) (ws []int) {
			if size != 3 {
				return
			}
			if int(width) < 8 {
				return
			}
			return []int{3, int(width) - 3 - 3 - 2, 3}
		},
	} {
		for _, width := range []uint{4, 6, 15, 20} {
			lh.Splitter = f
			screen.GetContents(width, cells)
			fmt.Fprintf(&buf, "%s", Convert(*cells))
		}
	}
	filename := filepath.Join(testdata, "ListHSplitter")
	compare.Test(t, filename, buf.Bytes())
}

func TestTextHeight(t *testing.T) {
	btn := new(Button)
	btn.SetText("Hello,World!")

	var screen Screen
	screen.SetRoot(btn)
	screen.SetHeight(5)

	var buf bytes.Buffer
	cells := new([][]Cell)

	for _, f := range []func(){
		func() {},
		func() {
			btn.SetLinesLimit(3)
		},
	} {
		f()
		for _, width := range []uint{4, 6, 15, 20} {
			screen.GetContents(width, cells)
			fmt.Fprintf(&buf, "%s", Convert(*cells))
		}
	}

	filename := filepath.Join(testdata, "TextHeight")
	compare.Test(t, filename, buf.Bytes())
}

func TestViewerInternal(t *testing.T) {
	v := new(Viewer)
	example := `In according to https://en.wikipedia.org/wiki/Representational_systems_(NLP)
According to Bandler and Grinder our chosen words, phrases and sentences are indicative of our referencing of each of the representational systems.[4] So for example the words "black", "clear", "spiral" and "image" reference the visual representation system; similarly the words "tinkling", "silent", "squeal" and "blast" reference the auditory representation system.[4] Bandler and Grinder also propose that ostensibly metaphorical or figurative language indicates a reference to a representational system such that it is actually literal. For example, the comment "I see what you're saying" is taken to indicate a visual representation.[5]`
	var str string
	for rep := 1; rep < 10; rep++ {
		str += strings.Repeat(example, rep) + "\n"
	}
	v.SetText(str)
	v.SetColorize([]Colorize{
		TypicalColorize(
			[]string{"see", "visual", "black", "white", "image", "indicate"},
			Style(tcell.ColorWhite, tcell.ColorGreen),
		),
		TypicalColorize(
			[]string{"bandler", "i", "you", "grinder"},
			Style(tcell.ColorDeepPink, tcell.ColorYellow),
		),
		TypicalColorize(
			[]string{"silent", "saying"},
			Style(tcell.ColorBlack, tcell.ColorBlue),
		),
		TypicalColorize(
			[]string{"or", "for example", "also", "is taken to",
				"in according to", "According to", "and", "to"},
			Style(tcell.ColorBlack, tcell.ColorDeepPink),
		),
	}...)
	width := uint(20)
	_ = v.Render(width, NilDrawer)
	{
		filename := filepath.Join(testdata, "Viewer.View")
		compare.Test(t, filename, []byte(Convert(v.data)))
	}
	{
		var str string
		for row := range v.linePos {
			for col := range v.linePos[row] {
				str += fmt.Sprintf("%04d ", v.linePos[row][col])
			}
			str += "\n"
		}
		filename := filepath.Join(testdata, "Viewer.LinePos")
		compare.Test(t, filename, []byte(str))
	}
}

// TestProgressiveAdd tests sequential element addition in container widgets.
// For each container, starts empty, snapshots after each step (0..5),
// then verifies against golden files. Checked at 2 different screen widths.
func TestProgressiveAdd(t *testing.T) {
	type addStep func(name string, i uint)

	type testCase struct {
		name   string
		names  []string
		height uint
		widths []uint
		setup  func() (Widget, addStep, func())
	}

	tcs := []testCase{
		{
			name:   "List",
			names:  []string{"First", "Second", "Third", "Fourth", "Fifth"},
			height: 6,
			widths: sizes,
			setup: func() (Widget, addStep, func()) {
				var w List
				return &w, func(name string, _ uint) { w.Add(TextStatic(name)) }, w.Clear
			},
		},
		{
			name:   "ListH",
			names:  []string{"A", "BB", "CCC", "DDDD", "EEEEE"},
			height: 3,
			widths: []uint{10, 20},
			setup: func() (Widget, addStep, func()) {
				var w ListH
				w.Compress()
				return &w, func(name string, _ uint) { w.Add(TextStatic(name)) }, w.Clear
			},
		},
		{
			name:   "RadioGroup",
			names:  []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon"},
			height: 6,
			widths: []uint{10, 20},
			setup: func() (Widget, addStep, func()) {
				var w RadioGroup
				return &w, func(name string, _ uint) { w.AddText(name) }, w.Clear
			},
		},
		{
			name:   "ComboBox",
			names:  []string{"One", "Two", "Three", "Four", "Five"},
			height: 6,
			widths: []uint{10, 20},
			setup: func() (Widget, addStep, func()) {
				var w ComboBox
				return &w, func(name string, _ uint) { w.Add(name) }, w.Clear
			},
		},
		{
			name:   "Tabs",
			names:  []string{"Tab1", "Tab2", "Tab3", "Tab4", "Tab5"},
			height: 6,
			widths: []uint{15, 30},
			setup: func() (Widget, addStep, func()) {
				var w Tabs
				w.UseCombo(false)
				return &w, func(name string, _ uint) { w.Add(name, TextStatic("content")) }, w.Clear
			},
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root, add, clear := tc.setup()

			var screen Screen
			screen.SetRoot(root)
			screen.SetHeight(tc.height)

			var buf bytes.Buffer
			cells := new([][]Cell)

			for _, width := range tc.widths {
				for i := 0; i <= len(tc.names); i++ {
					fmt.Fprintf(&buf, "Step %d. Width %02d:\n", i, width)
					screen.GetContents(width, cells)
					fmt.Fprintf(&buf, "%s", Convert(*cells))
					if i < len(tc.names) {
						add(tc.names[i], uint(i))
					}
				}
				clear()
			}

			filename := filepath.Join(testdata, "ProgressiveAdd-"+tc.name)
			compare.Test(t, filename, buf.Bytes())
		})
	}
}

// capture renders a widget at every (width, height) pair and returns the output.
func capture(t *testing.T, root Widget, widths, heights []uint) []byte {
	t.Helper()
	var buf bytes.Buffer
	cells := new([][]Cell)
	for _, h := range heights {
		var screen Screen
		screen.SetRoot(root)
		screen.SetHeight(h)
		for _, w := range widths {
			fmt.Fprintf(&buf, "Height %02d. Width %02d:\n", h, w)
			screen.GetContents(w, cells)
			fmt.Fprintf(&buf, "%s", Convert(*cells))
		}
	}
	return buf.Bytes()
}

// TestFrameEdgeCases checks Frame widget with different border/header/root combos
// at multiple width and height values.
func TestFrameEdgeCases(t *testing.T) {
	type testCase struct {
		name       string
		noBorder   bool
		withHeader bool
		withRoot   bool
	}
	tcs := []testCase{
		{"NoBorder", true, false, false},
		{"WithHeader", false, true, false},
		{"WithRoot", false, false, true},
		{"FullBorder", false, true, true},
		{"FullNoBorder", true, true, true},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var fr Frame
			fr.NoBorder = tc.noBorder
			if tc.withHeader {
				var ch CheckBox
				ch.SetText("Test Header")
				fr.Header = &ch
			}
			if tc.withRoot {
				fr.SetRoot(TextStatic("Inside content"))
			}
			out := capture(t, &fr, []uint{8, 16}, []uint{3, 5})
			compare.Test(t, filepath.Join(testdata, "Frame-"+tc.name), out)
		})
	}
}

// TestCheckBoxVariants checks CheckBox with different states
// at multiple width and height values.
func TestCheckBoxVariants(t *testing.T) {
	type testCase struct {
		name     string
		checked  bool
		readonly bool
	}
	tcs := []testCase{
		{"Unchecked", false, false},
		{"Checked", true, false},
		{"ReadOnly", false, true},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var ch CheckBox
			ch.SetText("Option")
			ch.Checked = tc.checked
			ch.ReadOnly = tc.readonly
			out := capture(t, &ch, []uint{6, 12}, []uint{2, 3})
			compare.Test(t, filepath.Join(testdata, "CheckBox-"+tc.name), out)
		})
	}
}

// TestTextSettings checks Text widget with MaxLines, LinesLimit, Compress
// at multiple width and height values.
func TestTextSettings(t *testing.T) {
	type testCase struct {
		name     string
		text     string
		maxLines uint
		useLimit bool
		compress bool
	}
	tcs := []testCase{
		{"Short", "Hello", 0, false, false},
		{"LongWithMaxLines", "Line1\nLine2\nLine3\nLine4\nLine5\nLine6", 3, false, false},
		{"WithLinesLimit", "A\nB\nC\nD\nE", 0, true, false},
		{"Compressed", "Compress Me", 0, false, true},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var txt Text
			txt.SetText(tc.text)
			if 0 < tc.maxLines {
				txt.SetMaxLines(tc.maxLines)
			}
			if tc.useLimit {
				txt.SetLinesLimit(3)
			}
			if tc.compress {
				txt.Compress()
			}
			out := capture(t, &txt, []uint{5, 12}, []uint{4, 6})
			compare.Test(t, filepath.Join(testdata, "Text-"+tc.name), out)
		})
	}
}

// TestScrollWithHeight checks Scroll widget with SetHeight and scrollbar
// at multiple width and height values.
func TestScrollWithHeight(t *testing.T) {
	type testCase struct {
		name     string
		addLimit bool
		height   uint
	}
	tcs := []testCase{
		{"NoLimit", false, 0},
		{"Limit3", true, 3},
		{"Limit5", true, 5},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var sc Scroll
			var list List
			list.Add(TextStatic("Line A"))
			list.Add(TextStatic("Line B"))
			list.Add(TextStatic("Line C"))
			list.Add(TextStatic("Line D"))
			sc.SetRoot(&list)
			heights := []uint{tc.height}
			if tc.addLimit {
				sc.SetHeight(tc.height)
			} else {
				heights = []uint{4, 6}
			}
			out := capture(t, &sc, []uint{8, 14}, heights)
			compare.Test(t, filepath.Join(testdata, "Scroll-"+tc.name), out)
		})
	}
}

// TestCollapsingHeaderStates checks CollapsingHeader open/closed/border modes
// at multiple width and height values.
func TestCollapsingHeaderStates(t *testing.T) {
	type testCase struct {
		name     string
		open     bool
		noBorder bool
		withRoot bool
	}
	tcs := []testCase{
		{"ClosedWithBorder", false, false, false},
		{"ClosedNoBorder", false, true, false},
		{"OpenWithRoot", true, false, true},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var ch CollapsingHeader
			ch.SetText("Header")
			ch.Open(tc.open)
			ch.BorderIfClosed(!tc.noBorder)
			if tc.withRoot {
				ch.SetRoot(TextStatic("Inside text"))
			}
			out := capture(t, &ch, []uint{8, 16}, []uint{3, 5})
			compare.Test(t, filepath.Join(testdata, "Collapsing-"+tc.name), out)
		})
	}
}

// TestImageSizes checks Image widget with various data sizes
// at multiple width and height values.
func TestImageSizes(t *testing.T) {
	type testCase struct {
		name string
		rows int
		cols int
	}
	tcs := []testCase{
		{"1x1", 1, 1},
		{"3x5", 3, 5},
		{"5x3", 5, 3},
		{"Empty", 0, 0},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var img Image
			if 0 < tc.rows && 0 < tc.cols {
				data := make([][]Cell, tc.rows)
				for r := range data {
					data[r] = make([]Cell, tc.cols)
					for c := range data[r] {
						data[r][c] = Cell{S: TextStyle, R: 'X'}
					}
				}
				data[0][0] = Cell{S: ButtonStyle, R: 'S'}
				img.SetImage(data)
			}
			out := capture(t, &img, []uint{5, 10}, []uint{2, 6})
			compare.Test(t, filepath.Join(testdata, "Image-"+tc.name), out)
		})
	}
}

// TestMiscFunctions tests small uncovered utility functions.
func TestMiscFunctions(t *testing.T) {
	t.Run("NilDrawer", func(t *testing.T) {
		_ = NilDrawer(0, 0, TextStyle, ' ')
	})

	t.Run("FillAndSetStyle", func(t *testing.T) {
		var called bool
		var screen Screen
		screen.Fill(func(r rune, s tcell.Style) {
			called = true
		})
		screen.SetHeight(2)
		_ = screen.Render(4, NilDrawer)
		if !called {
			t.Error("Fill callback was not called")
		}

		var txt Text
		st := ButtonStyle
		txt.SetStyle(&st)
	})

	t.Run("GetLimit", func(t *testing.T) {
		var cvf ContainerVerticalFix
		cvf.SetHeight(10)
		add, hmax := cvf.GetLimit()
		if !add {
			t.Error("expected addlimit true")
		}
		if hmax != 10 {
			t.Errorf("expected hmax 10, got %d", hmax)
		}
	})

	t.Run("SpecificSymbol", func(t *testing.T) {
		SpecificSymbol(false)
		if LineHorizontalFocus != '\u2550' {
			t.Error("expected unicode horizontal focus")
		}
		SpecificSymbol(true)
		if LineHorizontalFocus != '=' {
			t.Error("expected ascii horizontal focus")
		}
	})

	t.Run("Stack", func(t *testing.T) {
		var stack Stack
		var stackSc Scroll
		stackSc.SetRoot(TextStatic("Stacked"))
		stack.Push(&stackSc)
		stack.Focus(true)
		stack.StoreSize(10, 5)
		stack.Event(tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone))
		stack.Pop()
		var stackSc2 Scroll
		stack.Push(&stackSc2)
		stack.SetHeight(3)
		_ = stack.Render(10, NilDrawer)
	})

	t.Run("ListGetUpdate", func(t *testing.T) {
		var list List
		list.Add(TextStatic("zero"))
		list.Add(TextStatic("one"))
		if g := list.Get(0); g == nil {
			t.Error("Get(0) is nil")
		}
		if g := list.Get(99); g != nil {
			t.Error("Get(99) should be nil")
		}
		var btn2 Button
		btn2.SetText("updated")
		list.Update(1, &btn2)
		list.Update(99, nil)
	})

	t.Run("ViewerPosition", func(t *testing.T) {
		var vr Viewer
		vr.SetPosition(5)
		if vr.GetPosition() != 5 {
			t.Errorf("position expected 5, got %d", vr.GetPosition())
		}
	})

	t.Run("ComboBox", func(t *testing.T) {
		var nilCB *ComboBox
		if nilCB.GetPos() != 0 {
			t.Error("nil ComboBox GetPos should return 0")
		}
		var cb2 ComboBox
		cb2.Add("a", "b", "c")
		cb2.Render(20, NilDrawer)
		var changed bool
		cb2.OnChange = func() { changed = true }
		cb2.SetPos(1)
		if !changed {
			t.Error("OnChange not called after SetPos")
		}
	})

	t.Run("TabsPosition", func(t *testing.T) {
		var tabs Tabs
		tabs.UseCombo(false)
		tabs.Add("Z", TextStatic("z"))
		tabs.Add("Y", TextStatic("y"))
		tabs.SetPos(1)
		if tabs.GetPos() != 1 {
			t.Errorf("tabs pos expected 1, got %d", tabs.GetPos())
		}
	})

	t.Run("ScrollKeyboard", func(t *testing.T) {
		var scrollEv Scroll
		scrollEv.SetRoot(TextStatic("X"))
		scrollEv.SetHeight(3)
		scrollEv.Event(tcell.NewEventKey(tcell.KeyPgDn, ' ', tcell.ModNone))
		scrollEv.Event(tcell.NewEventKey(tcell.KeyPgUp, ' ', tcell.ModNone))
		scrollEv.Focus(true)
		scrollEv.Event(tcell.NewEventKey(tcell.KeyPgDn, ' ', tcell.ModNone))
	})

	t.Run("DrawerLimit", func(t *testing.T) {
		var hit int
		dr := func(row, col uint, s tcell.Style, r rune) (isVisibleRow bool) {
			hit++
			return true
		}
		lim := DrawerLimit(dr, 0, 0, 2, 3)
		lim(0, 0, TextStyle, 'A')
		lim(5, 0, TextStyle, 'B')
		lim(0, 5, TextStyle, 'C')
		if hit != 1 {
			t.Errorf("expected 1 draw call, got %d", hit)
		}
	})
}

// TestTreeEvent checks Tree event handling without prior Render.
func TestTreeEvent(t *testing.T) {
	var tr Tree
	tr.Root = TextStatic("Root")
	tr.Nodes = []Tree{
		{Root: TextStatic("Child")},
	}

	tr.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
	tr.Focus(true)
	tr.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
}

// TestEdgePanic finds and fixes code that panics on valid edge cases.
func TestEdgePanic(t *testing.T) {
	t.Run("RadioGroupDirectListAdd", func(t *testing.T) {
		var rg RadioGroup
		rg.list.Add(TextStatic("bypass"))
		rg.Render(10, NilDrawer)
	})

	t.Run("RadioGroupEmptyList", func(t *testing.T) {
		var rg RadioGroup
		rg.Render(10, NilDrawer)
		rg.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
	})

	t.Run("ListHCompressNilWidget", func(t *testing.T) {
		var lh ListH
		lh.Compress()
		lh.Add(nil)
		lh.Render(10, NilDrawer)
	})

	t.Run("ListHRenderEmpty", func(t *testing.T) {
		var lh ListH
		lh.Render(10, NilDrawer)
	})

	t.Run("RadioGroupEventDirectList", func(t *testing.T) {
		var rg RadioGroup
		rg.list.Add(TextStatic("X"))
		rg.Focus(true)
		rg.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
	})
}

// TestEdgeBehavior covers remaining uncovered widget behavior.
func TestEdgeBehavior(t *testing.T) {
	t.Run("StaticCompressNil", func(t *testing.T) {
		var s Static
		s.Compress()
	})

	t.Run("RadioEvent", func(t *testing.T) {
		var r radio
		r.Focus(true)
		r.Event(tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone))
		r.Event(tcell.NewEventMouse(1, 0, tcell.Button1, tcell.ModNone))
		r.Focus(false)
	})

	t.Run("InputBoxKeyEvents", func(t *testing.T) {
		var in InputBox
		in.SetText("ABC")
		in.SetMaxLines(1)
		// Render once to initialize text field dimensions.
		in.Render(10, NilDrawer)
		in.Focus(true)
		in.Event(tcell.NewEventKey(tcell.KeyLeft, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(tcell.KeyRight, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(tcell.KeyUp, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(tcell.KeyDown, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(tcell.KeyBackspace, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(tcell.KeyDelete, ' ', tcell.ModNone))
		in.Event(tcell.NewEventKey(0, 'X', tcell.ModNone))
		in.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
	})

	t.Run("ViewerPageMove", func(t *testing.T) {
		var vr Viewer
		vr.SetText("Line1\nLine2\nLine3\nLine4\nLine5")
		vr.render(10)
		vr.NextPage()
		vr.PrevPage()
		// With SetHeight (addlimit).
		vr.SetHeight(2)
		vr.NextPage()
		vr.PrevPage()
	})

	t.Run("StackEmpty", func(t *testing.T) {
		var s Stack
		s.Pop()
		_ = s.Render(10, NilDrawer)
		w, h := s.GetSize()
		if w != 0 && h != 0 {
			t.Logf("Stack.GetSize = %d,%d", w, h)
		}
	})

	t.Run("ListHSetHeight", func(t *testing.T) {
		var l ListH
		l.Add(TextStatic("A"))
		l.SetHeight(5)
		// Re-render to cover SetHeight propagation path.
		_ = l.Render(10, NilDrawer)
	})

	t.Run("FixOffsetEdge", func(t *testing.T) {
		var sc Scroll
		sc.internal.offset = 5
		sc.internal.height = 1
		sc.fixOffset()
	})

	t.Run("ScrollFocusNil", func(t *testing.T) {
		var sc Scroll
		sc.Focus(true)
	})

	t.Run("ScrollOffsetEdge", func(t *testing.T) {
		var sc Scroll
		var list List
		list.Add(TextStatic("A"))
		sc.SetRoot(&list)
		sc.SetHeight(3)
		sc.internal.offset = 10
		sc.fixOffset()
	})

	t.Run("ViewerNextPageEdge", func(t *testing.T) {
		var vr Viewer
		vr.SetText("S")
		vr.render(10)
		vr.NextPage()
		vr.PrevPage()
	})

	t.Run("RadioEventBanner", func(t *testing.T) {
		var r radio
		var btn Button
		btn.SetText("In")
		r.SetRoot(&btn)
		r.Focus(true)
		r.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		r.Event(tcell.NewEventMouse(10, 0, tcell.Button1, tcell.ModNone))
		// Key event routed to root.
		r.Event(tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone))
	})

	t.Run("RunSimulation", func(t *testing.T) {
		simulation = true
		defer func() { simulation = false }()
		qu := make(chan struct{}, 1)
		action := make(chan func(), 10)
		root := Demo()[0]
		go func() {
			qu <- struct{}{}
		}()
		err := Run(root, action, qu)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})
}

// TestAddModifyRemove exercises add/update/delete cycles on all container
// widgets to verify no panic occurs during normal lifecycle operations.
func TestAddModifyRemove(t *testing.T) {
	t.Run("ListCRUD", func(t *testing.T) {
		var list List
		list.Add(TextStatic("a"))
		list.Add(TextStatic("b"))
		list.Add(nil)
		list.Add(TextStatic("c"))
		list.Render(10, NilDrawer)

		list.Update(1, TextStatic("B2"))
		list.Update(3, nil)
		list.Render(10, NilDrawer)

		if g := list.Get(0); g == nil {
			t.Error("Get(0) should not be nil")
		}
		if g := list.Get(99); g != nil {
			t.Error("Get(99) should be nil")
		}
		list.Clear()
		list.Render(10, NilDrawer)

		list.Add(TextStatic("after clear"))
		list.Render(10, NilDrawer)
	})

	t.Run("ListHCRUD", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("x"))
		lh.Add(TextStatic("y"))
		lh.Render(10, NilDrawer)

		lh.Compress()
		lh.Render(10, NilDrawer)

		lh.Clear()
		lh.Render(10, NilDrawer)

		lh.Add(TextStatic("z"))
		lh.Render(10, NilDrawer)
	})

	t.Run("RadioGroupCRUD", func(t *testing.T) {
		var rg RadioGroup
		rg.AddText("one", "two", "three")
		rg.Render(10, NilDrawer)

		rg.Clear()
		rg.Render(10, NilDrawer)

		rg.AddText("four")
		rg.SetPos(0)
		rg.Render(10, NilDrawer)

		var onchange int
		rg.OnChange = func() { onchange++ }
		rg.AddText("five")
		if onchange == 0 {
			t.Error("OnChange should fire on Add")
		}
	})

	t.Run("ComboBoxCRUD", func(t *testing.T) {
		var cb ComboBox
		cb.Add("a", "b")
		cb.Render(10, NilDrawer)

		cb.Clear()
		cb.Render(10, NilDrawer)

		cb.Add("c", "d", "e")
		cb.SetPos(2)
		cb.Render(10, NilDrawer)
	})

	t.Run("TabsCRUD", func(t *testing.T) {
		var tabs Tabs
		tabs.UseCombo(false)
		tabs.Add("X", TextStatic("contentX"))
		tabs.Render(10, NilDrawer)

		tabs.Clear()
		tabs.UseCombo(false)
		tabs.Render(10, NilDrawer)

		tabs.Add("Y", TextStatic("contentY"))
		tabs.Add("Z", TextStatic("contentZ"))
		tabs.SetPos(1)
		tabs.Render(10, NilDrawer)

		tabs.UseCombo(true)
		tabs.Render(10, NilDrawer)
	})

	t.Run("ScrollRootSwap", func(t *testing.T) {
		var sc Scroll
		sc.SetRoot(nil)
		sc.Render(10, NilDrawer)
		sc.Focus(true)
		sc.Event(tcell.NewEventKey(tcell.KeyPgDn, ' ', tcell.ModNone))

		sc.SetRoot(TextStatic("Hello"))
		sc.Render(10, NilDrawer)
		sc.Event(tcell.NewEventKey(tcell.KeyPgDn, ' ', tcell.ModNone))

		sc.SetRoot(nil)
		sc.Render(10, NilDrawer)
	})

	t.Run("MenuCRUD", func(t *testing.T) {
		var menu Menu
		menu.AddButton("Btn", nil)
		menu.AddText("Txt")
		var sub Menu
		sub.AddButton("SubBtn", nil)
		menu.AddMenu("Sub", &sub)
		menu.Render(10, NilDrawer)

		menu.SetHeight(5)
		menu.Render(10, NilDrawer)
	})

	t.Run("FrameRootSwap", func(t *testing.T) {
		var fr Frame
		fr.SetRoot(TextStatic("first"))
		fr.Render(10, NilDrawer)

		fr.SetRoot(TextStatic("second"))
		fr.Render(10, NilDrawer)

		fr.SetRoot(nil)
		fr.Render(10, NilDrawer)

		fr.NoBorder = true
		fr.SetRoot(TextStatic("no border"))
		fr.Render(10, NilDrawer)
	})

	t.Run("ButtonTextCycle", func(t *testing.T) {
		var b Button
		b.SetText("start")
		b.Compress()
		b.Render(10, NilDrawer)

		b.SetText("changed")
		b.Render(10, NilDrawer)

		b.SetMaxLines(2)
		b.SetLinesLimit(2)
		b.SetText("multi\nline\ntext")
		b.Render(10, NilDrawer)

		var clicked int
		b.OnClick = func() { clicked++ }
		b.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if clicked != 1 {
			t.Error("button click not fired")
		}
	})

	t.Run("TextSetGetCycle", func(t *testing.T) {
		var txt Text
		txt.SetText("initial")
		if txt.GetText() != "initial" {
			t.Error("GetText mismatch")
		}
		txt.SetText("update")
		if txt.GetText() != "update" {
			t.Error("GetText mismatch after update")
		}
		txt.SetLinesLimit(3)
		txt.SetMaxLines(2)
		txt.Compress()
		txt.Filter(func(r rune) bool { return r != ' ' })
		txt.Render(10, NilDrawer)
	})

	t.Run("CheckBoxToggle", func(t *testing.T) {
		var ch CheckBox
		ch.SetText("toggle")
		ch.Render(10, NilDrawer)

		ch.Checked = true
		ch.Render(10, NilDrawer)

		ch.Checked = false
		ch.ReadOnly = true
		ch.Render(10, NilDrawer)

		var changed int
		ch.OnChange = func() { changed++ }
		ch.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if ch.ReadOnly && changed != 0 {
			t.Error("ReadOnly should block change")
		}
		ch.ReadOnly = false
		ch.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if changed != 1 {
			t.Error("change should fire")
		}
	})

	t.Run("StackPushPopCycle", func(t *testing.T) {
		var s Stack
		for i := 0; i < 5; i++ {
			var sc Scroll
			sc.SetRoot(TextStatic("item"))
			s.Push(&sc)
		}
		s.Render(10, NilDrawer)
		for i := 0; i < 5; i++ {
			s.Pop()
		}
		s.Pop()
		s.Render(10, NilDrawer)
	})

	t.Run("CollapsingHeaderCycle", func(t *testing.T) {
		var ch CollapsingHeader
		ch.SetText("header")
		ch.Render(10, NilDrawer)

		ch.Open(true)
		ch.SetRoot(TextStatic("inner"))
		ch.Render(10, NilDrawer)

		ch.Open(false)
		ch.BorderIfClosed(false)
		ch.Render(10, NilDrawer)
	})
}

// TestIntToUintEdgeCases verifies that list operations using sentinel -1
// values for nil nodes do not cause uint overflow or panic.
func TestIntToUintEdgeCases(t *testing.T) {
	t.Run("ListAllNilNodes", func(t *testing.T) {
		var list List
		list.Add(nil)
		list.Add(nil)
		list.Add(nil)
		h := list.Render(10, NilDrawer)
		if h != 0 {
			t.Errorf("expected height 0 for all-nil list, got %d", h)
		}
	})

	t.Run("ListLastNodeNil", func(t *testing.T) {
		var list List
		list.Add(TextStatic("A"))
		list.Add(nil)
		h := list.Render(10, NilDrawer)
		if h == 0 {
			t.Error("expected non-zero height when first node is valid")
		}
	})

	t.Run("ListFirstNodeNil", func(t *testing.T) {
		var list List
		list.Add(nil)
		list.Add(TextStatic("B"))
		var buf bytes.Buffer
		cells := new([][]Cell)
		var screen Screen
		screen.SetRoot(&list)
		screen.SetHeight(3)
		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "%s", Convert(*cells))
		filename := filepath.Join(testdata, "ListFirstNodeNil")
		compare.Test(t, filename, buf.Bytes())
	})

	t.Run("ListHeightZeroWidget", func(t *testing.T) {
		var list List
		list.Add(new(Separator))
		list.Render(10, NilDrawer)
	})

	t.Run("ListMultipleNilInterleaved", func(t *testing.T) {
		var list List
		list.Add(TextStatic("A"))
		list.Add(nil)
		list.Add(TextStatic("B"))
		list.Add(nil)
		list.Add(TextStatic("C"))
		h := list.Render(10, NilDrawer)
		if h < 3 {
			t.Errorf("expected height >= 3 for 3 valid items, got %d", h)
		}
	})
}

// TestOnClickInit verifies that setting callback functions before any
// initialization/rendering does not lose them during widget setup.
func TestOnClickInit(t *testing.T) {
	t.Run("ButtonOnClickBeforeRender", func(t *testing.T) {
		var b Button
		b.SetText("click")
		var clicked int
		b.OnClick = func() { clicked++ }
		b.Render(10, NilDrawer)
		b.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if clicked != 1 {
			t.Errorf("OnClick lost after Render: clicked=%d", clicked)
		}
	})

	t.Run("CheckBoxOnChangeBeforeRender", func(t *testing.T) {
		var ch CheckBox
		ch.SetText("check")
		var changed int
		ch.OnChange = func() { changed++ }
		ch.Render(10, NilDrawer)
		ch.Focus(true)
		ch.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if changed == 0 {
			t.Error("OnChange lost after Render")
		}
	})

	t.Run("ComboBoxOnChangeBeforeRender", func(t *testing.T) {
		var cb ComboBox
		cb.Add("a", "b")
		var changed int
		cb.OnChange = func() { changed++ }
		cb.Render(10, NilDrawer)
		if changed == 0 {
			t.Error("ComboBox OnChange lost during init Render")
		}
	})

	t.Run("RadioGroupOnChangeBeforeRender", func(t *testing.T) {
		var rg RadioGroup
		rg.AddText("x", "y")
		var changed int
		rg.OnChange = func() { changed++ }
		rg.Render(10, NilDrawer)
		rg.Focus(true)
		rg.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
	})

	t.Run("ButtonOnClickReplaced", func(t *testing.T) {
		var b Button
		b.SetText("btn")
		var first int
		b.OnClick = func() { first++ }
		b.Render(10, NilDrawer)
		var second int
		b.OnClick = func() { second++ }
		b.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if second != 1 {
			t.Error("replaced OnClick not called")
		}
		if first != 0 {
			t.Error("old OnClick still called after replacement")
		}
	})

	t.Run("CheckBoxOnChangeDuringToggle", func(t *testing.T) {
		var ch CheckBox
		ch.SetText("opt")
		var count int
		ch.OnChange = func() { count++ }
		ch.Render(10, NilDrawer)
		ch.Focus(true)
		ch.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		ch.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		ch.Event(tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone))
		if count != 3 {
			t.Errorf("OnChange called %d times, expected 3", count)
		}
	})

	t.Run("MenuAddButtonCallback", func(t *testing.T) {
		var menu Menu
		var called int
		menu.AddButton("Test", func() { called++ })
		menu.Render(10, NilDrawer)
	})

	t.Run("CollapsingHeaderOnChange", func(t *testing.T) {
		var ch CollapsingHeader
		ch.SetText("head")
		ch.Render(10, NilDrawer)
	})

	t.Run("RadioGroupOnChangeSequence", func(t *testing.T) {
		var rg RadioGroup
		rg.AddText("a", "b", "c")
		rg.SetPos(0)
		rg.Render(10, NilDrawer)

		var changed int
		rg.OnChange = func() { changed++ }
		rg.SetPos(1)
		if changed != 1 {
			t.Errorf("SetPos should fire OnChange: changed=%d", changed)
		}
	})
}

// TestListUpdate checks that List.Update changes the rendered output
// by capturing before and after snapshots and comparing.
func TestListUpdate(t *testing.T) {
	t.Run("ReplaceText", func(t *testing.T) {
		var list List
		list.Add(TextStatic("before"))
		list.Add(TextStatic("keep"))

		var buf bytes.Buffer
		cells := new([][]Cell)

		var screen Screen
		screen.SetRoot(&list)
		screen.SetHeight(3)

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "Before update:\n%s", Convert(*cells))

		list.Update(0, TextStatic("after"))

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "After update:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListUpdateReplaceText")
		compare.Test(t, filename, buf.Bytes())
	})

	t.Run("ReplaceToNil", func(t *testing.T) {
		var list List
		list.Add(TextStatic("visible"))
		list.Add(TextStatic("removed"))

		var buf bytes.Buffer
		cells := new([][]Cell)

		var screen Screen
		screen.SetRoot(&list)
		screen.SetHeight(3)

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "Before update:\n%s", Convert(*cells))

		list.Update(1, nil)

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "After update:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListUpdateReplaceToNil")
		compare.Test(t, filename, buf.Bytes())
	})

	t.Run("NilToWidget", func(t *testing.T) {
		var list List
		list.Add(nil)
		list.Add(TextStatic("second"))

		var buf bytes.Buffer
		cells := new([][]Cell)

		var screen Screen
		screen.SetRoot(&list)
		screen.SetHeight(3)

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "Before update:\n%s", Convert(*cells))

		list.Update(0, TextStatic("first"))

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "After update:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListUpdateNilToWidget")
		compare.Test(t, filename, buf.Bytes())
	})
}

// TestListHUpdate verifies ListH.Update changes rendered output.
func TestListHUpdate(t *testing.T) {
	t.Run("ReplaceText", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("before"))
		lh.Add(TextStatic("keep"))

		var buf bytes.Buffer
		cells := new([][]Cell)
		var screen Screen
		screen.SetRoot(&lh)
		screen.SetHeight(3)

		screen.GetContents(14, cells)
		fmt.Fprintf(&buf, "Before update:\n%s", Convert(*cells))

		lh.Update(0, TextStatic("after"))

		screen.GetContents(14, cells)
		fmt.Fprintf(&buf, "After update:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListHUpdateReplaceText")
		compare.Test(t, filename, buf.Bytes())
	})

	t.Run("ReplaceToNil", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("visible"))
		lh.Add(TextStatic("removed"))

		var buf bytes.Buffer
		cells := new([][]Cell)
		var screen Screen
		screen.SetRoot(&lh)
		screen.SetHeight(3)

		screen.GetContents(14, cells)
		fmt.Fprintf(&buf, "Before update:\n%s", Convert(*cells))

		lh.Update(1, nil)

		screen.GetContents(14, cells)
		fmt.Fprintf(&buf, "After update:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListHUpdateReplaceToNil")
		compare.Test(t, filename, buf.Bytes())
	})

	t.Run("InvalidIndex", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("x"))
		lh.Update(-1, TextStatic("y"))
		lh.Update(5, TextStatic("z"))
		if lh.Size() != 1 {
			t.Errorf("size should remain 1, got %d", lh.Size())
		}
	})

	t.Run("Get", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("a"))
		lh.Add(TextStatic("b"))
		if g := lh.Get(0); g == nil {
			t.Error("Get(0) should not be nil")
		}
		if g := lh.Get(1); g == nil {
			t.Error("Get(1) should not be nil")
		}
		if g := lh.Get(-1); g != nil {
			t.Error("Get(-1) should be nil")
		}
		if g := lh.Get(5); g != nil {
			t.Error("Get(5) should be nil")
		}
	})
}

// TestDivisionByZero verifies that division operations handle zero values.
func TestDivisionByZero(t *testing.T) {
	t.Run("getItemHmaxEmpty", func(t *testing.T) {
		var l List
		h := l.getItemHmax()
		if h != 0 {
			t.Errorf("expected 0 for empty list, got %d", h)
		}
	})

	t.Run("getItemHmaxWithItems", func(t *testing.T) {
		var l List
		l.Add(TextStatic("a"))
		l.Add(TextStatic("b"))
		l.SetHeight(6)
		h := l.getItemHmax()
		if h != 3 {
			t.Errorf("expected 3 (6/2), got %d", h)
		}
	})

	t.Run("ListHRenderEmptyDefaultSplitter", func(t *testing.T) {
		var lh ListH
		h := lh.Render(10, NilDrawer)
		if h != 0 {
			t.Errorf("expected 0 for empty, got %d", h)
		}
	})

	t.Run("ScrollHmaxSmall", func(t *testing.T) {
		var sc Scroll
		sc.SetHeight(1)
		var list List
		list.Add(TextStatic("item"))
		sc.SetRoot(&list)
		h := sc.Render(10, NilDrawer)
		if h != 0 {
			t.Logf("Scroll height with hmax=1: %d", h)
		}
	})

	t.Run("ScrollEventRatioWithHmaxTwo", func(t *testing.T) {
		var sc Scroll
		sc.SetHeight(2)
		var list List
		list.Add(TextStatic("A"))
		list.Add(TextStatic("B"))
		list.Add(TextStatic("C"))
		sc.SetRoot(&list)
		sc.Focus(true)
		sc.Render(10, NilDrawer)
		sc.Event(tcell.NewEventMouse(0, 0, tcell.WheelDown, tcell.ModNone))
	})

	t.Run("ScrollHeightEqualsHmax", func(t *testing.T) {
		var sc Scroll
		sc.SetHeight(3)
		var list List
		list.Add(TextStatic("oneline"))
		sc.SetRoot(&list)
		sc.Focus(true)
		h := sc.Render(10, NilDrawer)
		if h == 0 {
			t.Errorf("expected non-zero height")
		}
	})

	t.Run("ViewerEmptyPresentRow", func(t *testing.T) {
		var vr Viewer
		if vr.presentRow() != -1 {
			t.Error("expected -1 for empty viewer linePos")
		}
	})

	t.Run("ListHRenderSingleNodeDefaultSplitter", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("X"))
		h := lh.Render(10, NilDrawer)
		if h == 0 {
			t.Errorf("expected non-zero height")
		}
	})

	t.Run("ListWithAddlimitAndItems", func(t *testing.T) {
		var l List
		l.Add(TextStatic("a"))
		l.Add(TextStatic("b"))
		l.SetHeight(4)
		h := l.Render(10, NilDrawer)
		if h != 4 {
			t.Errorf("expected height 4 (hmax), got %d", h)
		}
	})

	t.Run("ViewerNextPageEmpty", func(t *testing.T) {
		var vr Viewer
		vr.NextPage()
		vr.PrevPage()
	})

	t.Run("ViewerNextPageWithTextNoAddlimit", func(t *testing.T) {
		var vr Viewer
		vr.SetText("hello")
		vr.render(10)
		vr.NextPage()
		vr.PrevPage()
	})
}

// TestListHSplitterChanges verifies ListH.Splitter set to a function then
// back to nil produces different rendered layouts.
func TestListHSplitterChanges(t *testing.T) {
	t.Run("WithSplitterThenNil", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("AAA"))
		lh.Add(TextStatic("BB"))
		lh.Add(TextStatic("C"))

		var buf bytes.Buffer
		cells := new([][]Cell)
		var screen Screen
		screen.SetRoot(&lh)
		screen.SetHeight(3)

		lh.Splitter = func(width uint, size int) (ws []int) {
			if size != 3 || int(width) < 10 {
				return nil
			}
			return []int{5, int(width) - 5 - 3 - 2, 3}
		}

		screen.GetContents(14, cells)
		fmt.Fprintf(&buf, "With splitter:\n%s", Convert(*cells))

		lh.Splitter = nil

		screen.GetContents(14, cells)
		fmt.Fprintf(&buf, "Without splitter:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListHSplitterChanges")
		compare.Test(t, filename, buf.Bytes())
	})

	t.Run("SplitterReturnsPartial", func(t *testing.T) {
		var lh ListH
		lh.Add(TextStatic("X"))
		lh.Add(TextStatic("YY"))

		var buf bytes.Buffer
		cells := new([][]Cell)
		var screen Screen
		screen.SetRoot(&lh)
		screen.SetHeight(3)

		// Splitter returns nil (should fall back to equal-width).
		lh.Splitter = func(_ uint, _ int) []int { return nil }

		screen.GetContents(10, cells)
		fmt.Fprintf(&buf, "Splitter nil fallback:\n%s", Convert(*cells))

		filename := filepath.Join(testdata, "ListHSplitterFallback")
		compare.Test(t, filename, buf.Bytes())
	})
}

func TestSnippet(t *testing.T) {
	snippet.Test(t, ".")
}

func TestListLayoutCache(t *testing.T) {
	var l List
	for i := 0; i < 10; i++ {
		l.Add(TextStatic(fmt.Sprintf("Line %d", i)))
	}
	capture := func(width uint) (cells [][]Cell) {
		l.Render(width, func(row, col uint, s tcell.Style, r rune) (isVisibleRow bool) {
			for len(cells) <= int(row) {
				cells = append(cells, nil)
			}
			if len(cells[row]) <= int(col) {
				cells[row] = append(cells[row], make([]Cell, int(col)-len(cells[row])+1)...)
			}
			cells[row][col] = Cell{S: s, R: r}
			return true
		})
		return
	}
	for _, width := range []uint{10, 20, 40} {
		c1 := capture(width)
		c2 := capture(width)
		if len(c1) != len(c2) {
			t.Errorf("width=%d: row count differs: %d vs %d", width, len(c1), len(c2))
		}
	}
}

func TestListWidthChange(t *testing.T) {
	var l List
	for i := 0; i < 5; i++ {
		l.Add(TextStatic("Instead, they use ModAlt, even for events"))
	}
	w40 := l.Render(40, NilDrawer)
	w5 := l.Render(5, NilDrawer)
	w40b := l.Render(40, NilDrawer)
	if w40 != w40b {
		t.Errorf("second width=40 height %d differs from first %d", w40b, w40)
	}
	if w5 == w40 {
		t.Errorf("width=5 height %d should differ from width=40 height %d", w5, w40)
	}
}

func TestListNilNodes(t *testing.T) {
	var l List
	l.Add(TextStatic("First"))
	l.Add(nil)
	l.Add(TextStatic("Third"))
	l.Add(nil)
	l.Add(TextStatic("Fifth"))
	h := l.Render(20, NilDrawer)
	if l.nodes[0].from != 0 || l.nodes[0].to <= 0 {
		t.Errorf("first node: from=%d to=%d", l.nodes[0].from, l.nodes[0].to)
	}
	if l.nodes[1].from != l.nodes[0].to || l.nodes[1].to != l.nodes[1].from {
		t.Errorf("nil node 1: from=%d to=%d", l.nodes[1].from, l.nodes[1].to)
	}
	if l.nodes[2].from != l.nodes[0].to {
		t.Errorf("third node: from=%d, expected %d", l.nodes[2].from, l.nodes[0].to)
	}
	if l.nodes[4].from != l.nodes[2].to {
		t.Errorf("fifth node: from=%d, expected %d", l.nodes[4].from, l.nodes[2].to)
	}
	if h < 3 {
		t.Errorf("height too small: %d", h)
	}
	var drawn int
	l.Render(20, func(row, col uint, s tcell.Style, r rune) (isVisibleRow bool) {
		drawn++
		return true
	})
	if drawn == 0 {
		t.Errorf("no content drawn at all")
	}
}

func TestListCompressAddlimit(t *testing.T) {
	var l List
	l.Compress()
	l.SetHeight(4)
	for i := 0; i < 3; i++ {
		l.Add(TextStatic(fmt.Sprintf("Text %d", i)))
	}
	l.Render(10, NilDrawer)
	if !l.addlimit || l.hmax != 4 {
		t.Errorf("addlimit=%v hmax=%d", l.addlimit, l.hmax)
	}
	for i := range l.nodes {
		if l.nodes[i].w == nil {
			continue
		}
		dh := int(l.getItemHmax())
		if l.nodes[i].to-l.nodes[i].from > dh {
			t.Errorf("item %d: height %d exceeds dh=%d",
				i, l.nodes[i].to-l.nodes[i].from, dh)
		}
	}
}

func TestListPreallocatedDrawer(t *testing.T) {
	var l List
	l.Add(TextStatic("A"))
	l.Add(TextStatic("B"))
	l.Add(TextStatic("C"))
	rows := make(map[uint][]rune)
	l.Render(10, func(row, col uint, s tcell.Style, r rune) (isVisibleRow bool) {
		rows[row] = append(rows[row], r)
		return true
	})
	if len(rows) < 3 {
		t.Errorf("expected at least 3 rows of output, got %d", len(rows))
	}
	for row := uint(0); row < uint(len(rows)); row++ {
		if len(rows[row]) == 0 {
			t.Errorf("row %d has no content", row)
		}
	}
}

func TestListGet(t *testing.T) {
	var l List
	l.Add(TextStatic("zero"))
	l.Add(TextStatic("one"))
	if l.Get(-1) != nil {
		t.Errorf("expected nil for negative index")
	}
	if l.Get(2) != nil {
		t.Errorf("expected nil for out-of-range index")
	}
	if l.Get(0) == nil {
		t.Errorf("expected non-nil for valid index")
	}
	if l.Get(1) == nil {
		t.Errorf("expected non-nil for valid index")
	}
}

func TestListClear(t *testing.T) {
	var l List
	l.Add(TextStatic("a"))
	l.Add(TextStatic("b"))
	if l.Size() != 2 {
		t.Errorf("expected size 2, got %d", l.Size())
	}
	l.Clear()
	if l.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", l.Size())
	}
}

func TestListSetHeightNonVerticalFix(t *testing.T) {
	var l List
	var txt Text
	txt.SetText("plain")
	l.Add(&txt)
	l.SetHeight(10)
}

func TestListRenderEdgeCases(t *testing.T) {
	var empty List
	h := empty.Render(10, NilDrawer)
	if h != 0 {
		t.Errorf("empty list height should be 0, got %d", h)
	}
	var l List
	l.Add(TextStatic("hello"))
	h = l.Render(1, NilDrawer)
	if h != 0 {
		t.Errorf("width<2 height should be 0, got %d", h)
	}
	h = l.Render(0, NilDrawer)
	if h != 0 {
		t.Errorf("width=0 height should be 0, got %d", h)
	}
}

func TestListEventMouseOutOfBounds(t *testing.T) {
	var l List
	l.Add(TextStatic("item"))
	l.Focus(true)
	l.Event(tcell.NewEventMouse(-1, 0, tcell.Button1, tcell.ModNone))
	l.Event(tcell.NewEventMouse(0, -1, tcell.Button1, tcell.ModNone))
}

func TestListEventKey(t *testing.T) {
	var l List
	l.Add(TextStatic("item"))
	l.Focus(true)
	l.Event(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
}

func TestListRenderCompressNoAddlimit(t *testing.T) {
	var l List
	l.Compress()
	l.Add(TextStatic("short"))
	l.Add(TextStatic("longer text"))
	h := l.Render(10, NilDrawer)
	if h == 0 {
		t.Errorf("expected non-zero height")
	}
}

func TestScrollNoAddlimit(t *testing.T) {
	var sc Scroll
	sc.SetRoot(TextStatic("hello"))
	h := sc.Render(10, NilDrawer)
	if h == 0 {
		t.Errorf("expected non-zero height")
	}
}

func TestScrollRenderSmallWidth(t *testing.T) {
	var sc Scroll
	sc.SetRoot(TextStatic("hello"))
	h := sc.Render(1, NilDrawer)
	if h != 0 {
		t.Errorf("expected 0, got %d", h)
	}
}

func TestStaticCompressNilRoot(t *testing.T) {
	var s Static
	s.Compress()
}

func TestFrameSetHeightNilRoot(t *testing.T) {
	var f Frame
	f.SetHeight(10)
	h := f.Render(10, NilDrawer)
	if h == 0 {
		t.Errorf("expected non-zero height")
	}
}

func TestListEventNoFocus(t *testing.T) {
	var l List
	l.Add(TextStatic("item"))
	l.Event(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
}

func TestListRenderAddlimitWithoutCompress(t *testing.T) {
	var l List
	l.SetHeight(6)
	l.Add(TextStatic("a"))
	l.Add(TextStatic("b"))
	l.Add(TextStatic("c"))
	dh := l.getItemHmax()
	if dh != 2 {
		t.Errorf("dh should be 2, got %d", dh)
	}
	h := l.Render(10, NilDrawer)
	if h != l.hmax {
		t.Errorf("height %d should equal hmax %d", h, l.hmax)
	}
	h2 := l.Render(5, NilDrawer)
	if h2 != l.hmax {
		t.Errorf("height %d should equal hmax %d after width change", h2, l.hmax)
	}
}

func TestContainerVerticalFixSetHeight(t *testing.T) {
	var cvf ContainerVerticalFix
	cvf.SetHeight(10)
	addlimit, hmax := cvf.GetLimit()
	if !addlimit || hmax != 10 {
		t.Errorf("expected addlimit=true, hmax=10, got %v, %d", addlimit, hmax)
	}
	cvf.SetHeight(0)
	addlimit, hmax = cvf.GetLimit()
	if !addlimit || hmax != 0 {
		t.Errorf("expected addlimit=true, hmax=0, got %v, %d", addlimit, hmax)
	}
}
