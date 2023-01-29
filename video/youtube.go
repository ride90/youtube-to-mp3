package video

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

type errorBadLink struct {
	link    string
	message string
}

func (e *errorBadLink) Error() string {
	return fmt.Sprintf("link \"%v\" has an issue: \"%v\"", e.link, e.message)
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
		_url, err := url.ParseRequestURI(link)
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
			errors = append(errors, &errorBadLink{link, "not a YouTube link"})
		}
		log.Printf("hi/there?: err=%+v url=%+v\n", err, _url)
	}

	return errors
}
