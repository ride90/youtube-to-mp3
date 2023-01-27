package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var names, links []string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:       "yt2mp3",
	Short:     "Command line tool for converting YouTube videos to mp3 files.",
	Long:      `Command line tool for converting YouTube videos to mp3 files. Created for educational purposes only.`,
	Args:      cobra.OnlyValidArgs,
	ValidArgs: []string{"name", "link", "yt2mp3"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(names) == 0 && len(links) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
		fmt.Println("names", names)
		fmt.Println("links", links)
	},
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
	rootCmd.Flags().StringSliceVarP(&names, "name", "n", []string{}, "")
	rootCmd.Flags().StringSliceVarP(&links, "link", "l", []string{}, "")
}
