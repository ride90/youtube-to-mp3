package video

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/gosimple/slug"
	"github.com/gosuri/uiprogress"
)

// ConvertVideoToAudio converts video to mp3 and saves in dstDir.
func ConvertVideoToAudio(video *Video, dstDir string, results chan<- ChannelMessage) {
	// Get bar with steps.
	var steps = []string{"Converting to audio", "DONE!"}
	bar := uiprogress.AddBar(len(steps))
	bar.Width = progressBarWidth
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("%v - %v", (*video).url, steps[b.Current()-1])
	})
	bar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("[Convert]    ")
	})

	// Prepare ffmpeg cmd.
	bar.Incr()
	time.Sleep(time.Millisecond * 50)
	inputPath, _ := filepath.Abs(video.File.Name())
	outputPath := path.Join(dstDir, fmt.Sprintf("%v.mp3", slug.Make(video.name)))
	// TODO: Think if it's possible (it is possible) to tune a command.
	cmd := exec.Command(
		"ffmpeg",
		"-i",
		inputPath,
		outputPath,
	)

	// Run ffmpeg.
	if err := cmd.Run(); err != nil {
		results <- ChannelMessage{Err: err}
		return
	}
	(*video).AudioFilePath = outputPath
	bar.Incr()
	time.Sleep(time.Millisecond * 50)
	results <- ChannelMessage{Result: video}
}
