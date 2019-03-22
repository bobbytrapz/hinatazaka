package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"sync"

	"github.com/bobbytrapz/hinatazaka/chrome"
	"github.com/bobbytrapz/hinatazaka/members"
	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/bobbytrapz/hinatazaka/scrape"
	"github.com/spf13/cobra"
)

var keywords []string

func init() {
	rootCmd.AddCommand(auctionCmd)
	auctionCmd.Flags().StringArrayVarP(&keywords, "keywords", "k", []string{"生写真"}, "Search using these keywords")
}

type res struct {
	Name string
	MOV  float32
}
type byMOV []res

func (s byMOV) Len() int {
	return len(s)
}

func (s byMOV) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func (s byMOV) Less(a, b int) bool {
	return s[a].MOV < s[b].MOV
}

var auctionCmd = &cobra.Command{
	Use:   "auction [names]",
	Short: "Estimate auction value of goods from Yahoo Auctions given keywords.",
	Long: `Estimate auction value of goods from Yahoo Auctions given keywords.
We use the median order value of the top winning bids.
`,
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return errors.New("We need at least one name")
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

		chrome.UserAgent = options.Get("user_agent")

		// start chrome
		if err := chrome.Start(ctx, userProfileDir, port); err != nil {
			panic(err)
		}

		// wait for chrome to close
		defer func() {
			chrome.Wait()
		}()

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
			uniqueArgs[addArg] = true
		}

		var wg sync.WaitGroup

		var results []res
		for m := range uniqueArgs {
			wg.Add(1)
			go func(member string) {
				defer wg.Done()
				mov, err := scrape.BidsMedianOrderValue(ctx, member, keywords)
				if err == nil {
					results = append(results, res{member, mov})
				} else {
					panic(err)
				}
			}(m)
		}

		defer func() {
			sort.Sort(sort.Reverse(byMOV(results)))
			for _, r := range results {
				fmt.Printf("[%s] %.2f\n", r.Name, r.MOV)
			}
		}()

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
