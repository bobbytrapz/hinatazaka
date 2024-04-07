package blog

import (
	"net/http"
	"time"
)

const (
	WaitForBlogList      = 1 * time.Minute
	WaitForBlogDownload  = 1 * time.Minute
	WaitForSpiderTimeout = 1 * time.Minute
)

// ShouldDryRun is the context key indicating a dry run
type ShouldDryRun struct{}

// UserAgent to use when fetching
var UserAgent = `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36`

// we use for fetching pages and for remote control of chrome
var httpClient = http.Client{
	Timeout: 1 * time.Minute,
}
