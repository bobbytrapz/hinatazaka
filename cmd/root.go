package cmd

import (
	"os"	

	"github.com/bobbytrapz/hinatazaka/options"
	"github.com/spf13/cobra"
)

var verbose bool
var userProfileDir = "~/.config/hinatazaka/hinatazaka-profile"
var port = options.GetInt("chrome_port")

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
}

var rootCmd = &cobra.Command{
	Use:   "hinatazaka",
	Short: "hinatazaka is collection of tools related to 日向坂４６",
	Long: `hinatazaka is collection of tools related to 日向坂４６
Bobby wrote this.
https://github.com/bobbytrapz/hinatazaka#readme
`,
	Run: func(cmd *cobra.Command, args []string) {	
		cmd.Usage()
	},
}

// Execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
