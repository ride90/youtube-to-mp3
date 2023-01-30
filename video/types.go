package video

import "fmt"

// ChannelMessage used to communicate between a main thread and goroutines.
type ChannelMessage struct {
	result Video
	err    error
	link   string
}

func (msg ChannelMessage) String() string {
	if msg.err != nil {
		return fmt.Sprintf("result=%v err=%v", msg.result, msg.err)
	}
	return fmt.Sprintf("Result: %#v", msg.result)
}

type ErrorBadLink struct {
	link    string
	message string
}

func (e *ErrorBadLink) Error() string {
	return fmt.Sprintf("Link %q %v", e.link, e.message)
}

type ErrorGetPlaybackURL struct {
	link    string
	message string
}

func (e *ErrorGetPlaybackURL) Error() string {
	return fmt.Sprintf("Failed to get a stream URL for the link %q reason: %q", e.link, e.message)
}

type Video struct {
	url       string
	streamUrl string
	name      string
}

func (v Video) String() string {
	hasStream := v.streamUrl != ""
	return fmt.Sprintf("name=%q url=%q hasStream=%v", v.name, v.url, hasStream)
}
