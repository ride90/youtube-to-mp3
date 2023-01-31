package cmd

import (
	"os"

	"github.com/gosuri/uiprogress"
	"github.com/spf13/cobra"

	"github.com/ride90/youtube-to-mp3/video"
)

var names, links []string
var destination string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:       "yt2mp3",
	Short:     "Command line tool for converting YouTube videos to mp3 files.",
	Long:      `Command line tool for converting YouTube videos to mp3 files. Created for educational purposes only.`,
	Args:      cobra.OnlyValidArgs,
	ValidArgs: []string{"names", "links", "yt2mp3"},
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no args provided.
		if len(names) == 0 && len(links) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
		// Init progress bar.
		uiprogress.Start()
		// Handle links.
		if len(links) > 0 {
			errs := handleLinks(cmd, links)
			// Stdout errors and exit.
			if len(errs) > 0 {
				cmd.Printf("\nThe following issues occurred during execution:\n")
				for _, err := range errs {
					cmd.Printf(" - %v\n", err)
				}
				cmd.Printf(
					"\nAddress errors and retry.\n" +
						"You can also send a pull request https://github.com/ride90/youtube-to-mp3 :)\n\n",
				)
				os.Exit(1)
			}
		}
		// Handle names.
		if len(names) > 0 {
			handleNames(names)
		}
	},
}

func handleLinks(cmd *cobra.Command, links []string) []error {
	// Validate links. If at least one link is not valid we stop an execution.
	errs := video.ValidateLinks(links)
	if len(errs) > 0 {
		return errs
	}

	// Get playback stream URLs.
	var videos []*video.Video
	var errors []error
	channelFetchPlaybackURL := make(chan video.ChannelMessage, len(links))
	for _, link := range links {
		// Start go runtime thread.
		go video.FetchPlaybackURL(link, channelFetchPlaybackURL)
	}
	for i := 0; i < len(links); i++ {
		// Wait until all threads are done.
		msg := <-channelFetchPlaybackURL
		if msg.Err != nil {
			errors = append(errors, msg.Err)
		} else if msg.Result.HasStreamURL() {
			videos = append(videos, msg.Result)
		}
	}
	close(channelFetchPlaybackURL)

	// Fetch metadata.
	channelFetchMetadata := make(chan video.ChannelMessage, len(videos))
	for _, _video := range videos {
		go video.FetchMetadata(_video, channelFetchMetadata)
	}
	for i := 0; i < len(videos); i++ {
		msg := <-channelFetchMetadata
		if msg.Err != nil {
			errors = append(errors, msg.Err)
		}
	}
	close(channelFetchMetadata)

	// Download and save temp video files.
	channelFetchVideo := make(chan video.ChannelMessage, len(videos))
	for _, _video := range videos {
		go video.FetchVideo(_video, channelFetchVideo)
	}
	for i := 0; i < len(videos); i++ {
		msg := <-channelFetchVideo
		if msg.Err != nil {
			errors = append(errors, msg.Err)
		}
	}
	// Cleanup file when main function is over.
	defer func(videos []*video.Video) {
		for _, v := range videos {
			err := os.Remove((*v.File).Name())
			if err != nil {
				errors = append(errors, err)
			}
		}
	}(videos)

	// Run ffmpeg and convert videos to mp3 files.
	channelConvertVideoToAudio := make(chan video.ChannelMessage, len(videos))
	for _, _video := range videos {
		go video.ConvertVideoToAudio(_video, channelConvertVideoToAudio)
	}
	for i := 0; i < len(videos); i++ {
		msg := <-channelConvertVideoToAudio
		if msg.Err != nil {
			errors = append(errors, msg.Err)
		}
	}

	return errors
}

func handleNames(names []string) {
	panic("handleNames is not implemented")
}

// Execute This is called by main.main().
// It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Define flags.
	rootCmd.Flags().StringSliceVarP(&names, "names", "n", []string{}, "")
	rootCmd.Flags().StringSliceVarP(&links, "links", "l", []string{}, "")
	rootCmd.Flags().StringP("dest", "d", "", "")
}
