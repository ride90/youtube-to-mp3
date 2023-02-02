# YouTube to mp3
Command line tool for converting YouTube videos to mp3 files.

:green_book: Created for educational purposes only (my 1st Go program).

## Usage

Provide links using `--links` parameter.
You can provide multiple links, tool will process them simultaneously.

![demo](static/demo.gif)

Run `yt2mp3 --help` for help info.

## Installation

### Get binary
You can get compiled binary or compile it by yourself.

#### 1. Get compiled binary
```shell
# Darwin amd64.
curl -L https://github.com/ride90/youtube-to-mp3/releases/download/v0.1/darwin_amd64_yt2mp3 -o yt2mp3

# Darwin arm64.
curl -L https://github.com/ride90/youtube-to-mp3/releases/download/v0.1/darwin_arm64_yt2mp3 -o yt2mp3

# Linux amd64.
curl -L https://github.com/ride90/youtube-to-mp3/releases/download/v0.1/darwin_arm64_yt2mp3 -o yt2mp3
```

#### Compile from sources
You must have a golang installed on your machine.

```shell
# Darwin amd64.
GOOS=darwin GOARCH=amd64 go build -o yt2mp3 main.go

# Darwin arm64.
GOOS=darwin GOARCH=arm64 go build -o yt2mp3 main.go

# Linux amd64.
GOOS=linux GOARCH=amd64 go build -o yt2mp3 main.go
```

You might want to add `yt2mp3` to your `PATH` or `alias` it in your `.bashrc`/`.zshrc`.
```
alias yt2mp3="/Path/to/binary/yt2mp3"
```

#### 2. Install FFmpeg

Official docs [https://ffmpeg.org/download.html](https://ffmpeg.org/download.html) (don't be scary).
Verify installation with `ffmpeg` command in your terminal emulator.

## TODO
- Solve TODOs in a codebase itself.
- Introduce `--names` parameter which will allow to search for videos using provided names
  and use 1st video (link) for every search result as a link for conversion.
  Don't use YouTube API for this since it requires API keys which is boring. We can't simply parse html either, 
  we should use something similar to headless chrome or puppeteer to execute js and only after we can get links.
  At the same time it would be great to avoid introducing big dependencies.
  Example: `yt2mp3 --names "judas priest hell patrol"`
- Download a video in parallel using different time ranges (if allowed/possible) and after glue together with ffmpeg.
