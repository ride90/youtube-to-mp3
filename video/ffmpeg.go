package video

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/gosimple/slug"
	"github.com/gosuri/uiprogress"
)

func ConvertVideoToAudio(video *Video, results chan<- ChannelMessage) {
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
	workingDir, _ := os.Getwd()
	inputPath, _ := filepath.Abs(video.File.Name())
	outputPath := path.Join(workingDir, fmt.Sprintf("%v.mp3", slug.Make(video.name)))
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
