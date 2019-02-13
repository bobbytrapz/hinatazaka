package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version string",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hinatazaka", "v0.1.0")
	},
}
