package video

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/kkdai/youtube/v2"
)

const audioQualityMedium string = "AUDIO_QUALITY_MEDIUM"
const videoQualityHigh string = "hd720"
const videoQualityMedium string = "medium"
const videoQualityTiny string = "tiny"
const prefixLong string = "https://www.youtube.com/"
const prefixShort string = "https://youtu.be/"

// ValidateLinks Ensures:
//   - links are valid parseable URLs
//   - links are YouTube links
func ValidateLinks(links []string) []error {
	var _errors []error

	// Get bar and do initial increment (seems like a bug).
	bar := uiprogress.AddBar(len(links))
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("[Check links]")
	})
	bar.Incr()

	// Validate links. If at least one link is not valid we stop an execution.
	for _, link := range links {
		// Check if link is parseable.
		_, err := url.ParseRequestURI(link)
		if err != nil {
			_errors = append(
				_errors,
				&ErrorBadLink{link, fmt.Sprintf("%v", err)},
			)
		}
		// Check if link is a YouTube link.
		// Valid links:
		// 	- https://youtu.be/<video_id>
		// 	- https://www.youtube.com/watch?v=<video_id>
		// Query arguments are ignored so far.
		// TODO: In the future query args can be used to cut video.
		// 	Might be handy if you want to extract audio using a specific range.
		if !strings.HasPrefix(link, prefixShort) && !strings.HasPrefix(link, prefixLong) {
			_errors = append(
				_errors,
				&ErrorBadLink{
					link,
					fmt.Sprintf(
						"not a YouTube video link. Expected formats: \"%v/<video_id>\" or \"%v?v=<video_id>\"",
						prefixLong, prefixShort,
					),
				},
			)
		}
		// Artificial micro delay wouldn't hurt here (to avoid steps not being rendered in time).
		bar.Incr()
		time.Sleep(time.Millisecond * 50)
	}
	return _errors
}

