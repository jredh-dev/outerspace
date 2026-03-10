// watcher-ctl — CLI client for the watcher HTTP control server.
//
// Usage:
//
//	watcher-ctl [--host HOST] [--port PORT] <command>
//
// Commands:
//
//	trigger    Run a full diff of all visible files and send to OpenCode
//	pause      Pause the watcher (stop sending diffs)
//	resume     Resume the watcher
//	interrupt  Immediately flush pending files without waiting for quiet period
//	status     Show current watcher status (JSON)
//
// Flags (also readable from env):
//
//	--host   WATCHER_CTL_HOST  default: localhost
//	--port   WATCHER_CTL_PORT  default: 4097
//
// Examples:
//
//	watcher-ctl trigger
//	watcher-ctl pause
//	watcher-ctl resume
//	watcher-ctl interrupt
//	watcher-ctl status
//	WATCHER_CTL_PORT=9000 watcher-ctl status
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("watcher-ctl", flag.ContinueOnError)
	host := fs.String("host", envOr("WATCHER_CTL_HOST", "localhost"), "watcher control server host")
	port := fs.String("port", envOr("WATCHER_CTL_PORT", "4097"), "watcher control server port")

	// Help flags — check before parsing so the FlagSet doesn't intercept them.
	for _, a := range args {
		if a == "--help" || a == "-h" || a == "-?" || a == "help" {
			usage(fs)
			return 0
		}
	}

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 2
	}

	if fs.NArg() == 0 {
		usage(fs)
		return 2
	}

	cmd := fs.Arg(0)
	baseURL := fmt.Sprintf("http://%s:%s", *host, *port)

	switch cmd {
	case "trigger":
		return postCmd(baseURL+"/trigger", "trigger")
	case "pause":
		return postCmd(baseURL+"/pause", "pause")
	case "resume":
		return postCmd(baseURL+"/resume", "resume")
	case "interrupt":
		return postCmd(baseURL+"/interrupt", "interrupt")
	case "status":
		return getStatus(baseURL + "/status")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", cmd)
		usage(fs)
		return 2
	}
}

// postCmd sends a POST to the given URL and prints the response.
func postCmd(url, name string) int {
	resp, err := http.Post(url, "application/json", nil) //nolint:noctx
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s: %v\n", name, err)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "error: %s returned %d: %s\n", name, resp.StatusCode, strings.TrimSpace(string(body)))
		return 1
	}

	// Pretty-print JSON if possible, otherwise raw.
	var m map[string]any
	if err := json.Unmarshal(body, &m); err == nil {
		fmt.Printf("%s: %s\n", name, m["message"])
	} else {
		fmt.Printf("%s: %s\n", name, strings.TrimSpace(string(body)))
	}
	return 0
}

// getStatus fetches GET /status and pretty-prints the result.
func getStatus(url string) int {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: status: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "error: status returned %d: %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
		return 1
	}

	// Decode and pretty-print.
	var s struct {
		Paused        bool   `json:"paused"`
		PendingFiles  int    `json:"pending_files"`
		LastBatchTime string `json:"last_batch_time"`
	}
	if err := json.Unmarshal(body, &s); err != nil {
		// Fall back to raw JSON.
		var raw any
		_ = json.Unmarshal(body, &raw)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(raw)
		return 0
	}

	paused := "no"
	if s.Paused {
		paused = "YES"
	}
	last := s.LastBatchTime
	if last == "" {
		last = "(none)"
	}
	fmt.Printf("paused:        %s\n", paused)
	fmt.Printf("pending files: %d\n", s.PendingFiles)
	fmt.Printf("last batch:    %s\n", last)
	return 0
}

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, `watcher-ctl — control the watcher service

Usage:
  watcher-ctl [flags] <command>

Commands:
  trigger    Run a full diff and send to OpenCode
  pause      Pause the watcher (hold diffs)
  resume     Resume the watcher
  interrupt  Flush pending files immediately
  status     Show watcher status

Flags:
`)
	fs.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Environment:
  WATCHER_CTL_HOST  default: localhost
  WATCHER_CTL_PORT  default: 4097
`)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
