package chrome

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// NumTabWorkersPerMember determines how many blogs we download at once
var NumTabWorkersPerMember = 8

// WorkerDelay is how long in seconds a worker waits before reentering the pool
// a bit random though
var WorkerDelay = 30

// TabJob is a job for a TabWorker
type TabJob struct {
	Name string    `json:"name"`
	At   time.Time `json:"at"`
	Link string    `json:"link"`
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
// each job we receieve is handled by the job function
func NewTabWorkers(ctx context.Context, N int, jobFn func(Tab, TabJob) error) (tw *TabWorkers) {
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
					if err := jobFn(w.tab, job); err != nil {
						Log("chrome.NewTabWorkers: %s", err)
						return
					}
					tw.wg.Done()
				}

				// wait a bit to be nice
				waitABit := WorkerDelay/2 + rand.Intn(WorkerDelay)
				<-time.After(time.Duration(waitABit) * time.Second)
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
