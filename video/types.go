package video

import "fmt"

// ChannelMessage used to communicate between a main thread and goroutines.
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

type ErrorBadLink struct {
	link    string
	message string
}

func (e *ErrorBadLink) Error() string {
	return fmt.Sprintf("link \"%v\" has an issue: %v", e.link, e.message)
}

type ErrorGetPlaybackURL struct {
	link    string
	message string
}

func (e *ErrorGetPlaybackURL) Error() string {
	return fmt.Sprintf("Failed to get a stream URL for the link \"%v\" reason: \"%v\"", e.link, e.message)
}