func FetchPlaybackURL(link string, results chan<- ChannelMessage) {
	var video Video

	// Get bar with steps.
	var steps = []string{"cleaning up links..", "getting direct stream..", "got a stream!"}
	bar := uiprogress.AddBar(len(steps))
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%v - %v", link, steps[b.Current()-1])
	})
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("[Get stream] ")
	})

	// Clean up links.
	//  - remove all query params except `v`
	bar.Incr()
	time.Sleep(time.Millisecond * 50)
	_url, _ := url.ParseRequestURI(link)
	query, _ := url.ParseQuery(_url.RawQuery)
	for k, _ := range query {
		if k != "v" {
			query.Del(k)
		}
	}
	_url.RawQuery = query.Encode()
	video.url = _url.String()

	// Make a request to YouTube.
	// TODO: research if http session can be used here (shared between goroutines?).
	bar.Incr()
	time.Sleep(time.Millisecond * 50)
	resp, err := http.Get(video.url)
	defer resp.Body.Close()
	if err != nil {
		results <- ChannelMessage{Err: err}
		return
	} else if resp.StatusCode != http.StatusOK {
		results <- ChannelMessage{Err: errors.New(resp.Status), Link: video.url}
		return
	}

	// Read body and convert to string.
	_body, err := io.ReadAll(resp.Body)
	if err != nil {
		results <- ChannelMessage{Err: err, Link: video.url}
	}
	var body = string(_body)

	// Compile regex and try to find a necessary JS var (json) with a streaming URL.
	// Compiling regex multiple times (multiple jobs) might be suboptimal.
	regex, _ := regexp.Compile("ytInitialPlayerResponse\\s*=\\s*(\\{.+?\\})\\s*;")
	// `map[string]any` is used due to we don't know a data structure in advance (well, almost).
	var playerResponseData map[string]any
	var jsonString string
	if idx := regex.FindStringIndex(body); len(idx) != 0 {
		jsonString = body[idx[0]+len("ytInitialPlayerResponse = ") : idx[1]-1]
		// String -> JSON.
		err = json.Unmarshal(
			[]byte(jsonString),
			&playerResponseData,
		)
		if err != nil {
			results <- ChannelMessage{Err: err, Link: video.url}
			return
		}
		// At his point we may or may not find a stream URl in next places:
		// 	- ytInitialPlayerResponse.streamingData.adaptiveFormats (TODO: implement me)
		// 	- ytInitialPlayerResponse.streamingData.formats
		// If `url` key exists then we can use it, but there is a high chance that it's not there,
		// in this case it means that this video is extra protected, for such cases we need another approach for getting
		// a direct stream link...
		// Another approach (see https://tyrrrz.me/blog/reverse-engineering-youtube):
		//  - Download video's embed page (e.g. https://www.youtube.com/embed/<videoID>).
		//  - Extract player source URL (e.g. https://www.youtube.com/yts/jsbin/player-vflYXLM5n/en_US/base.js).
		//  - Get the value of sts (e.g. 17488).
		//  - Download and parse player source code.
		//  - Request video metadata (e.g. https://www.youtube.com/get_video_info?video_id=e_<videoID>&sts=17488&hl=en).
		//    Try with el=detailpage if it fails.
		//  - Parse the URL-encoded metadata and extract information about streams.
		//  - If they have signatures, use the player source to decipher them and update the URLs.
		//  - If there's a reference to DASH manifest, extract the URL and decipher it if necessary as well.
		//  - Download the DASH manifest and extract additional streams.
		//  - Use itag to classify streams by their properties.
		//  - Choose a stream and download it in segments.(see another further in the code).
		//  To handle this case youtube-dl lib will be used (see further in the code).
		// TODO: Handle interface conversion: interface {} is nil, not map[string]interface {}.
		// 	Appears when youtube link provided but id doesn't exist.
		formats := playerResponseData["streamingData"].(map[string]any)["formats"].([]any)
		for _, v := range formats {
			// Ensure we have all keys we need.
			var hasQuality bool = v.(map[string]any)["quality"] != nil
			var hasURL bool = v.(map[string]any)["url"] != nil
			var hasAudioQuality bool = v.(map[string]any)["audioQuality"] != nil
			var hasMimeType bool = v.(map[string]any)["mimeType"] != nil
			if !hasQuality || !hasURL || !hasAudioQuality || hasMimeType {
				continue
			}
			// Try to get strings from underlying value.
			quality, okQuality := v.(map[string]any)["quality"].(string)
			audioQuality, okAudioQuality := v.(map[string]any)["audioQuality"].(string)
			mimeType, okMimeType := v.(map[string]any)["mimeType"].(string)
			streamURL, okStreamURL := v.(map[string]any)["url"].(string)
			if !okQuality || !okAudioQuality || !okStreamURL || okMimeType {
				continue
			}
			// Get only tiny/medium/hd with a medium audio quality.
			// No need for an explicit sorting (at least for now) since tiny goes first in the response array.
			isQuality := quality == videoQualityTiny || quality == videoQualityMedium || quality == videoQualityHigh
			if isQuality && audioQuality == audioQualityMedium {
				video.streamUrl, video.mimeType = streamURL, mimeType
				break
			}
		}
		// At this point if we have a stream URL we are fine, and we can use it.
		// If not, we'll use YouTube dl lib to get the stream url of protected videos/channels.
		if video.HasStreamURL() {
			bar.Incr()
			time.Sleep(time.Millisecond * 50)
			results <- ChannelMessage{
				Result: &video,
				Link:   video.url,
			}
			return
		}
	}

	// Try to get video stream using a youtube-dl library (ported from python).
	client := youtube.Client{}
	dlvideo, err := client.GetVideo(link)
	formats := dlvideo.Formats.WithAudioChannels()
	// Loop through formats until we find the one which fits our needs: lightest video (if possible), medium audio.
	for _, format := range formats {
		isQuality := format.Quality == videoQualityTiny || format.Quality == videoQualityMedium || format.Quality == videoQualityHigh
		if isQuality && format.AudioQuality == audioQualityMedium {
			// Magic happens here and we get our video stream URL.
			streamURL, err := client.GetStreamURL(dlvideo, &format)
			if err != nil {
				results <- ChannelMessage{Err: err, Link: link}
				return
			}
			video.streamUrl, video.mimeType = streamURL, format.MimeType
			bar.Incr()
			time.Sleep(time.Millisecond * 50)
			results <- ChannelMessage{
				Result: &video,
				Link:   link,
			}
			return
		}
	}

	// Desired video stream was not found.
	results <- ChannelMessage{
		Link: link,
		Err:  &ErrorFetchPlaybackURL{link: link, message: "desired video stream was not found"},
	}
}

func FetchMetadata(video *Video, results chan<- ChannelMessage) {
	// Get bar with steps.
	var steps = []string{"getting metadata..", "got metadata!"}
	bar := uiprogress.AddBar(len(steps))
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%v - %v", (*video).url, steps[b.Current()-1])
	})
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("[Metadata]   ")
	})

	// Get youtube dl client and fetch metadata for the video.
	bar.Incr()
	time.Sleep(time.Millisecond * 50)
	client := youtube.Client{}
	// TODO: Add error handling.
	videoMeta, _ := client.GetVideo((*video).url)
	(*video).name = videoMeta.Title

	bar.Incr()
	time.Sleep(time.Millisecond * 50)
	results <- ChannelMessage{Result: video}
}

func FetchVideo(video *Video, results chan<- ChannelMessage) {
	// Create tmp file.
	// TODO: Add error handling.
	file, _ := os.CreateTemp("", "yt2mp3_*_.mp4")
	defer file.Close()
	(*video).File = file

	// Send http request, check status, read file.
	// TODO: Add error handling.
	resp, _ := http.Get((*video).streamUrl)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// TODO: Add error handling.
	}

	// Add progress bar to track fetching progress.
	time.Sleep(time.Millisecond * 50)
	bar := uiprogress.AddBar(int(resp.ContentLength) + 1)
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%v bytes - %q", b.Current(), (*video).name)
	})
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("[Download]   ")
	})

	// Download video using a custom io reader.
	pbreader := &PBReader{Reader: resp.Body, bar: bar}
	// TODO: Add error handling.
	_, _ = io.Copy(file, pbreader)
	time.Sleep(time.Millisecond * 50)

	results <- ChannelMessage{Result: video}
}
