module fifv/playground

go 1.25.6

require github.com/jchv/go-webview2 v0.0.0-20250406165304-0bcfea011047

require github.com/creack/goselect v0.1.2 // indirect

require (
	github.com/gonutz/w32 v1.0.0 // indirect
	github.com/gonutz/w32/v2 v2.12.1
	github.com/gonutz/w32/v3 v3.0.0-beta9
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/wailsapp/go-webview2 v1.0.23
	go.bug.st/serial v1.6.4
	golang.org/x/sys v0.27.0 // indirect
)

replace github.com/jchv/go-webview2 => ../go-webview2
