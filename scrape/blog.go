package scrape

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bobbytrapz/chrome"
)

// uses Array toString() to make a comma-separated list of image urls
var jsBlogImages = `[...document.querySelectorAll(':scope .p-blog-article img:not(.emoji)')].map(el => el.src).toString()`

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
	Title string     `json:"title"`
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
    pages: [...document.querySelectorAll('.c-pager__item--count a')].map(el => el.href),
    blogs: [...document.querySelectorAll('.p-blog-article')].map(el => {
		link = el.querySelector('.p-button__blog_detail > a').href;
		head = el.querySelector('.p-blog-article__head');
    name = head.querySelector('.c-blog-article__name').textContent.trim();
		title = head.querySelector('.c-blog-article__title').textContent.trim();
		t = head.querySelector('.c-blog-article__date').textContent.trim();
    [year, month, day] = t.split(' ')[0].split('.');
    return {
      title: title,
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
	tab.WaitForLoad(15 * time.Second)
	tab.CaptureSnapshot(saveBlogAs)
	SaveBlogImagesFromTab(ctx, tab, saveImagesTo)
}

func newSaveBlogTabWorkerFn(ctx context.Context) chrome.TabWorkerFn {
	return func(tab chrome.Tab, job chrome.TabJob) error {
		name := job.GetString("Name")
		if name == "" {
			return fmt.Errorf("scrape.newSaveBlogTabWorkerFn: could not find 'Name'")
		}
		t := job.GetTime("At")
		if t.IsZero() {
			return fmt.Errorf("scrape.newSaveBlogTabWorkerFn: could not find 'At'")
		}
		at := t.Format("2006-01-02")

		h := sha1.New()
		h.Write([]byte(job.Link))
		hash := base32.StdEncoding.EncodeToString(h.Sum(nil))
		saveImagesTo := filepath.Join(SaveTo, name, at)
		saveBlogAs := filepath.Join(saveImagesTo, fmt.Sprintf("%s.mhtml", hash))

		err := os.MkdirAll(saveImagesTo, os.ModePerm)
		if err != nil {
			fmt.Println("[nok]", err)
			return fmt.Errorf("scrape.newSaveBlogTabWorkerFn: %s", err)
		}

		fmt.Println("[save]", job.Link)
		fmt.Println("[title]", job.GetString("Title"))
		SaveBlogFromTab(ctx, tab, job.Link, saveBlogAs, saveImagesTo)

		return nil
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
				tab.WaitForLoad(10 * time.Second)
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
							"Title": b.Title,
							"Name":  b.Name,
							"At":    at,
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

	jobFn := newSaveBlogTabWorkerFn(ctx)

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

// SaveBlogsOn finds up to maxSaved blogs on a given date and saves them
func SaveBlogsOn(ctx context.Context, names map[string]bool, on time.Time, maxSaved int) error {
	jobs := make(chan chrome.TabJob)
	dy := fmt.Sprintf("%04d%02d%02d", on.Year(), on.Month(), on.Day())
	listPage := fmt.Sprintf("https://www.hinatazaka46.com/s/official/diary/member/list?ima=0000&dy=%s", dy)

	// use tokyo time
	// note: we do not really need this here for now
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	tab, err := chrome.ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("scrape.SaveBlogsOn: %s", err)
	}
	count := 0

	go func() {
		// get list of blogs
		fmt.Println("[visit]", listPage)
		tab.PageNavigate(listPage)
		tab.WaitForLoad(10 * time.Second)
		res := BlogsFromTab(tab)

		for _, b := range res.Blogs {
			at := time.Date(b.Year, b.Month, b.Day, 0, 0, 0, 0, loc)
			if count >= maxSaved {
				chrome.Log("scrape.SaveBlogsOn: reached max blog save count")
				return
			}

			author := strings.ReplaceAll(b.Name, " ", "")
			if _, ok := names[author]; !ok {
				continue
			}

			// send each single blog we find for processing by a tab worker
			jobs <- chrome.TabJob{
				Link: b.Link,
				Data: map[string]interface{}{
					"Title": b.Title,
					"Name":  b.Name,
					"At":    at,
				},
			}
			count++
		}

		close(jobs)
	}()

	jobFn := newSaveBlogTabWorkerFn(ctx)

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
