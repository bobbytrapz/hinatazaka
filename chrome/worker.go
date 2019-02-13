package chrome

import (
	"context"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// NumTabWorkersPerMember determines how many blogs we download at once
var NumTabWorkersPerMember = 8

// TabJob is a job for a TabWorker
type TabJob struct {
	Blog Blog
}

// TabWorker completes TabJobs
type TabWorker struct {
	tab  Tab
	job  chan TabJob
	stop chan struct{}
}

// TabWorkers manages a pool of TabWorkers
type TabWorkers struct {
	w    []TabWorker
	job  chan TabJob
	pool chan chan TabJob
	wid  int
	wg   sync.WaitGroup
	stop chan struct{}
}

// Wait for jobs to complete
func (tw *TabWorkers) Wait() {
	tw.wg.Wait()
}

// Stop all work
func (tw *TabWorkers) Stop() {
	close(tw.stop)
}

// NewTabWorkers builds a pool of TabWorkers
// each worker opens a new tab so N new tabs are opened in chrome
// after TabWorker.Stop those tabs should close
func NewTabWorkers(ctx context.Context, N int) (tw *TabWorkers) {
	tw = &TabWorkers{
		w:    make([]TabWorker, 0),
		wid:  0,
		stop: make(chan struct{}),
		job:  make(chan TabJob),
		pool: make(chan chan TabJob, N),
	}

	for n := 0; n < N; n++ {
		tab, err := ConnectToNewTab(ctx)
		if err != nil {
			panic(err)
		}

		newWorker := TabWorker{
			tab:  tab,
			job:  make(chan TabJob),
			stop: make(chan struct{}),
		}

		// tab worker
		go func(w TabWorker) {
			for {
				// we are now free for more work
				tw.pool <- w.job

				select {
				case <-ctx.Done():
					return
				case <-w.stop:
					w.tab.PageClose()
					tw.wg.Done()
					return
				case job := <-w.job:
					b := job.Blog
					h := sha1.New()
					h.Write([]byte(b.Link))

					hash := base32.StdEncoding.EncodeToString(h.Sum(nil))
					saveImagesTo := filepath.Join(saveTo, b.Name, b.At.Format("2006-01-02"))
					saveBlogAs := filepath.Join(saveImagesTo, fmt.Sprintf("%s.pdf", hash))

					err := os.MkdirAll(saveImagesTo, os.ModePerm)
					if err != nil {
						Log("chrome.NewTabWorkers: %s", err)
						fmt.Println("[nok]", err)
						return
					}

					fmt.Println("[save]", b.Link)
					w.tab.SaveBlog(ctx, b.Link, saveBlogAs, saveImagesTo)
					tw.wg.Done()
				}
			}
		}(newWorker)

		tw.w = append(tw.w, newWorker)
	}

	// dispatch
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-tw.stop:
				// stop tab workers
				tw.wg.Add(N)
				for _, w := range tw.w {
					close(w.stop)
				}
				// wait for tabs to close
				tw.Wait()
				return
			case job := <-tw.job:
				go func(tj TabJob) {
					// try to get a free worker
					// blocks until a work is free
					ch := <-tw.pool
					// send the job to the free worker
					ch <- tj
				}(job)
			}
		}
	}()

	return
}

// Add distributes a job to free worker
func (tw *TabWorkers) Add(job TabJob) {
	tw.wg.Add(1)
	tw.job <- job
}

// SaveAllBlogs gets the list of blogs and save them all
func SaveAllBlogs(ctx context.Context, root string) error {
	return SaveAllBlogsSince(ctx, root, time.Time{})
}

// SaveAllBlogsSince gets the list of blogs and saves any that came after since
func SaveAllBlogsSince(ctx context.Context, root string, since time.Time) error {
	jobs := make(chan TabJob)
	visit := make(chan string)
	done := make(chan struct{})
	visited := make(map[string]bool)

	// include the given date
	since = since.AddDate(0, 0, -1)

	var rw sync.RWMutex

	// spider
	tab, err := ConnectToNewTab(ctx)
	if err != nil {
		return fmt.Errorf("chrome.SaveAllBlogsSince: %s", err)
	}
	count := 0
	go func() {
		defer tab.PageClose()
		defer close(jobs)
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case link := <-visit:
				// get list of blogs
				tab.PageNavigate(link)
				tab.WaitForLoad()
				rw.Lock()
				visited[link] = true
				rw.Unlock()
				res := tab.Blogs()
				for _, b := range res.Blogs {
					// the blogs are found in reverse chronological order so
					// I think this should work
					if b.At.Before(since) {
						Log("chrome.SaveAllBlogsSince: found oldest blog")
						close(done)
						return
					}
					// send each single blog we find for processing by a tab worker
					jobs <- TabJob{b}
					count++
				}
				// read pages
				wasNewPage := false
				for _, page := range res.Pages {
					rw.RLock()
					if _, ok := visited[page]; !ok {
						visit <- page
						wasNewPage = true
					}
					rw.RUnlock()
				}
				// if we found no new pages then we are done
				if !wasNewPage {
					Log("chrome.SaveAllBlogsSince: found nothing new")
					close(done)
				}
			}
		}
	}()
	visit <- root

	// make some tab workers
	tw := NewTabWorkers(ctx, NumTabWorkersPerMember)

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
