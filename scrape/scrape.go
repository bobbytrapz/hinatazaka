package scrape

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobbytrapz/gochrome"
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
func SaveImagesFrom(ctx context.Context, browser *gochrome.Browser, link string, saveImagesTo string, jsCode string) error {
	tab, err := browser.NewTab(ctx)
	if err != nil {
		return fmt.Errorf("scrape.SaveImagesFrom: %s", err)
	}
	defer tab.Close()
	SaveImagesFromTabWith(ctx, tab, link, saveImagesTo, jsCode)
	return nil
}

// SaveImagesFromTabWith sends given javascript to get urls for images
// then it downloads each image from a list of comma-separated urls
func SaveImagesFromTabWith(ctx context.Context, tab *gochrome.Tab, link string, saveTo string, jsCode string) {
	_, err := tab.Goto(link)
	if err != nil {
		panic(err)
	}
	tab.WaitForLoad(10 * time.Second)

	got, err := tab.Evaluate(jsCode)
	if err != nil {
		panic(err)
	}

	value, ok := got.Result["value"]
	if !ok {
		gochrome.Log("scrape.SaveImagesFromTabWith: no images")
		return
	}
	links := value.(string)
	gochrome.Log("scrape.SaveImagesFromTabWith: links: %+v", links)

	count := 0
	for _, u := range strings.Split(links, ",") {
		if u == "" {
			continue
		}

		purl, err := url.Parse(u)
		if err != nil {
			gochrome.Log("scrape.SaveImagesFromTabWith: %s", err)
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
			gochrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			gochrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		fmt.Println("[save]", fn)
		_, err = io.Copy(f, res.Body)
		if err != nil {
			gochrome.Log("scrape.SaveImagesFromTabWith: %s", err)
			fmt.Println("[nok]", err)
		}
		count++

		f.Close()
		res.Body.Close()
	}

	fmt.Printf("[saved] %d images\n", count)
}
