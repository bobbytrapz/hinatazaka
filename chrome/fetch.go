package chrome

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// UserAgent to use when fetching
var UserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36`

// we use for fetching pages and for remote control of chrome
var httpClient = http.Client{
	Timeout: 60 * time.Second,
}

func newRequest(ctx context.Context, host string, method string, url string) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, nil)
	if err != nil {
		err = fmt.Errorf("fetch.newRequest")
		return
	}

	// headers
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("DNT", "1")
	req.Header.Add("Host", host)
	req.Header.Add("Pragma", "no-cache")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("User-Agent", UserAgent)

	req = req.WithContext(ctx)

	return
}

// Fetch a page
func Fetch(ctx context.Context, link string) (*http.Response, error) {
	u, err := url.ParseRequestURI(link)
	if err != nil {
		panic("fetch.Get: invalid url" + link)
	}
	req, err := newRequest(ctx, u.Host, "GET", link)
	if err != nil {
		return nil, fmt.Errorf("fetch.Get: %s", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch.Get: %s", err)
	}
	return res, nil
}
