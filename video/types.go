package video

import (
	"fmt"
	"os"
)

// ChannelMessage used to communicate between a main thread and goroutines.
type ChannelMessage struct {
	Result *Video
	Err    error
	Link   string
}

func (msg ChannelMessage) String() string {
	if msg.Err != nil {
		return fmt.Sprintf("result=%v err=%v", msg.Result, msg.Err)
	}
	return fmt.Sprintf("Result: %#v", msg.Result)
}

type ErrorBadLink struct {
	link    string
	message string
}

func (e *ErrorBadLink) Error() string {
	return fmt.Sprintf("Link %q %v", e.link, e.message)
}

type ErrorFetchPlaybackURL struct {
	link    string
	message string
}

func (e *ErrorFetchPlaybackURL) Error() string {
	return fmt.Sprintf("Failed to get a stream URL for the link %q reason: %q", e.link, e.message)
}

type Video struct {
	url       string
	streamUrl string
	name      string
	File      *os.File
}

func (v Video) String() string {
	return fmt.Sprintf(
		"<name=%q url=%q hasStream=%v> file=%v", v.name, v.url, v.HasStreamURL(), v.File,
	)
}

func (v Video) HasStreamURL() bool {
	return v.streamUrl != ""
}
