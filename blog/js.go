package blog

import "time"

// result from jsBlogs
type blogsFromPage struct {
	Pages []string `json:"pages"`
	Blogs []blog   `json:"blogs"`
}

// an individual blog
type blog struct {
	Title string     `json:"title"`
	Name  string     `json:"name"`
	Year  int        `json:"year"`
	Month time.Month `json:"month"`
	Day   int        `json:"day"`
	Link  string     `json:"link"`
}

// an image from the blog
type image struct {
	Link string `json:"link"`
	Data []byte
}

// uses Array toString() to make a comma-separated list of image urls
var jsBlogImages = `
() => {
    return [...document.querySelectorAll(":scope .p-blog-article img:not(.emoji)")].map(el => el.src).toString();
};
`

// each member has a page that lists all of their blogs
// this extracts the individual blog links from that page
// we can get the member's name and date posted with a bit more work
// we also get all the links to other pages from the pager at the bottom
// the page urls are provided to the spider
var jsBlogs = `
() => {
	return {
		pages: [...document.querySelectorAll(".c-pager__item--count a")].map(el => el.href),
		blogs: [...document.querySelectorAll(".p-blog-article")].map(el => {
			const link = el.querySelector(".p-button__blog_detail > a").href;
			const head = el.querySelector(".p-blog-article__head");
			const name = head.querySelector(".c-blog-article__name").textContent.trim();
			const title = head.querySelector(".c-blog-article__title").textContent.trim();
			const t = head.querySelector(".c-blog-article__date").textContent.trim();
			[year, month, day] = t.split(" ")[0].split(".");
			return {
			  title: title,
			  name: name,
			  year: parseInt(year),
			  month: parseInt(month),
			  day: parseInt(day),
			  link: link
			};
		})
		};
	}
`
