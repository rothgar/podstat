package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "podstat",
	Short: "A tool to fetch YouTube playlist video statistics",
	Long:  `Podstat fetches view counts and other statistics for videos in a YouTube playlist.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}