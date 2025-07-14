module github.com/Konstantin8105/vl

go 1.23.0

toolchain go1.24.5

require (
	github.com/Konstantin8105/compare v0.0.0-20240706101316-2b8aefbb57c9
	github.com/Konstantin8105/snippet v0.0.0-20240712185128-0b654b2df8c7
	github.com/Konstantin8105/tf v0.0.0-20240403170626-5245216e7740
	github.com/gdamore/tcell/v2 v2.8.1
)

// replace github.com/Konstantin8105/tf => ../tf
// replace github.com/Konstantin8105/snippet => ../snippet

require (
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/olegfedoseev/image-diff v0.0.0-20171116094004-897a4e73dfd6 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/term v0.33.0 // indirect
	golang.org/x/text v0.27.0 // indirect
)
