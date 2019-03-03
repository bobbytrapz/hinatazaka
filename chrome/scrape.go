package chrome

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobbytrapz/hinatazaka/fetch"
	"github.com/bobbytrapz/hinatazaka/options"
)

// SaveTo decides where we save everything that we find
var SaveTo = options.Get("save_to")

// WaitForLoad waits a few seconds for page to load
// todo: if we start missing blogs we need to find a better way
func (t Tab) WaitForLoad() {
	<-time.After(5 * time.Second)
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

// SaveImagesWith sends given javascript to get urls for images
// then it downloads each image from a list of comma-separated urls
func (t Tab) SaveImagesWith(ctx context.Context, saveTo string, jsCode string) {
	t.Command("Runtime.evaluate", TabParams{
		"expression": jsCode,
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

		purl, err := url.Parse(u)
		if err != nil {
			Log("chrome.Tab.SaveImages: %s", err)
			fmt.Println("[nok]", err)
			continue
		}
		fn := filepath.Join(saveTo, filepath.Base(purl.Path))

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

// SaveImages a webpage with the given classes
func (t Tab) SaveImages(ctx context.Context, link string, saveImagesTo string, jsCode string) {
	t.PageNavigate(link)
	t.WaitForLoad()
	t.SaveImagesWith(ctx, saveImagesTo, jsCode)
}

// SaveAllBlogs gets the list of blogs and save them all
func SaveAllBlogs(ctx context.Context, root string) error {
	return SaveAllBlogsSince(ctx, root, time.Time{}, math.MaxInt32)
}

// SaveAllBlogsSince gets the list of blogs and saves any that came after since
func SaveAllBlogsSince(ctx context.Context, root string, since time.Time, maxSaved int) error {
	jobs := make(chan TabJob)
	visit := make(chan string)
	done := make(chan struct{})
	visited := make(map[string]bool)

	// spider
	tab, err := ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("chrome.SaveAllBlogsSince: %s", err)
	}
	count := 0
	spiderTimeout := time.Duration(WorkerDelay) * time.Second
	timeout := time.NewTimer(spiderTimeout)
	go func() {
		defer tab.PageClose()
		defer close(jobs)
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-timeout.C:
				// dumb way to stop searching for blogs
				// use a timeout to decide when done
				// or to just stop if we have network issues
				Log("chrome.SaveAllBlogsSince: timeout since we found nothing")
				return
			case link := <-visit:
				// check visited
				_, ok := visited[link]
				if ok {
					continue
				}

				// save visit
				fmt.Println("[visit]", link)
				visited[link] = true

				// we found a page so reset timeout
				if timeout.Stop() {
					timeout.Reset(spiderTimeout)
				}

				// get list of blogs
				tab.PageNavigate(link)
				tab.WaitForLoad()
				res := tab.Blogs()
				for _, b := range res.Blogs {
					if count >= maxSaved {
						Log("chrome.SaveAllBlogsSince: reached max blog save count")
						close(done)
						return
					}
					// the blogs are found in reverse chronological order so
					// I think this should work
					if b.At.Before(since) {
						Log("chrome.SaveAllBlogsSince: found oldest blog")
						close(done)
						return
					}
					// send each single blog we find for processing by a tab worker
					jobs <- TabJob{
						Name: b.Name,
						Link: b.Link,
						At:   b.At,
					}
					count++
				}
				// read pages
				for _, page := range res.Pages {
					go func(p string) {
						visit <- p
					}(page)
				}
			}
		}
	}()
	visit <- root

	jobFn := func(tab Tab, job TabJob) error {
		h := sha1.New()
		h.Write([]byte(job.Link))

		hash := base32.StdEncoding.EncodeToString(h.Sum(nil))
		saveImagesTo := filepath.Join(SaveTo, job.Name, job.At.Format("2006-01-02"))
		saveBlogAs := filepath.Join(saveImagesTo, fmt.Sprintf("%s.pdf", hash))

		err := os.MkdirAll(saveImagesTo, os.ModePerm)
		if err != nil {
			fmt.Println("[nok]", err)
			return fmt.Errorf("chrome.SaveAllBlogsSince: %s", err)
		}

		fmt.Println("[save]", job.Link)
		tab.SaveBlog(ctx, job.Link, saveBlogAs, saveImagesTo)

		return nil
	}

	// make some tab workers
	tw := NewTabWorkers(ctx, NumTabWorkersPerMember, jobFn)

	// distribute jobs
	for tj := range jobs {
		tw.Add(tj)
	}
	// wait for all our jobs to finish
	tw.Wait()
	// remove all the worker tabs
	tw.Stop()
	fmt.Println("[saved]", count, "blogs")

	return nil
}

// SaveImagesFrom a webpage
func SaveImagesFrom(ctx context.Context, link string, saveImagesTo string, jsCode string) error {
	tab, err := ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("chrome.SaveAllBlogsSince: %s", err)
	}
	defer tab.PageClose()
	tab.SaveImages(ctx, link, saveImagesTo, jsCode)
	return nil
}
