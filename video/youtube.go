package video

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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
	}
	return _errors
}

func GetPlaybackURL(link string, results chan<- ChannelMessage) {
	var streamURL string

	// Remove all query params except `v`.
	_url, _ := url.ParseRequestURI(link)
	query, _ := url.ParseQuery(_url.RawQuery)
	for k, _ := range query {
		if k != "v" {
			query.Del(k)
		}
	}
	_url.RawQuery = query.Encode()
	link = _url.String()
	// Make a request to YouTube.
	resp, err := http.Get(link)
	defer resp.Body.Close()
	if err != nil {
		results <- ChannelMessage{err: err}
		return
	} else if resp.StatusCode != http.StatusOK {
		results <- ChannelMessage{err: errors.New(resp.Status), link: link}
		return
	}
	// Read body and convert to string.
	_body, err := io.ReadAll(resp.Body)
	if err != nil {
		results <- ChannelMessage{err: err, link: link}
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
			results <- ChannelMessage{err: err, link: link}
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
		formats := playerResponseData["streamingData"].(map[string]any)["formats"].([]any)
		for _, v := range formats {
			// Ensure we have all keys we need.
			var hasQuality bool = v.(map[string]any)["quality"] != nil
			var hasURL bool = v.(map[string]any)["url"] != nil
			var hasAudioQuality bool = v.(map[string]any)["audioQuality"] != nil
			if !hasQuality || !hasURL || !hasAudioQuality {
				continue
			}
			// Try to get strings from underlying value.
			quality, okQuality := v.(map[string]any)["quality"].(string)
			audioQuality, okAudioQuality := v.(map[string]any)["audioQuality"].(string)
			_streamURL, okStreamURL := v.(map[string]any)["url"].(string)
			if !okQuality || !okAudioQuality || !okStreamURL {
				continue
			}
			// Get only tiny/medium/hd with a medium audio quality.
			// No need for an explicit sorting (at least for now) since tiny goes first in the response array.
			isQuality := quality == videoQualityTiny || quality == videoQualityMedium || quality == videoQualityHigh
			if isQuality && audioQuality == audioQualityMedium {
				streamURL = _streamURL
				break
			}
		}
		// At this point if we have a stream URL we are fine, and we can use it.
		// If not, we'll use YouTube dl lib to get the stream url of protected videos/channels.
		if streamURL != "" {
			results <- ChannelMessage{
				result: Video{streamUrl: streamURL, url: link},
				link:   link,
			}
			return
		}
	}
	// Try to get video stream using a youtube-dl library (ported from python).
	client := youtube.Client{}
	video, err := client.GetVideo(link)
	formats := video.Formats.WithAudioChannels()
	// Loop through formats until we find the one which fits our needs: lightest video (if possible), medium audio.
	for _, format := range formats {
		isQuality := format.Quality == videoQualityTiny || format.Quality == videoQualityMedium || format.Quality == videoQualityHigh
		if isQuality && format.AudioQuality == audioQualityMedium {
			// Magic happens here and we get our video stream URL.
			_streamURL, err := client.GetStreamURL(video, &format)
			if err != nil {
				results <- ChannelMessage{err: err, link: link}
				return
			}
			streamURL = _streamURL
			results <- ChannelMessage{
				result: Video{streamUrl: streamURL, url: link},
				link:   link,
			}
			return
		}
	}
	results <- ChannelMessage{
		link: link,
		err:  &ErrorGetPlaybackURL{link: link, message: "desired video stream was not found"},
	}
}

func FetchMetadata(link string) {

}

func DownloadVideo(streamURL string) {

}
