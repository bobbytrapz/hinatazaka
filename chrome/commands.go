package chrome

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

var nextReqID = -1

func addReq() int {
	rw.Lock()
	defer rw.Unlock()
	nextReqID++
	return nextReqID
}

// Command builds a command and sends it
// { "id": 0, "method": "Page.navigate", params: {"url": "..."} }
func (t Tab) Command(method string, params TabParams) {
	// build command
	var buf bytes.Buffer
	id := addReq()
	fmt.Fprintf(&buf, `{"id":%d,"method":"%s"`, id, method)
	// fill in params
	comma := false
	fmt.Fprintf(&buf, `,"params":{`)
	for k, v := range params {
		if comma {
			fmt.Fprintf(&buf, ",")
		} else {
			comma = true
		}
		switch v := v.(type) {
		case bool:
			fmt.Fprintf(&buf, `"%s":%v`, k, v)
		case string:
			fmt.Fprintf(&buf, `"%s":%q`, k, v)
		case float32:
			fmt.Fprintf(&buf, `"%s":%f`, k, v)
		case float64:
			fmt.Fprintf(&buf, `"%s":%f`, k, v)
		case int:
			fmt.Fprintf(&buf, `"%s":%d`, k, v)
		case int32:
			fmt.Fprintf(&buf, `"%s":%d`, k, v)
		case int64:
			fmt.Fprintf(&buf, `"%s":%d`, k, v)
		case []byte:
			fmt.Fprintf(&buf, `"%s":%q`, k, v)
		case []rune:
			fmt.Fprintf(&buf, `"%s":%q`, k, v)
		default:
			panic("chrome.Command: unsupported type")
		}
	}
	fmt.Fprintf(&buf, `}`)

	fmt.Fprintf(&buf, `}`)
	t.send <- buf.Bytes()
}

// Wait for a response
func (t Tab) Wait() []byte {
	select {
	case v := <-t.recv:
		return v
	case <-time.After(5 * time.Second):
		Log("tab.Wait: timeout")
		return nil
	}
}

// ResPageSetLifecycleEventsEnabled is a response
type ResPageSetLifecycleEventsEnabled struct {
	FrameID  string `json:"frameId"`
	LoaderID string `json:"loaderId"`
}

// PageSetLifecycleEventsEnabled sends method Page.setLifecycleEventsEnabled
func (t Tab) PageSetLifecycleEventsEnabled(shouldEnable bool) {
	t.Command("Page.setLifecycleEventsEnabled", TabParams{
		"enabled": shouldEnable,
	})
	// wait for response
	data := <-t.recv
	var res ResPageSetLifecycleEventsEnabled
	err := json.Unmarshal(data, &res)
	if err != nil {
		Log("chrome.Tab.PageSetLifecycleEventsEnabled: %s", err)
		return
	}
	Log("chrome.Tab.PageSetLifecycleEventsEnabled: res: %+v", res)
}

// ResPageNavigate is a response
type ResPageNavigate struct {
	FrameID  string `json:"frameId"`
	LoaderID string `json:"loaderId"`
}

// PageNavigate sends method Page.navigate
func (t Tab) PageNavigate(url string) {
	t.Command("Page.navigate", TabParams{
		"url": url,
	})
	// wait for response
	data := <-t.recv
	var res ResPageNavigate
	err := json.Unmarshal(data, &res)
	if err != nil {
		Log("chrome.Tab.PageNavigate: %s", err)
		return
	}
	Log("chrome.Tab.PageNavigate: res: %+v", res)
}

// PageClose sends method Page.close (experimental)
func (t Tab) PageClose() {
	t.Command("Page.close", TabParams{})
}

// ResPageCaptureScreenshot is a response
type ResPageCaptureScreenshot struct {
	Data string `json:"data"`
}

// PageCaptureScreenshot sends method Page.captureScreenshot
func (t Tab) PageCaptureScreenshot(saveAs string) {
	// use defaults "png"
	t.Command("Page.captureScreenshot", TabParams{})
	// wait for response
	data := <-t.recv
	var res ResPageCaptureScreenshot
	err := json.Unmarshal(data, &res)
	if err != nil {
		panic(err)
	}
	Log("chrome.Tab.PageCaptureScreenshot: res: %+v", res)

	// decode
	img, err := base64.StdEncoding.DecodeString(res.Data)
	if err != nil {
		Log("chrome.Tab.PageCaptureScreenshot: %s", err)
	}

	// save
	if err := ioutil.WriteFile(saveAs, img, 0644); err != nil {
		Log("chrome.Tab.PageCaptureScreenshot: %s", err)
	}
}

// ResPagePrintToPDF is a response
type ResPagePrintToPDF struct {
	Data string `json:"data"`
}

// PagePrintToPDF sends method Page.printToPDF
func (t Tab) PagePrintToPDF(saveAs string) {
	// use defaults "png"
	t.Command("Page.printToPDF", TabParams{
		"displayHeaderFooter": true,
		"headerTemplate":      `<span class=url></span>`,
		"printBackground":     true,
		"marginTop":           0,
		"marginBottom":        0,
		"marginLeft":          0,
		"marginRight":         0,
	})
	// wait for response
	data := <-t.recv
	var res ResPagePrintToPDF
	err := json.Unmarshal(data, &res)
	if err != nil {
		Log("chrome.Tab.PagePrintToPDF: %s", err)
		fmt.Println("[nok]", err)
		return
	}
	// Log("chrome.Tab.PagePrintToPDF: res: %+v", res)

	// decode
	doc, err := base64.StdEncoding.DecodeString(res.Data)
	if err != nil {
		Log("chrome.Tab.PagePrintToPDF: %s", err)
		fmt.Println("[nok]", err)
	}

	// save
	fmt.Println("[save]", saveAs)
	if err := ioutil.WriteFile(saveAs, doc, 0644); err != nil {
		Log("chrome.Tab.PagePrintToPDF: %s", err)
		fmt.Println("[nok]", err)
	}
}
