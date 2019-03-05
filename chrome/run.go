package chrome

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var rw sync.RWMutex
var wg sync.WaitGroup

var cmd *exec.Cmd

// Log function
var Log = func(string, ...interface{}) {}

// Wait for chrome to close
func Wait() {
	wg.Wait()
}

// Start finds chrome and runs it
func Start(ctx context.Context, userProfileDir string, port int) (err error) {
	var app string
	if userProfileDir[:2] == "~/" {
		var home string
		home, err = os.UserHomeDir()
		if err != nil {
			err = fmt.Errorf("chrome.Start: %s", err)
			return
		}
		userProfileDir = filepath.Join(home, userProfileDir[2:])
	}
	switch runtime.GOOS {
	case "darwin":
		path := "/Applications/Google Chrome.app"
		if s, err := os.Stat(path); err == nil && s.IsDir() {
			app = fmt.Sprintf("open %s --args", path)
		}
	case "linux":
		names := []string{
			"chromium-browser",
			"chromium",
			"google-chrome",
		}
		for _, name := range names {
			if _, err := exec.LookPath(name); err == nil {
				app = name
				break
			}
		}
	case "windows":
		// todo: find chrome on windows
	}

	opts := []string{
		"--headless",
		"--disable-gpu", // for Windows
		fmt.Sprintf("--user-data-dir=%s", userProfileDir),
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"about:blank",
	}

	if app == "" {
		err = fmt.Errorf("chrome.Run: Could not find chrome")
		return
	}
	cmd = exec.CommandContext(ctx, app, opts...)

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("chrome.Run: %s", err)
		return
	}
	Log("chrome.Run: %s (%d)", cmd.Path, cmd.Process.Pid)

	// monitor process
	wg.Add(1)
	exit := make(chan struct{}, 1)
	go func() {
		cmd.Wait()
		close(exit)
	}()

	// handle exit
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			Log("chrome.Run: cancel: %s", ctx.Err())
			return
		case <-exit:
			Log("chrome.Run: exited")
			return
		}
	}()

	// connect to running chrome process
	connect(ctx, fmt.Sprintf("localhost:%d", port))

	return
}

func connect(ctx context.Context, addr string) (err error) {
	remoteAddr = addr
	u := url.URL{Scheme: "http", Host: remoteAddr, Path: "/"}

	// wait for connection
	Log("chrome.connect: wait for connection...")
	timeout := time.After(WaitForOpen)
	for {
		select {
		case err := <-ctx.Done():
			return fmt.Errorf("chrome.connect: cancel: %s", err)
		case <-timeout:
			Log("chrome.connect: timeout")
			return errors.New("chrome.connect: timeout")
		default:
			res, err := Fetch(ctx, u.String())
			if err == nil {
				res.Body.Close()
				goto connected
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
connected:
	Log("chrome.connect: connected: %s", remoteAddr)

	return
}
