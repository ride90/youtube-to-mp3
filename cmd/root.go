package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ride90/youtube-to-mp3/video"
)

var names, links []string

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
		// Handle links.
		if len(links) > 0 {
			errs := handleLinks(links)
			// Stdout errors and exit.
			if len(errs) > 0 {
				cmd.Printf("\nThe following issues occurred during execution:\n")
				for _, err := range errs {
					cmd.Printf(" - %v\n", err)
				}
				cmd.Printf(
					"\nAddress errors and retry." +
						"\nYou can also send a pull request https://github.com/ride90/youtube-to-mp3 :)\n\n",
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

func handleLinks(links []string) []error {
	// Validate links. If at least one link is not valid we stop an execution.
	errs := video.ValidateLinks(links)
	if len(errs) > 0 {
		return errs
	}
	// Get playback stream URLs.
	var videos []*video.Video
	var errors []error
	channelGetPlaybackURL := make(chan video.ChannelMessage, len(links))
	for _, link := range links {
		// Start go runtime thread.
		go video.GetPlaybackURL(link, channelGetPlaybackURL)
	}
	for i := 0; i < len(links); i++ {
		// Wait until all threads are done.
		msg := <-channelGetPlaybackURL
		if msg.Err != nil {
			errors = append(errors, msg.Err)
		} else if msg.Result.HasStreamURL() {
			videos = append(videos, msg.Result)
		}
	}
	close(channelGetPlaybackURL)
	// Fetch metadata.
	channelFetchMetadata := make(chan video.ChannelMessage, len(videos))
	for _, _video := range videos {
		go video.FetchMetadata(_video, channelFetchMetadata)
	}
	for i := 0; i < len(videos); i++ {
		msg := <-channelFetchMetadata
		fmt.Println(*(msg.Result))
	}
	close(channelFetchMetadata)

	return nil
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
}
