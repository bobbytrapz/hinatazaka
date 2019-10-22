package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanCmd)
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete chrome user profile directory.",
	Run: func(cmd *cobra.Command, args []string) {
		d := userProfileDir
		if d[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				panic(err)
			}
			d = filepath.Join(home, d[2:])
		}
		println("[delete]", d)
		if err := os.RemoveAll(d); err != nil {
			panic(err)
		}
	},
}
