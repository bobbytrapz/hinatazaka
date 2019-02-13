package chrome

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/bobbytrapz/hinatazaka/fetch"
	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/bobbytrapz/homedir"
)

var rw sync.RWMutex
var wg sync.WaitGroup

var chrome *exec.Cmd

// Log function
var Log = func(string, ...interface{}) {}

// Wait for chrome to close
func Wait() {
	wg.Wait()
}

// Start finds chrome and runs it
func Start(ctx context.Context) (err error) {
	var app string
	var userProfileDir string
	switch runtime.GOOS {
	case "darwin":
		path := "/Applications/Google Chrome.app"
		if s, err := os.Stat(path); err == nil && s.IsDir() {
			app = fmt.Sprintf("open %s --args", path)
		}
		userProfileDir, err = homedir.Expand("~/.config/hinatazaka/hinatazaka-profile")
		if err != nil {
			err = fmt.Errorf("chrome.Start: %s", err)
		}
	case "linux":
		names := []string{
			"chromium-browser",
			"chromium",
			"google-chrome",
		}
		userProfileDir, err = homedir.Expand("~/.config/hinatazaka/hinatazaka-profile")
		if err != nil {
			err = fmt.Errorf("chrome.Start: %s", err)
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

	port := options.GetInt("chrome_port")
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
	chrome = exec.CommandContext(ctx, app, opts...)

	wg.Add(1)
	if err = chrome.Start(); err != nil {
		err = fmt.Errorf("chrome.Run: %s", err)
		return
	}
	Log("chrome.Run: %s (%d)", chrome.Path, chrome.Process.Pid)

	// monitor process
	exit := make(chan error)
	go func() {
		err := chrome.Wait()
		exit <- err
	}()

	// handle exit
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			Log("chrome.Run: %s", ctx.Err())
			return
		case err := <-exit:
			Log("chrome.Run: exited: %s", err)
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
			res, err := fetch.Get(ctx, u.String())
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