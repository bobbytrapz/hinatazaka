A collection of hinatazaka command-line tools. Requires Google Chrome.

This is just a toy project I'm using to check out remote controlling Chrome DevTools.

Currently not well tested but I'm using it.

There is no support for Windows but there probably will be eventually.

## Install

No releases yet. Use 'go get'.

```
go get github.com/bobbytrapz/hinatazaka
```

## Usage

hinatazaka \[item\] \[member names\] \[flags\]

Download Kyoko's blogs since today:

```
hinatazaka blog kyoko
```

Download Kyoko and Kageyama's blogs since Independence Day:

```
hinatazaka blog kyoko kagechan --since 2019-02-11
```

Download two of Kyoko's blogs from this week:

```
hinatazaka blog kyoko --since week --count 2
```

Archive Iguchi's blog:

```
hinatazaka blog iguchi --since forever
```

Blog and images to \$HOME/hinatazaka by default. You can change this in the options ~/.config/hinatazaka/options.toml

The blog is stored in an archive in mhtml format and all the images are saved in a directory according to the date the blog was posted. You can open mhtml files with Google Chrome.

Items supported so far:

- blog: archives the blog and saves image
- web: scrape images from any one of the supported websites

## Scripts

Some [python scripts](./scripts) for data collection are also included

```
youtube-comments.py {video_url}
```

## Showroom

If you want to download Showroom streams checkout my other project: [autosr](https://github.com/bobbytrapz/autosr)

## Other Groups

I have no interest in supporting other groups. You are welcome to fork this project and add whomever you like.
