package blog

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bobbytrapz/gochrome"
)

const (
	WaitForBlogList      = 2 * time.Second
	WaitForBlogDownload  = 10 * time.Second
	WaitForBlogRender    = 20 * time.Second
	WaitForSpiderTimeout = 30 * time.Second
)

// ShouldDryRun is the context key indicating a dry run
type ShouldDryRun struct{}

// UserAgent to use when fetching
var UserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36`

// we use for fetching pages and for remote control of chrome
var httpClient = http.Client{
	Timeout: 10 * time.Second,
}

// result from jsBlogs
type blogsFromPage struct {
	Pages []string `json:"pages"`
	Blogs []blog   `json:"blogs"`
}

// an individual blog
type blog struct {
	Title string     `json:"title"`
	Name  string     `json:"name"`
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Day   int        `json:"day"`
	Link  string     `json:"link"`
}

// an image from the blog
type image struct {
	Link string `json:"link"`
	Data []byte
}

// uses Array toString() to make a comma-separated list of image urls
var jsBlogImages = `[...document.querySelectorAll(':scope .p-blog-article img:not(.emoji)')].map(el => el.src).toString()`

// each member has a page that lists all of their blogs
// this extracts the individual blog links from that page
// we can get the member's name and date posted with a bit more work
// we also get all the links to other pages from the pager at the bottom
// the page urls are provided to the spider
var jsBlogs = `
JSON.stringify({
    pages: [...document.querySelectorAll('.c-pager__item--count a')].map(el => el.href),
    blogs: [...document.querySelectorAll('.p-blog-article')].map(el => {
		const link = el.querySelector('.p-button__blog_detail > a').href;
		const head = el.querySelector('.p-blog-article__head');
    	const name = head.querySelector('.c-blog-article__name').textContent.trim();
		const title = head.querySelector('.c-blog-article__title').textContent.trim();
		const t = head.querySelector('.c-blog-article__date').textContent.trim();
    	[year, month, day] = t.split(' ')[0].split('.');
		return {
		  title: title,
		  name: name,
		  year: parseInt(year),
		  month: parseInt(month),
		  day: parseInt(day),
		  link: link
		};
    })
})
`

// blogsFromTab uses jsBlogs to get links and dates of all blogs on a page
func blogsFromTab(tab *gochrome.Tab) (blogsFromPage blogsFromPage, err error) {
	res, err := tab.Evaluate(jsBlogs)
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(res.Result["value"].(string)), &blogsFromPage)
	if err != nil {
		return
	}

	return
}

// uses jsBlogImages to grab images from a blog
func imagesFromBlog(tab *gochrome.Tab) (images []image, err error) {
	res, err := tab.Evaluate(jsBlogImages)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	links := strings.Split(res.Result["value"].(string), ",")
	for _, link := range links {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			var req *http.Request
			req, err = http.NewRequest("GET", l, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", UserAgent)

			var res *http.Response
			res, err = httpClient.Do(req)
			if err != nil {
				return
			}

			var data []byte
			limited := io.LimitReader(res.Body, 100000000)
			data, err = ioutil.ReadAll(limited)
			if err != nil {
				return
			}
			images = append(images, image{
				Link: l,
				Data: data,
			})

			_ = res.Body.Close()
		}(link)
	}
	wg.Wait()

	return
}

func saveBlog(ctx context.Context, tab *gochrome.Tab, link string, title string, name string, at time.Time, saveTo string) error {
	h := sha1.New()
	h.Write([]byte(link))
	hash := base32.StdEncoding.EncodeToString(h.Sum(nil))

	saveImagesTo := filepath.Join(saveTo, name, at.Format("2006-01-02"))
	saveBlogAs := filepath.Join(saveImagesTo, fmt.Sprintf("%s.mhtml", hash))

	fmt.Println("[save]", link)
	fmt.Println("[title]", title)

	if v := ctx.Value(ShouldDryRun{}); v != nil {
		fmt.Printf("[dry-run] %s\n", saveBlogAs)
		return nil
	}

	err := os.MkdirAll(saveImagesTo, os.ModePerm)
	if err != nil {
		fmt.Println("[nok]", err)
		return err
	}

	// visit the blog and take a screenshot
	_, err = tab.Goto(link)
	if err != nil {
		return err
	}

	// make sure everything downloads
	tab.WaitForNetworkIdle(WaitForBlogDownload)
	// make sure everything finishes rendering
	tab.WaitForLoad(WaitForBlogRender)

	err = tab.Snapshot(saveBlogAs)
	if err != nil {
		return err
	}

	// scrape images from an individual blog
	blogImages, err := imagesFromBlog(tab)
	if err != nil {
		return err
	}

	fmt.Printf("%d images from %q\n", len(blogImages), title)

	// save images to disk
	for _, bi := range blogImages {
		saveTo := filepath.Join(saveImagesTo, filepath.Base(bi.Link))
		err = ioutil.WriteFile(saveTo, bi.Data, 0644)
		if err != nil {
			return err
		}
		fmt.Println("[save] [image]", saveTo)
	}

	return nil
}

func SaveBlogsSince(ctx context.Context, browser *gochrome.Browser, root string, since time.Time, saveTo string, maxSaved int) error {
	visit := make(chan string)
	visited := make(map[string]bool)

	// use tokyo time
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	// spider
	spiderBlogList, err := browser.NewTab(ctx)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsSince: %s", err)
	}
	defer spiderBlogList.Close()

	tabPool, err := browser.NewTabPool(ctx, 8)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsSince: %s", err)
	}
	defer tabPool.Close()

	timeout := time.NewTimer(WaitForSpiderTimeout)

	count := 0

	// run spider
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeout.C:
				// dumb way to stop searching for blogs
				// use a timeout to decide when done
				// or to just stop if we have network issues
				gochrome.Log("blog.SaveBlogsSince: timeout since we found nothing")
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
					timeout.Reset(WaitForSpiderTimeout)
				}

				// get list of blogs
				_, err = spiderBlogList.Goto(link)
				if err != nil {
					panic(err)
				}
				spiderBlogList.WaitForNetworkIdle(WaitForBlogList)
				blogsFromPage, err := blogsFromTab(spiderBlogList)
				if err != nil {
					return
				}

				// read pages
				for _, page := range blogsFromPage.Pages {
					go func(p string) {
						visit <- p
					}(page)
				}

				// save each blog
				for _, b := range blogsFromPage.Blogs {
					if count >= maxSaved {
						gochrome.Log("blog.SaveBlogsSince: reached max blog save count")
						return
					}

					// the blogs are found in reverse chronological order so
					// I think this should work
					at := time.Date(b.Year, b.Month, b.Day, 23, 59, 59, 0, loc)
					if at.Before(since) {
						gochrome.Log("blog.SaveBlogsSince: found oldest blog")
						return
					}

					// if there is space between this member's names remove it
					author := strings.ReplaceAll(b.Name, " ", "")
					link := b.Link
					title := b.Title

					// save a blog
					count++
					tab := tabPool.Grab()
					go func() {
						defer tabPool.Release(tab)

						err := saveBlog(ctx, tab, link, title, author, at, saveTo)
						if err != nil {
							gochrome.Log("blog.SaveBlogsSince: saveBlog: %s", err)
						}
					}()
				}
			}
		}
	}()

	// seed spider
	visit <- root

	// give spider some time to start
	<-time.After(WaitForBlogList + WaitForBlogDownload)

	// wait for jobs to finish
	tabPool.Wait()

	fmt.Println("[saved]", count, "blogs")

	return nil
}

// SaveBlogsOn finds up to maxSaved blogs on a given date and saves them
func SaveBlogsOn(ctx context.Context, browser *gochrome.Browser, authorShouldSave map[string]bool, on time.Time, saveTo string, maxSaved int) error {
	dy := fmt.Sprintf("%04d%02d%02d", on.Year(), on.Month(), on.Day())
	listPage := fmt.Sprintf("https://www.hinatazaka46.com/s/official/diary/member/list?ima=0000&dy=%s", dy)

	// use tokyo time
	// note: we do not really need this here for now
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	listPageTab, err := browser.NewTab(ctx)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsOn: %s", err)
	}
	defer listPageTab.Close()
	count := 0

	tabPool, err := browser.NewTabPool(ctx, 8)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsOn: %s", err)
	}
	defer tabPool.Close()

	// get list of blogs
	fmt.Println("[visit]", listPage)
	_, err = listPageTab.Goto(listPage)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsOn: %s", err)
	}
	listPageTab.WaitForNetworkIdle(WaitForBlogList)
	blogsFromPage, err := blogsFromTab(listPageTab)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsOn: %s", err)
	}

	for _, b := range blogsFromPage.Blogs {
		if count > maxSaved {
			gochrome.Log("blog.SaveBlogsOn: reached max blog save count")
			break
		}

		at := time.Date(b.Year, b.Month, b.Day, 0, 0, 0, 0, loc)
		// if there is space between this member's name remove it
		author := strings.ReplaceAll(b.Name, " ", "")
		if _, ok := authorShouldSave[author]; !ok {
			continue
		}
		link := b.Link
		title := b.Title

		count++
		tab := tabPool.Grab()
		go func() {
			defer tabPool.Release(tab)

			if count >= maxSaved {
				gochrome.Log("blog.SaveBlogsOn: reached max blog save count")
				return
			}
			err := saveBlog(ctx, tab, link, title, author, at, saveTo)
			if err != nil {
				gochrome.Log("blog.SaveBlogsOn: %s", err)
			}
		}()
	}
	tabPool.Wait()

	fmt.Println("[saved]", count, "blogs")

	return nil
}
