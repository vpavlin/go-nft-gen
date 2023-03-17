package nftgen

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var debug = false
var version = "v0.0.0"

var rootCmd = &cobra.Command{
	Use:     "nftgen",
	Version: version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logrus.Fatalln(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", "config.json", "Path to a config file")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable DEBUG log level")

}

func toggleDebug(cmd *cobra.Command, args []string) {
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
}
