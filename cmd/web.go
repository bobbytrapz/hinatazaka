package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/bobbytrapz/chrome"
	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/bobbytrapz/hinatazaka/scrape"
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
		// create save directory if it does not exist
		err := os.MkdirAll(saveWebImagesTo, os.ModePerm)
		if err != nil {
			panic(err)
		}

		// parse given links
		var urls []*url.URL
		for _, link := range args {
			u, err := url.ParseRequestURI(link)
			if err != nil {
				fmt.Println("Parse error:", err)
				return
			}
			urls = append(urls, u)
		}

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

		var wg sync.WaitGroup
		for _, u := range urls {
			switch u.Host {
			case "hustlepress.co.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('img.size-full')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "ray-web.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.scale_full > a > img,.top_photo > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "bisweb.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[
							...document.querySelector(".tieup_wrap").querySelectorAll("p > img")
						].map(el => el.src)
						.concat([document.querySelector(".single_kv").style.backgroundImage.slice(5, -2)])
						.toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "mdpr.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					if !strings.Contains(l, "photo") {
						// todo: maybe add support for news page
						fmt.Println("We need https://mdpr.jp/photo/detail/{num}")
						return
					}
					jsCode := `[...document.querySelectorAll('figure.square > a > img')].map(el => {
							link = el.src;
							return link.slice(0, link.indexOf('?'));
						}).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "tokyopopline.com":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					var jsCode string
					if strings.Contains(l, "archives") {
						jsCode = `[...document.querySelector('main').querySelectorAll(".gallery-icon > a")].map(el => el.href).toString()`
					} else {
						jsCode = `[document.querySelector('main').querySelector(".entry-thumbnail > img"), ...document.querySelectorAll('.gallery-icon > a')].map(el => el.src || el.href).toString()`
					}
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "taishu.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					if !strings.Contains(l, "photo") {
						fmt.Println("We need https://taishu.jp/photo/{num}")
						return
					}
					jsCode := `[...document.querySelectorAll('.swiper-slide > figure > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "cancam.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('a')].filter(el => el.href.includes('.jpg')).map(el => el.href).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "jj-jj.net":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('img')].filter(i => i.width >= 600).map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "news.dwango.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.stop-tap > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "news.mynavi.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.photo_table__link')].map(el => el.href.replace('/photo', '')).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "lineblog.me":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('img.pict')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "nonno.hpplus.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.article > .part .image figure > div > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "abematimes.com":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('.blog-article__content').querySelectorAll('.img__item > img')].map(el => {
							link = el.src;
							return link.includes('?') ? link.slice(0, link.indexOf('?')) : link;
						}).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "bltweb.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('.mh-content').querySelectorAll('img')]
						.filter(i => i.width >= 300)
						.map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "image.itmedia.co.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('#imgThumb_in').querySelectorAll('a')]
						.map(el => el.href.replace('/l/im', '')).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "ar-mag.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('.posts__contents').querySelectorAll('img')]
						.filter(i => i.width >= 300)
						.map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "www.nikkansports.com":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.article-main img[style]')]
						.map(el => el.style.backgroundImage.slice(5, -2))
						.map(url => url.replace('w200', 'w1300'))
						.map(url => url.replace('w500', 'w1300'))
						.toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "news.line.me":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('section').querySelectorAll('img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, l, saveWebImagesTo, jsCode)
				}(u.String())
			default:
				fmt.Println("We cannot handle:", u.String())
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
