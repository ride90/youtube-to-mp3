package video

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// TODO: Think if it's possible (it is possible) to tune a command.
//
//	It might be interesting to introduce an audio quality level.
const ffmpegCommandTemplate string = "ffmpeg -i %q %q"

func ConvertVideoToAudio(video *Video, results chan<- ChannelMessage) {
	workingDir, _ := os.Getwd()
	inputPath, _ := filepath.Abs(filepath.Dir(video.File.Name()))
	outputPath := path.Join(workingDir, fmt.Sprintf("%v.mp3", video.name))
	cmd := exec.Command(
		fmt.Sprintf(
			ffmpegCommandTemplate,
			inputPath,
			outputPath,
		),
	)

	if err := cmd.Run(); err != nil {
		results <- ChannelMessage{Err: err}
		return
	}
	(*video).AudioFilePath = outputPath
	results <- ChannelMessage{Result: video}
}
