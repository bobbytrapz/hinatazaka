package scrape

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobbytrapz/chrome"
	"github.com/bobbytrapz/hinatazaka/options"
)

var httpClient = http.Client{
	Timeout: 60 * time.Second,
}

// NumTabWorkersPerMember decides how many tabs to open for each blog spider
var NumTabWorkersPerMember = 8

// SaveTo decides where we save everything that we find
var SaveTo = options.Get("save_to")

// ResJSString is a response
type ResJSString struct {
	Result struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"result"`
}

// ResJSInt is a response
type ResJSInt struct {
	Result struct {
		Type  string `json:"type"`
		Value int    `json:"value"`
	} `json:"result"`
}

// ResJSFloat32 is a response
type ResJSFloat32 struct {
	Result struct {
		Type  string  `json:"type"`
		Value float32 `json:"value"`
	} `json:"result"`
}

// SaveImagesFrom a webpage
func SaveImagesFrom(ctx context.Context, link string, saveImagesTo string, jsCode string) error {
	tab, err := chrome.ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("scrape.SaveImagesFrom: %s", err)
	}
	defer tab.PageClose()
	SaveImagesFromTabWith(ctx, tab, link, saveImagesTo, jsCode)
	return nil
}

// SaveImagesFromTabWith sends given javascript to get urls for images
// then it downloads each image from a list of comma-separated urls
func SaveImagesFromTabWith(ctx context.Context, tab chrome.Tab, link string, saveTo string, jsCode string) {
	tab.PageNavigate(link)
	tab.WaitForLoad(10 * time.Second)

	tab.Command("Runtime.evaluate", chrome.TabParams{
		"expression": jsCode,
	})
	// wait for reponse
	data := tab.Wait()
	var res ResJSString
	err := json.Unmarshal(data, &res)
	if err != nil {
		chrome.Log("scrape.SaveImagesFromTabWith: %s", err)
	}
	chrome.Log("scrape.SaveImagesFromTabWith: res: %+v", res)

	count := 0
	for _, u := range strings.Split(res.Result.Value, ",") {
		if u == "" {
			continue
		}

		purl, err := url.Parse(u)
		if err != nil {
			chrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
			continue
		}
		fn := filepath.Join(saveTo, filepath.Base(purl.Path))

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			err = fmt.Errorf("scrape.SaveImagesFromTabWith: %s", err)
			return
		}
		req = req.WithContext(ctx)

		res, err := httpClient.Do(req)
		if err != nil {
			chrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			chrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		fmt.Println("[save]", fn)
		_, err = io.Copy(f, res.Body)
		if err != nil {
			chrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
		}
		count++

		f.Close()
		res.Body.Close()
	}

	fmt.Printf("[saved] %d images\n", count)
}
