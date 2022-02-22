package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/bobbytrapz/gochrome"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"

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

		browser := gochrome.NewBrowser()
		browser.UserAgent = options.Get("user_agent")

		_, err = browser.Start(ctx, gochrome.TemporaryUserProfileDirectory, port)
		if err != nil {
			panic(err)
		}
		defer browser.Wait()

		if verbose {
			gochrome.Log = log.Printf
		}

		var wg sync.WaitGroup
		for _, u := range urls {
			switch u.Host {
			case "hustlepress.co.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('img.size-full')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "ray-web.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.scale_full > a > img,.top_photo > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
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
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
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
					jsCode := `[...document.querySelectorAll('img.c-image__image, .pg-photo__webImageListLink > img')].map(el => {
							link = el.src;
							return link.slice(0, link.indexOf('?'));
						}).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
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
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "taishu.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					if !strings.Contains(l, "photo") {
						fmt.Println("We need https://taishu.jp/articles/photo/{num}")
						return
					}
					jsCode := `[...document.querySelectorAll('.swiper-slide > figure > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "cancam.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('a')].filter(el => el.href.includes('.jpg')).map(el => el.href).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "jj-jj.net":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('img')].filter(i => i.width >= 600).map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "news.dwango.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.stop-tap > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "news.mynavi.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.photo_table__link')].map(el => el.href.replace('/photo', '')).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "lineblog.me":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('img.pict')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "nonno.hpplus.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelectorAll('.article > .part .image figure > div > img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
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
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "bltweb.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('.mh-content').querySelectorAll('img')]
						.filter(i => i.width >= 300)
						.map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "image.itmedia.co.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('#imgThumb_in').querySelectorAll('a')]
						.map(el => el.href.replace('/l/im', '')).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "ar-mag.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('.posts__contents').querySelectorAll('img')]
						.filter(i => i.width >= 300)
						.map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
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
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "news.line.me":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('section').querySelectorAll('img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "girlswalker.com":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					jsCode := `[...document.querySelector('.gw-content__entry-article').querySelectorAll('img')].map(el => el.src).toString()`
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
				}(u.String())
			case "thetv.jp":
				wg.Add(1)
				go func(l string) {
					defer wg.Done()
					base := path.Base(u.Path)
					jsCode := fmt.Sprintf(`[...document.querySelector('.galleryArea').querySelectorAll('a')]
							.map(el => new URL(el.href).pathname.split('/').slice(-2)[0])
							.map(name => 'https://thetv.jp/i/nw/%s/' + name + '.jpg')
							.toString()`, base)
					fmt.Printf("Saving all images from %s to %s\n", l, saveWebImagesTo)
					scrape.SaveImagesFrom(ctx, browser, l, saveWebImagesTo, jsCode)
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
