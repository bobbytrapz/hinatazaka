package scrape

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

	"github.com/bobbytrapz/hinatazaka/chrome"
	"github.com/bobbytrapz/hinatazaka/options"
)

// NumTabWorkersPerMember decides how many tabs to open for each blog spider
var NumTabWorkersPerMember = 8

// SaveTo decides where we save everything that we find
var SaveTo = options.Get("save_to")

// uses Array toString() to make a comma-separated list of image urls
var jsBlogImages = `[...document.querySelectorAll(':scope article img:not(.emoji)')].map(el => el.src).toString()`

// ResJSString is a response
type ResJSString struct {
	Result struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"result"`
}

// SaveBlogImagesFromTab sends javascript to get urls for blog images
func SaveBlogImagesFromTab(ctx context.Context, tab chrome.Tab, saveTo string) {
	tab.Command("Runtime.evaluate", chrome.TabParams{
		"expression": jsBlogImages,
	})
	// wait for reponse
	data := tab.Wait()
	var res ResJSString
	err := json.Unmarshal(data, &res)
	if err != nil {
		chrome.Log("scrape.SaveBlogImagesFromTab: %s", err)
	}
	chrome.Log("scrape.SaveBlogImagesFromTab: res: %+v", res)

	for _, u := range strings.Split(res.Result.Value, ",") {
		if u == "" {
			continue
		}
		fn := filepath.Join(saveTo, filepath.Base(u))

		res, err := chrome.Fetch(ctx, u)
		if err != nil {
			chrome.Log("scrape.SaveBlogImagesFromTab: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			chrome.Log("scrape.SaveBlogImagesFromTab: %s", err)
			fmt.Println("[nok]", err)
			continue
		}

		fmt.Println("[save]", fn)
		_, err = io.Copy(f, res.Body)
		if err != nil {
			chrome.Log("scrape.SaveBlogImagesFromTab: %s", err)
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
	Name  string     `json:"name"`
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Day   int        `json:"day"`
	Link  string     `json:"link"`
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
    [year, month] = t[0].textContent.split('.')
    day = t[1].textContent;
    return {
      name: name,
      year: parseInt(year),
      month: parseInt(month),
      day: parseInt(day),
      link: link,
    }
})})
`

// BlogsFromTab sends javascript to get links and dates of all blogs on a page
func BlogsFromTab(tab chrome.Tab) (blogs ResJSBlog) {
	tab.Command("Runtime.evaluate", chrome.TabParams{
		"expression": jsBlogs,
	})
	// wait for reponse
	data := tab.Wait()
	var res ResJSString
	err := json.Unmarshal(data, &res)
	if err != nil {
		chrome.Log("scrape.BlogsFromTab: %s", err)
		return
	}

	err = json.Unmarshal([]byte(res.Result.Value), &blogs)
	if err != nil {
		chrome.Log("scrape.BlogsFromTab: %s", err)
		return
	}
	chrome.Log("scrape.BlogsFromTab: res: %+v", blogs)

	return
}

// SaveBlogFromTab saves a single individual blog
func SaveBlogFromTab(ctx context.Context, tab chrome.Tab, link string, saveBlogAs string, saveImagesTo string) {
	tab.PageNavigate(link)
	tab.WaitForLoad()
	tab.PagePrintToPDF(saveBlogAs)
	SaveBlogImagesFromTab(ctx, tab, saveImagesTo)
}

// SaveImagesFromTabWith sends given javascript to get urls for images
// then it downloads each image from a list of comma-separated urls
func SaveImagesFromTabWith(ctx context.Context, tab chrome.Tab, link string, saveTo string, jsCode string) {
	tab.PageNavigate(link)
	tab.WaitForLoad()

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

		res, err := chrome.Fetch(ctx, u)
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

		f.Close()
		res.Body.Close()
	}
}

// SaveAllBlogs gets the list of blogs and save them all
func SaveAllBlogs(ctx context.Context, root string) error {
	return SaveAllBlogsSince(ctx, root, time.Time{}, math.MaxInt32)
}

// SaveAllBlogsSince gets the list of blogs and saves any that came after since
func SaveAllBlogsSince(ctx context.Context, root string, since time.Time, maxSaved int) error {
	jobs := make(chan chrome.TabJob)
	visit := make(chan string)
	done := make(chan struct{})
	visited := make(map[string]bool)

	// use tokyo time
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	// spider
	tab, err := chrome.ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("scrape.SaveAllBlogsSince: %s", err)
	}
	count := 0
	spiderTimeout := time.Duration(30) * time.Second
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
				chrome.Log("scrape.SaveAllBlogsSince: timeout since we found nothing")
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
				res := BlogsFromTab(tab)
				for _, b := range res.Blogs {
					if count >= maxSaved {
						chrome.Log("scrape.SaveAllBlogsSince: reached max blog save count")
						close(done)
						return
					}
					// the blogs are found in reverse chronological order so
					// I think this should work
					at := time.Date(b.Year, b.Month, b.Day, 0, 0, 0, 0, loc)
					if at.Before(since) {
						chrome.Log("scrape.SaveAllBlogsSince: found oldest blog")
						close(done)
						return
					}
					// send each single blog we find for processing by a tab worker
					jobs <- chrome.TabJob{
						Link: b.Link,
						Data: map[string]interface{}{
							"Name": b.Name,
							"At":   at,
						},
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

	jobFn := func(tab chrome.Tab, job chrome.TabJob) error {
		name := job.GetString("Name")
		if name == "" {
			return fmt.Errorf("scrape.SaveAllBlogsSince: could not find 'Name'")
		}
		t := job.GetTime("At")
		if t.IsZero() {
			return fmt.Errorf("scrape.SaveAllBlogsSince: could not find 'At'")
		}
		at := t.Format("2006-01-02")

		h := sha1.New()
		h.Write([]byte(job.Link))
		hash := base32.StdEncoding.EncodeToString(h.Sum(nil))
		saveImagesTo := filepath.Join(SaveTo, name, at)
		saveBlogAs := filepath.Join(saveImagesTo, fmt.Sprintf("%s.pdf", hash))

		err := os.MkdirAll(saveImagesTo, os.ModePerm)
		if err != nil {
			fmt.Println("[nok]", err)
			return fmt.Errorf("scrape.SaveAllBlogsSince: %s", err)
		}

		fmt.Println("[save]", job.Link)
		SaveBlogFromTab(ctx, tab, job.Link, saveBlogAs, saveImagesTo)

		return nil
	}

	// make some tab workers
	tw := chrome.NewTabWorkers(ctx, NumTabWorkersPerMember, jobFn)

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
	tab, err := chrome.ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("scrape.SaveImagesFrom: %s", err)
	}
	defer tab.PageClose()
	SaveImagesFromTabWith(ctx, tab, link, saveImagesTo, jsCode)
	return nil
}
