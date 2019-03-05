package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// WaitForOpen decides how long we wait for chrome to open
var WaitForOpen = 5 * time.Second

var remoteAddr string

// Tab command channel
// A single tab can only handle one command at a time
type Tab struct {
	send   chan []byte
	recv   chan []byte
	closed chan struct{}
}

// TabParams args
type TabParams map[string]interface{}

// get a response from chrome
func get(ctx context.Context, path string) (res *http.Response, err error) {
	u := url.URL{Scheme: "http", Host: remoteAddr, Path: path}
	res, err = Fetch(ctx, u.String())
	if err != nil {
		err = fmt.Errorf("chrome.get: %s", err)
		return
	}

	return
}

// DumpProtocol for debug
func DumpProtocol() {
	res, err := get(context.TODO(), "/json/protocol")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	io.Copy(os.Stdout, res.Body)
}

// ResJSON is a response
type ResJSON struct {
	Description          string `json:"description"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// GetJSON /json
func GetJSON(ctx context.Context) (response []ResJSON, err error) {
	res, err := get(ctx, "/json")
	if err != nil {
		err = fmt.Errorf("chrome.GetJSON: %s", err)
		return
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		err = fmt.Errorf("chrome.GetJSON: %s", err)
		return
	}

	return
}

// ResNewTab is a response
type ResNewTab struct {
	Description          string `json:"description"`
	DevtoolsFrontendURL  string `json:"devtoolsFrontendUrl"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// NewTab opens a new tab
func NewTab(ctx context.Context) (response ResNewTab, err error) {
	res, err := get(ctx, "/json/new")
	if err != nil {
		err = fmt.Errorf("chrome.NewTab: %s", err)
		return
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		err = fmt.Errorf("chrome.NewTab: %s", err)
		return
	}

	return
}

// ConnectToNewTab opens a new tab and connects to it
func ConnectToNewTab(ctx context.Context) (tab Tab, err error) {
	// open new tab
	res, err := NewTab(ctx)
	if err != nil {
		err = fmt.Errorf("chrome.ConnectToNewTab: %s", err)
		return
	}
	return ConnectToTab(ctx, res.WebSocketDebuggerURL)
}

// ResChrome is a response
type ResChrome struct {
	// call response
	Result json.RawMessage
	// event
	Method string `json:"method"`
	Params json.RawMessage
}

// EvPageLifecycle is an event
type EvPageLifecycle struct {
	Name      string  `json:"name"`
	Timestamp float64 `json:"timestamp"`
	FrameID   string  `json:"frameId"`
	LoaderID  string  `json:"loaderId"`
}

// EvInspectorDetached is an event
// when Page.close causes a target to close we get this
// so we treat it like an event
type EvInspectorDetached struct {
	Params struct {
		Reason string `json:"reason"`
	} `json:"params"`
}

// ConnectToTab remote control using websocketDebuggerUrl
func ConnectToTab(ctx context.Context, wsURL string) (tab Tab, err error) {
	if wsURL == "" {
		// connect to first open tab
		var res []ResJSON
		res, err = GetJSON(ctx)
		if err != nil {
			err = fmt.Errorf("chrome.ConnectToTab: %s", err)
			return
		}
		if len(res) == 0 {
			// is this possible?
			err = fmt.Errorf("chrome.ConnectToTab: no open tabs")
			return
		}

		wsURL = res[0].WebSocketDebuggerURL
	}

	remote, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		err = fmt.Errorf("chrome.ConnectToTab: %s", err)
		return
	}

	tab = Tab{
		send:   make(chan []byte),
		recv:   make(chan []byte),
		closed: make(chan struct{}),
	}

	// read
	// handle events
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := remote.ReadMessage()
			// Log("chrome.ConnectToTab: got: %s", data)
			if err != nil {
				Log("chrome.ConnectToTab: closed: %s", err)
				return
			}
			var msg ResChrome
			err = json.Unmarshal(data, &msg)
			if err != nil {
				Log("chrome.ConnectToTab: json: %s", err)
				return
			}
			switch msg.Method {
			case "Page.lifecycleEvent":
				var ev EvPageLifecycle
				err := json.Unmarshal(msg.Params, &ev)
				if err != nil {
					Log("chrome.ConnectToTab: Page.lifecycleEvent: %s", err)
					return
				}
				Log("chrome.ConnectToTab: Page.lifecycleEvent: ev: %+v", ev)
			case "Inspector.detached":
				var ev EvInspectorDetached
				err := json.Unmarshal(msg.Params, &ev)
				if err != nil {
					Log("chrome.ConnectToTab: Inspector.detached: %s", err)
					return
				}
				Log("chrome.ConnectToTab: Inspector.detached: ev: %+v", ev)
				close(tab.closed)
			default:
				select {
				case tab.recv <- msg.Result:
					Log("chrome.ConnectToTab: result: %+v", msg.Result)
				case <-time.After(500 * time.Millisecond):
					Log("chrome.ConnectToTab: result: timeout")
				}
			}
		}
	}()

	// handle writing/closing
	go func() {
		defer func() {
			remote.Close()
			wg.Done()
		}()
		for {
			select {
			case <-tab.closed:
				Log("chrome.ConnectToTab: tab was closed")
				return
			case <-done:
				return
			case <-ctx.Done():
				return
			case msg := <-tab.send:
				Log("chrome.ConnectToTab: send: %s", msg)
				err := remote.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	return
}

// WaitForLoad waits a few seconds for page to load
// todo: if we start missing blogs we need to find a better way
func (t Tab) WaitForLoad() {
	<-time.After(5 * time.Second)
}
