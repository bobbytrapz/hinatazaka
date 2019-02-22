package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"

	"github.com/bobbytrapz/hinatazaka/chrome"
	"github.com/spf13/cobra"
)

var saveWebImagesTo string

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.Flags().StringVar(&saveWebImagesTo, "saveto", "./", "Save images to the given path")
}

var webCmd = &cobra.Command{
	Use:   "web [urls]",
	Short: "Save all images from a url with supported hostnames",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if len(args) < 1 {
			return errors.New("We need a website to gather images from")
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
		if err := chrome.Start(ctx); err != nil {
			panic(err)
		}

		// wait for chrome to close
		defer func() {
			chrome.Wait()
		}()

		var wg sync.WaitGroup
		for _, link := range args {
			u, err := url.ParseRequestURI(link)
			if err != nil {
				fmt.Println("Parse error:", err)
				return
			}
			switch u.Host {
			case "hustlepress.co.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					chrome.SaveImagesFrom(ctx, link, saveWebImagesTo, ".size-full")
				}(link)
			default:
				fmt.Println("We cannot handle:", link)
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
