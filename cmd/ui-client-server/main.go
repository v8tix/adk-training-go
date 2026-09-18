// Command ui-client-server serves static/index.html — a hand-written
// vanilla JavaScript chat client for the UI Agent (cmd/ui-agent) — over
// plain HTTP. This is the Go-only replacement for Python's own lab step
// (`python3 -m http.server`), keeping this repo's "no Python required"
// scope intact: any static file server would do, but this repo provides
// its own rather than reaching outside Go tooling for one line of shell.
package main

import (
	"log"
	"net/http"
	"path/filepath"
	"runtime"
)

const addr = ":9094"

// thisFileDir is this source file's own directory, resolved once via
// runtime.Caller — cwd-independent, unlike a bare "static" relative path.
// A relative path resolves against the *process's* working directory,
// which for `go run ./cmd/ui-client-server` from the repo root would look
// for "static" under the repo root, not next to this file — the same class
// of cwd-relative bug module-27's own review caught in a different cmd/
// program.
var thisFileDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}()

func main() {
	staticDir := filepath.Join(thisFileDir, "static")
	log.Printf("💻 ui-client-server serving %s on http://localhost%s\n", staticDir, addr)
	if err := http.ListenAndServe(addr, http.FileServer(http.Dir(staticDir))); err != nil {
		log.Fatalf("run failed: %v", err)
	}
}
