package blog

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

var browser *rod.Browser

var openBrowserOnce sync.Once

func openBrowser() {
	log.Print("blog: Opening browser...")
	browser = rod.New().MustConnect()
}

func SaveBlogsSince(ctx context.Context, root string, since time.Time, saveTo string, maxSaved uint64) error {
	openBrowserOnce.Do(openBrowser)

	visit := make(chan string)
	var visited sync.Map
	var failed sync.Map

	// use tokyo time
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	poolCount := 8
	pool := rod.NewPagePool(poolCount)
	createFn := func() *rod.Page {
		return browser.MustPage().MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: options.Get("user_agent"),
		})
	}

	var count atomic.Uint64

	timeout := time.NewTimer(WaitForSpiderTimeout)

	job := func() error {
		page := pool.Get(createFn)
		defer pool.Put(page)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timeout.C:
				return nil
			case link := <-visit:
				if _, ok := visited.Load(link); ok {
					continue
				}

				// we found a page so reset timeout
				if timeout.Stop() {
					timeout.Reset(WaitForSpiderTimeout)
				}

				visited.Store(link, link)
				log.Printf("blog: visit: %q", link)

				err = page.Timeout(WaitForSpiderTimeout).MustNavigate(link).WaitLoad()
				if err != nil {
					// delete so we can maybe try again
					visited.Delete(link)
					continue
				}

				blogs, err := getBlogsFromPage(page)
				if err != nil {
					// delete so we can maybe try again
					visited.Delete(link)
					continue
				}

				// add more pages to visit
				for _, blogPage := range blogs.Pages {
					go func(p string) {
						visit <- p
					}(blogPage)
				}

				// download blogs
				for _, b := range blogs.Blogs {
					if count.Load() >= maxSaved {
						log.Print("blog.SaveBlogsSince: reached max blog save count")
						return nil
					}
					count.Add(1)

					// the blogs are found in reverse chronological order so
					// I think this should work
					at := time.Date(b.Year, b.Month, b.Day, 23, 59, 59, 0, loc)
					if at.Before(since) {
						log.Print("blog.SaveBlogsSince: found oldest blog")
						break
					}

					// if there is space between this member's names remove it
					author := strings.ReplaceAll(b.Name, " ", "")
					blogLink := b.Link
					blogTitle := b.Title

					// save a blog
					err = saveBlogFromPage(ctx, page, blogLink, blogTitle, author, at, saveTo)
					if err != nil {
						log.Printf("blog.SaveBlogsSince: saveBlogFromPage: %s", err)
						failed.Store(b.Link, b.Link)
					}
				}
			}
		}
	}

	// spider
	var wg sync.WaitGroup
	for i := 0; i < poolCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := job(); err != nil {
				panic(err)
			}
		}()
	}

	// initialize spider
	visit <- root

	// give spider some time to start
	<-time.After(WaitForBlogDownload)

	wg.Wait()

	pool.Cleanup(func(p *rod.Page) {
		p.MustClose()
	})

	visited.Range(func(k, v interface{}) bool {
		fmt.Println("[visited]", v.(string))
		return true
	})
	failed.Range(func(k, v interface{}) bool {
		fmt.Println("[failed]", v.(string))
		return true
	})
	fmt.Println("[saved]", count.Load(), "blogs")

	return nil
}

func SaveBlogsOn(ctx context.Context, authorShouldSave map[string]bool, on time.Time, saveTo string, maxSaved int) error {
	openBrowserOnce.Do(openBrowser)

	dy := fmt.Sprintf("%04d%02d%02d", on.Year(), on.Month(), on.Day())
	listPage := fmt.Sprintf("https://www.hinatazaka46.com/s/official/diary/member/list?ima=0000&dy=%s", dy)

	// use tokyo time
	// note: we do not really need this here for now
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	poolCount := 8
	pool := rod.NewPagePool(poolCount)
	createFn := func() *rod.Page {
		return browser.MustPage().MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: options.Get("user_agent"),
		})
	}

	// get list of blogs
	fmt.Println("[visit]", listPage)

	page := pool.Get(createFn)
	page.Timeout(WaitForSpiderTimeout).MustNavigate(listPage).MustWaitLoad()

	count := 0

	blogs, err := getBlogsFromPage(page)
	if err != nil {
		return fmt.Errorf("blog.SaveBlogsOn: %s", err)
	}

	for _, b := range blogs.Blogs {
		if count > maxSaved {
			log.Print("blog.SaveBlogsOn: reached max blog save count")
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
		if count >= maxSaved {
			log.Print("blog.SaveBlogsOn: reached max blog save count")
			return nil
		}

		err = saveBlogFromPage(ctx, page, link, title, author, at, saveTo)
		if err != nil {
			log.Printf("blog.SaveBlogsOn: %s", err)
		}
	}

	fmt.Println("[saved]", count, "blogs")

	return nil
}

// use jsBlogs to get blogs from a page
func getBlogsFromPage(page *rod.Page) (blogsFromPage, error) {
	var blogs blogsFromPage
	evaluated, err := page.Eval(jsBlogs)
	if err != nil {
		return blogs, err
	}

	err = evaluated.Value.Unmarshal(&blogs)
	if err != nil {
		return blogs, err
	}

	return blogs, nil
}

// uses jsBlogImages to grab images from a blog
func getImagesFromPage(page *rod.Page) (images []image, err error) {
	evaluated, err := page.Eval(jsBlogImages)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	links := strings.Split(evaluated.Value.String(), ",")
	for _, link := range links {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			var req *http.Request
			req, err = http.NewRequest(http.MethodGet, l, nil)
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
			data, err = io.ReadAll(limited)
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

func saveBlogFromPage(ctx context.Context, page *rod.Page, link string, title string, name string, at time.Time, saveTo string) error {
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
	err = page.Navigate(link)
	if err != nil {
		return err
	}

	page.MustWaitLoad()

	snapshot, err := proto.PageCaptureSnapshot{}.Call(page)
	if err != nil {
		return fmt.Errorf("while taking snapshot: %w", err)
	}
	err = utils.OutputFile(saveBlogAs, snapshot.Data)
	if err != nil {
		return fmt.Errorf("while saving snapshot: %w", err)
	}

	// scrape images from an individual blog
	blogImages, err := getImagesFromPage(page)
	if err != nil {
		return err
	}

	fmt.Printf("%d images from %q\n", len(blogImages), title)

	// save images to disk
	for _, bi := range blogImages {
		saveTo := filepath.Join(saveImagesTo, filepath.Base(bi.Link))
		err = os.WriteFile(saveTo, bi.Data, 0644)
		if err != nil {
			return err
		}
		fmt.Println("[save] [image]", saveTo)
	}

	return nil
}
