module github.com/Konstantin8105/vl

go 1.22.3

toolchain go1.22.5

require (
	github.com/Konstantin8105/compare v0.0.0-20240706101316-2b8aefbb57c9
	github.com/Konstantin8105/snippet v0.0.0-20240712185128-0b654b2df8c7
	github.com/Konstantin8105/tf v0.0.0-20231007135105-ef617777c299
	github.com/gdamore/tcell/v2 v2.6.0
)

// replace github.com/Konstantin8105/tf => ../tf
// replace github.com/Konstantin8105/snippet => ../snippet

require (
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/olegfedoseev/image-diff v0.0.0-20171116094004-897a4e73dfd6 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/term v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
