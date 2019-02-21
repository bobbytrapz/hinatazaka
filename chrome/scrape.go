package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobbytrapz/hinatazaka/fetch"
	"github.com/bobbytrapz/hinatazaka/options"
)

var saveTo = options.Get("save_to")

// WaitForLoad waits a few seconds for page to load
// todo: if we start missing blogs we need to find a better way
func (t Tab) WaitForLoad() {
	<-time.After(2 * time.Second)
}

// uses Array toString() to make a comma-separated list of image urls
var jsBlogImages = `[...document.querySelectorAll(':scope article img:not(.emoji)')].map(el => el.src).toString()`

// ResJSString is a response
type ResJSString struct {
	Result struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"result"`
}

// BlogImages sends javascript to get urls for blog images
func (t Tab) BlogImages(ctx context.Context, saveTo string) {
	t.Command("Runtime.evaluate", TabParams{
		"expression": jsBlogImages,
	})
	// wait for reponse
	data := <-t.recv
	var res ResJSString
	err := json.Unmarshal(data, &res)
	if err != nil {
		Log("chrome.Tab.BlogImages: %s", err)
	}
	Log("chrome.Tab.BlogImages: res: %+v", res)

	for _, u := range strings.Split(res.Result.Value, ",") {
		if u == "" {
			continue
		}
		fn := filepath.Join(saveTo, filepath.Base(u))

		res, err := fetch.Get(ctx, u)
		if err != nil {
			Log("chrome.Tab.BlogImages: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			Log("chrome.Tab.BlogImages: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		fmt.Println("[save]", fn)
		_, err = io.Copy(f, res.Body)
		if err != nil {
			Log("chrome.Tab.BlogImages: %s", err)
			fmt.Println("[nok]", err)
		}

		f.Close()
		res.Body.Close()
	}
}

// ResJSBlog is needed to retrieve a blog
type ResJSBlog struct {
	Pages []string `json:"pages"`
	Blogs []Blog   `json:"blogs"`
}

// Blog is an individual blog
type Blog struct {
	Name string    `json:"name"`
	At   time.Time `json:"at"`
	Link string    `json:"link"`
}

// each member has a page that lists all of their blogs
// this extracts the individual blog links from that page
// we can get the member's name and date posted with a bit more work
// we also get all the links to other pages from the pager at the bottom
// the page urls are provided to the spider
var jsBlogs = `
JSON.stringify({
	pages: [...document.querySelectorAll('.pager a')].map(el => el.href),
	blogs: [...document.querySelectorAll('article > .innerHead')].map(el => {
  	name = el.querySelector('.box-ttl .name').textContent.trim();
  	link = el.querySelector('.box-ttl a').href;
    t = el.querySelectorAll('.box-date > time');
  	yearmonth = t[0].textContent.replace('.', '-');
    day = t[1].textContent;
    return {
      name: name,
      at: new Date(yearmonth + '-' + day),
      link: link,
    }
})})
`

// Blogs sends javascript to get links and dates of all blogs on a page
func (t Tab) Blogs() (blogs ResJSBlog) {
	t.Command("Runtime.evaluate", TabParams{
		"expression": jsBlogs,
	})
	// wait for reponse
	data := <-t.recv
	var res ResJSString
	err := json.Unmarshal(data, &res)
	if err != nil {
		Log("chrome.Tab.Blogs: %s", err)
		return
	}

	err = json.Unmarshal([]byte(res.Result.Value), &blogs)
	if err != nil {
		Log("chrome.Tab.Blogs: %s", err)
		return
	}
	Log("chrome.Tab.Blogs: res: %+v", blogs)

	return
}

// SaveBlog saves a single individual blog
func (t Tab) SaveBlog(ctx context.Context, link string, saveBlogAs string, saveImagesTo string) {
	t.PageNavigate(link)
	t.WaitForLoad()
	t.PagePrintToPDF(saveBlogAs)
	t.BlogImages(ctx, saveImagesTo)
}

// format string requires a list of classes
var jsImages = `[...document.querySelectorAll('img%s')].map(el => el.src).toString()`

// SaveImages sends javascript to get urls for images
func (t Tab) SaveImages(ctx context.Context, saveTo string, classes string) {
	t.Command("Runtime.evaluate", TabParams{
		"expression": fmt.Sprintf(jsImages, classes),
	})
	// wait for reponse
	data := <-t.recv
	var res ResJSString
	err := json.Unmarshal(data, &res)
	if err != nil {
		Log("chrome.Tab.SaveImages: %s", err)
	}
	Log("chrome.Tab.SaveImages: res: %+v", res)

	for _, u := range strings.Split(res.Result.Value, ",") {
		if u == "" {
			continue
		}
		fn := filepath.Join(saveTo, filepath.Base(u))

		res, err := fetch.Get(ctx, u)
		if err != nil {
			Log("chrome.Tab.SaveImages: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			Log("chrome.Tab.SaveImages: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		fmt.Println("[save]", fn)
		_, err = io.Copy(f, res.Body)
		if err != nil {
			Log("chrome.Tab.SaveImages: %s", err)
			fmt.Println("[nok]", err)
		}

		f.Close()
		res.Body.Close()
	}
}

// SaveImagesFrom a webpage with the given classes
func (t Tab) SaveImagesFrom(ctx context.Context, link string, saveImagesTo string, classes string) {
	t.PageNavigate(link)
	t.WaitForLoad()
	t.SaveImages(ctx, saveImagesTo, classes)
}
