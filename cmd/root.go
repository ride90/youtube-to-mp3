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
				cmd.Println("The following issues occurred during execution:")
				for _, err := range errs {
					cmd.Printf(" - %v\n", err)
				}
				cmd.Println("Address errors and retry.")
				cmd.Println("You can also send a pull request https://github.com/ride90/youtube-to-mp3 :)")
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
	// TODO: Uncomment me when time has come.
	errs := video.ValidateLinks(links)
	if len(errs) > 0 {
		return errs
	}
	// Get playback stream URLs.
	numberCount := len(links)
	resultsChanel := make(chan video.ChannelMessage, numberCount)
	for _, link := range links {
		go video.GetPlaybackURL(link, resultsChanel)
	}
	for a := 1; a <= numberCount; a++ {
		fmt.Println(<-resultsChanel)
		fmt.Printf("\n\n\n")
	}
	close(resultsChanel)

	return nil
}

func handleNames(names []string) {
	fmt.Println(names)
	fmt.Println("Not implemented handleNames")
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
