package video

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type ChannelMessage struct {
	link   string
	result string
	err    error
}

func (msg ChannelMessage) String() string {
	if msg.result != "" {
		return fmt.Sprintf("Link: %v Result: %v Error: %v", msg.link, msg.result, msg.err)
	}
	return fmt.Sprintf("Link: %v Error: %v", msg.link, msg.err)
}

type errorBadLink struct {
	link    string
	message string
}

func (e *errorBadLink) Error() string {
	return fmt.Sprintf("link \"%v\" has an issue: %v", e.link, e.message)
}

type errorGetPlaybackURL struct {
	link    string
	message string
}

func (e *errorGetPlaybackURL) Error() string {
	return fmt.Sprintf("Failed to get a stream URL for link \"%v\" with an issue: \"%v\"", e.link, e.message)
}

// ValidateLinks Ensures:
// - links are valid parseable URLs
// - links are YouTube links
func ValidateLinks(links []string) []error {
	const prefixLong string = "https://www.youtube.com/"
	const prefixShort string = "https://youtu.be/"
	var errors []error

	// Validate links. If at least one link is not valid we stop an execution.
	for _, link := range links {
		// Check if link is parseable.
		_, err := url.ParseRequestURI(link)
		if err != nil {
			errors = append(
				errors,
				&errorBadLink{link, fmt.Sprintf("%v", err)},
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
			errors = append(
				errors,
				&errorBadLink{
					link,
					fmt.Sprintf(
						"not a YouTube video link. Expected formats: \"%v/<video_id>\" or \"%v?v=<video_id>\"",
						prefixLong, prefixShort,
					),
				},
			)
		}
	}

	return errors
}

func GetPlaybackURL(link string, results chan<- ChannelMessage) {
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
		results <- ChannelMessage{err: err, link: link}
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
		// At his point we may or may not find a stream URl in the next places:
		// 	- ytInitialPlayerResponse.streamingData.adaptiveFormats (skip it for now)
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
		// To handle this case youtube dl lib will be used (see further in the code).
		var streamURL string
		formats := playerResponseData["streamingData"].(map[string]any)["formats"].([]any)
		for _, v := range formats {
			// Ensure we have all keys we need.
			var hasQualityLabel bool = v.(map[string]any)["qualityLabel"] != nil
			var hasURL bool = v.(map[string]any)["url"] != nil
			var hasMimeType bool = v.(map[string]any)["mimeType"] != nil
			if !hasQualityLabel || !hasURL || !hasMimeType {
				continue
			}
			// Try to get strings from underlying value.
			quality, okQuality := v.(map[string]any)["qualityLabel"].(string)
			mimetype, okMimetype := v.(map[string]any)["mimeType"].(string)
			_streamURL, okStreamURL := v.(map[string]any)["url"].(string)
			if !okQuality || !okMimetype || !okStreamURL {
				continue
			}
			if (quality == "360p" || quality == "240p") && strings.HasPrefix(mimetype, "video/mp4") {
				streamURL = _streamURL
				break
			}
		}
		// At this point if we have a stream URL we are fine, and we can use it.
		// If not, we'll use youtube dl lib to get the stream url of protected videos/channels.
		if streamURL != "" {
			results <- ChannelMessage{result: streamURL, link: link}
			return
		}
	}
	results <- ChannelMessage{link: link}
}

func DownloadVideo(playbackLink string) {

}

func download() {
	link := "https://www.youtube.com/watch?v=hS5CfP8n_js"
	//id := "hS5CfP8n_js"

	// Get metadata.
	log.Printf("Making a request.. \"%v\"", link)
	//metaURL := "https://www.youtube.com/get_video_info?video_id=" + id
	resp, err := http.Get(link)
	defer resp.Body.Close()

	if err != nil {
		log.Printf("Error %s", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("All bad. Status %s", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	var bodyText = string(body)

	//r, _ := regexp.Compile("var ytInitialPlayerResponse")
	//fmt.Println(
	//	r.FindStringIndex(string(body)),
	//)
	r, _ := regexp.Compile("ytInitialPlayerResponse\\s*=\\s*(\\{.+?\\})\\s*;")

	var playerResponseData map[string]any
	var jsonString string

	if idx := r.FindStringIndex(bodyText); len(idx) != 0 {
		jsonString = bodyText[idx[0]+len("ytInitialPlayerResponse = ") : idx[1]-1]
		//fmt.Printf("%v", jsonString)
		err = json.Unmarshal(
			[]byte(jsonString),
			&playerResponseData,
		)
		if err != nil {
			log.Printf("Error %s", err)
			return
		}
	}

	formats := playerResponseData["streamingData"].(map[string]any)["formats"].([]any)
	//fmt.Printf("%T %v\n\n", formats, formats)
	for _, v := range formats {
		fmt.Printf("%v\n\n", v.(map[string]any)["url"])
		fmt.Printf("%v\n\n", v.(map[string]any)["qualityLabel"])
	}

}
