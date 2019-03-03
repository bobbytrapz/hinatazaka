package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/bobbytrapz/hinatazaka/chrome"
	"github.com/bobbytrapz/hinatazaka/members"
	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/bobbytrapz/hinatazaka/scrape"
	"github.com/spf13/cobra"
)

var saveBlogsSince string
var saveTo string
var maxSaved int
var since time.Time

func init() {
	rootCmd.AddCommand(blogCmd)
	blogCmd.Flags().StringVar(&saveBlogsSince, "since", "", "Save any blogs newer than this date ex: 2019-03-27")
	blogCmd.Flags().IntVar(&maxSaved, "count", math.MaxInt32, "The max number of blogs to save.")
	blogCmd.Flags().StringVar(&saveTo, "saveto", "", "Directory path to save blog data to")
}

var blogCmd = &cobra.Command{
	Use:   "blog [members]",
	Short: "Save a blog as a pdf along with save each image",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return errors.New("We need at least one name/nickname of a hinatazaka member")
		}

		if saveTo != "" {
			if stat, err := os.Stat(saveTo); os.IsNotExist(err) || !stat.IsDir() {
				return errors.New("Save path must be a directory")
			}
		}

		// use tokyo time
		loc, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			panic(err)
		}

		y, m, d := time.Now().In(loc).Date()
		today := time.Date(y, m, d, 0, 0, 0, 0, loc)

		if saveBlogsSince == "" {
			// default to today's blogs
			since = today
			return nil
		}

		switch saveBlogsSince {
		case "forever":
			// save all blogs since forever
			since = time.Time{}
			return nil
		case "today":
			// same as default
			since = today
			return nil
		case "yesterday":
			since = today.AddDate(0, 0, -1)
			return nil
		case "week":
			// within this week
			weekday := today.Weekday()
			since = today.AddDate(0, 0, -int(weekday))
			return nil
		case "month":
			// within this month
			day := today.Day()
			since = today.AddDate(0, 0, -int(day)+1)
			return nil
		case "year":
			// within this year
			month := today.Month()
			day := today.Day()
			since = today.AddDate(0, -int(month)+1, -int(day)+1)
			return nil
		default:
			since, err = time.Parse("2006-01-02", saveBlogsSince)
			if err != nil {
				return err
			}
			return nil
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if verbose {
			chrome.Log = log.Printf
		}

		chrome.UserAgent = options.Get("user_agent")

		// start chrome
		if err := chrome.Start(ctx, userProfileDir, port); err != nil {
			panic(err)
		}

		// wait for chrome to close
		defer func() {
			chrome.Wait()
		}()

		// chrome.DumpProtocol()

		// unique args
		uniqueArgs := map[string]bool{}
		for _, a := range args {
			if a == "all" {
				for m := range members.Blog {
					uniqueArgs[m] = true
				}
				break
			}
			addArg := members.RealName(a)
			uniqueArgs[addArg] = true
		}

		if saveTo != "" {
			scrape.SaveTo = saveTo
		}

		var wg sync.WaitGroup
		for member := range uniqueArgs {
			wg.Add(1)
			go func(m string) {
				defer wg.Done()
				link := members.BlogURL(m)
				if link == "" {
					fmt.Printf("We do not know who %q is.\n", m)
					return
				}
				fmt.Printf("Saving %s blogs since %s\n", m, since.Format("2006-01-02"))
				scrape.SaveAllBlogsSince(ctx, link, since, maxSaved)
			}(member)
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
