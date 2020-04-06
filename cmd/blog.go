package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/bobbytrapz/chrome"
	"github.com/bobbytrapz/hinatazaka/members"
	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/bobbytrapz/hinatazaka/scrape"
	"github.com/spf13/cobra"
)

var saveBlogsSince string
var saveBlogsOn string
var saveTo string
var maxSaved int
var since time.Time
var shouldPrintPath bool
var shouldDryRun bool

func init() {
	rootCmd.AddCommand(blogCmd)
	blogCmd.Flags().StringVar(&saveBlogsSince, "since", "", "Save any blogs newer than this date ex: 2019-03-27")
	blogCmd.Flags().StringVar(&saveBlogsOn, "on", "", "Save any blogs posted on this date ex: 2019-03-27")
	blogCmd.Flags().IntVar(&maxSaved, "count", math.MaxInt32, "The max number of blogs to save.")
	blogCmd.Flags().StringVar(&saveTo, "saveto", "", "Directory path to save blog data to")
	blogCmd.Flags().BoolVar(&shouldPrintPath, "path", false, "Print the path where we will save blog data")
	blogCmd.Flags().BoolVar(&shouldDryRun, "dry-run", false, "Show where we would save a blog but do not save it")
}

var blogCmd = &cobra.Command{
	Use:   "blog [members]",
	Short: "Save a blog as a pdf along with save each image",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return errors.New("We need at least one name/nickname of a hinatazaka member")
		}

		if saveTo == "" {
			saveTo = options.Get("save_to")
		}

		if stat, err := os.Stat(saveTo); os.IsNotExist(err) || !stat.IsDir() {
			return errors.New("Save path must be a directory")
		}

		if saveBlogsSince != "" && saveBlogsOn != "" {
			return errors.New("You cannot use both 'on' and 'since'")
		}

		if shouldPrintPath && saveBlogsSince != "" {
			return errors.New("You cannot use both 'path' and 'since'")
		}

		// use tokyo time
		loc, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			panic(err)
		}

		y, m, d := time.Now().In(loc).Date()
		today := time.Date(y, m, d, 0, 0, 0, 0, loc)

		if saveBlogsOn != "" {
			saveBlogsSince = saveBlogsOn
		}

		switch saveBlogsSince {
		case "":
			fallthrough
		case "today":
			// same as default
			since = today
		case "forever":
			// save all blogs since forever
			since = time.Time{}
		case "yesterday":
			since = today.AddDate(0, 0, -1)
		case "week":
			// within this week
			weekday := today.Weekday()
			since = today.AddDate(0, 0, -int(weekday))
		case "month":
			// within this month
			day := today.Day()
			since = today.AddDate(0, 0, -int(day)+1)
		case "year":
			// within this year
			month := today.Month()
			day := today.Day()
			since = today.AddDate(0, -int(month)+1, -int(day)+1)
		default:
			t, err := time.Parse("2006-01-02", saveBlogsSince)
			if err == nil {
				// parsed a date
				y, m, d := t.In(loc).Date()
				since = time.Date(y, m, d, 0, 0, 0, 0, loc)
			} else {
				// not a date; check for a number of days ago
				numDays, e := strconv.ParseInt(saveBlogsSince, 10, 64)
				if e != nil {
					return err
				}
				if numDays < 0 {
					return errors.New("Number of days must be positive")
				}
				y, m, d := today.AddDate(0, 0, int(-numDays)).Date()
				since = time.Date(y, m, d, 0, 0, 0, 0, loc)
			}
		}

		if shouldPrintPath {
			if len(args) > 1 {
				return errors.New("We can only print one member save path at a time")
			}
			if members.RealName(args[0]) == "" {
				return errors.New("You must provide a valid member name")
			}
		}

		return
	},
	Run: func(cmd *cobra.Command, args []string) {
		if shouldPrintPath {
			name := members.RealName(args[0])
			at := since.Format("2006-01-02")
			path := filepath.Join(options.Get("save_to"), name, at)
			fmt.Printf("%s", path)
			return
		}

		// unique args
		uniqueArgs := map[string]bool{}
		for _, a := range args {
			if a == "all" {
				for m := range members.Blogs {
					uniqueArgs[m] = true
				}
				break
			}
			addArg := members.RealName(a)
			if _, ok := members.Blogs[addArg]; !ok {
				fmt.Printf("We do not know who %q is.\n", a)
				return
			}
			uniqueArgs[addArg] = true
		}

		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if shouldDryRun {
			ctx = context.WithValue(ctx, scrape.ShouldDryRun{}, struct{}{})
		}

		if verbose {
			chrome.Log = log.Printf
		}

		chrome.UserAgent = options.Get("user_agent")

		// start chrome
		if err := chrome.Start(ctx, userProfileDir, port); err != nil {
			panic(err)
		}

		// chrome.DumpProtocol()

		// wait for chrome to close
		defer func() {
			chrome.Wait()
		}()

		if saveTo != "" {
			scrape.SaveTo = saveTo
		}

		var wg sync.WaitGroup

		// if --on is used --since is ignored
		if saveBlogsOn != "" {
			fmt.Printf("Saving blogs posted on %s\n", since.Format("2006-01-02"))
			wg.Add(1)
			go func() {
				defer wg.Done()
				scrape.SaveBlogsOn(ctx, uniqueArgs, since, maxSaved)
			}()
		} else {
			// save blogs since
			for member := range uniqueArgs {
				wg.Add(1)
				go func(m string) {
					defer wg.Done()
					link := members.BlogURL(m)
					if link == "" {
						fmt.Printf("Missing blog url for %q.\n", m)
						return
					}
					fmt.Printf("Saving %s blogs since %s\n", m, since.Format("2006-01-02"))
					scrape.SaveAllBlogsSince(ctx, link, since, maxSaved)
				}(member)
			}
		}

		go func() {
			wg.Wait()
			cancel()
		}()

		// handle interrupt
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)

		for {
			select {
			case <-sig:
				signal.Stop(sig)
				cancel()
			case <-ctx.Done():
				return
			}
		}
	},
}
