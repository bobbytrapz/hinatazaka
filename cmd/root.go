package cmd

import (
	"os"
	"path/filepath"

	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/spf13/cobra"
)

var verbose bool
var userProfileDir = "~/.config/hinatazaka/hinatazaka-profile"
var port = options.GetInt("chrome_port")

var shouldDeleteProfileDirectory = false

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	rootCmd.Flags().BoolVar(&shouldDeleteProfileDirectory, "delete-profile", false, "Delete chrome user profile directory.")
}

var rootCmd = &cobra.Command{
	Use:   "hinatazaka",
	Short: "hinatazaka is collection of tools related to 日向坂４６",
	Long: `hinatazaka is collection of tools related to 日向坂４６
Bobby wrote this.
https://github.com/bobbytrapz/hinatazaka#readme
`,
	Run: func(cmd *cobra.Command, args []string) {
		if shouldDeleteProfileDirectory {
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
			return
		}

		cmd.Usage()
	},
}

// Execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
