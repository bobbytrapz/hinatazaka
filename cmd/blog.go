package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/bobbytrapz/hinatazaka/chrome"
	"github.com/bobbytrapz/hinatazaka/members"
	"github.com/spf13/cobra"
)

var saveBlogsSince string
var since time.Time

func init() {
	rootCmd.AddCommand(blogCmd)
	blogCmd.Flags().StringVar(&saveBlogsSince, "since", "", "Save any blogs newer than this date ex: 2019-03-27")
}

var blogCmd = &cobra.Command{
	Use:   "blog [members]",
	Short: "Save a blog as a pdf along with save each image",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return errors.New("We need at least one name/nickname of a hinatazaka member")
		}

		if saveBlogsSince == "" {
			// default to today's blogs
			since = time.Now()
			return nil
		}

		if saveBlogsSince == "forever" {
			// save all blogs since forever
			since = time.Time{}
			return nil
		}

		since, err = time.Parse("2006-01-02", saveBlogsSince)
		if err != nil {
			return err
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if verbose {
			chrome.Log = log.Printf
		}

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
				chrome.SaveAllBlogsSince(ctx, link, since)
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
