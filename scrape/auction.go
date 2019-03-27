package scrape

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bobbytrapz/chrome"
)

const jsPricesMOV = `
(() => {
	const prices = [...document.querySelectorAll('.ePrice')].map(el => {
		return parseInt(el.innerText.replace(',', ''));
	}).sort();
	const mid = Math.floor(prices.length / 2);
	return prices.length % 2 == 0 ? (prices[mid-1] + prices[mid]) / 2 : prices[mid];
})()
`

const resPricesMOV = `
`

// MedianClosingBidValue the given idol using the given keywords
// by calculating the median closing bid value of the most recent bids
func MedianClosingBidValue(ctx context.Context, name string, keywords []string) (median float32, err error) {
	tab, err := chrome.ConnectToNewTab(ctx)
	if err != nil {
		err = fmt.Errorf("scrape.MedianClosingBidValue: %s", err)
		return
	}
	defer tab.PageClose()

	endpoint := `https://auctions.yahoo.co.jp/closedsearch/closedsearch?p=%s&va=%s&b=1&n=%d&select=2&slider=undefined`
	keywords = append(keywords, name)
	params := strings.Join(keywords, "+")

	perPage := 100
	p := params
	va := params
	link := fmt.Sprintf(endpoint, p, va, perPage)
	fmt.Println("[link]", link)

	tab.PageNavigate(link)
	// note: it takes a bit to load up since there are more items on a page
	// so we wait a bit longer
	tab.WaitForLoad(20 * time.Second)

	tab.Command("Runtime.evaluate", chrome.TabParams{
		"expression": jsPricesMOV,
	})

	// wait for reponse
	data := tab.Wait()
	var res ResJSFloat32
	err = json.Unmarshal(data, &res)
	if err != nil {
		chrome.Log("scrape.MedianClosingBidValue: %s", err)
	}
	chrome.Log("scrape.MedianClosingBidValue: res: %+v", res)
	median = res.Result.Value

	return
}
