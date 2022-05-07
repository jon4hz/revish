package cmd

import (
	"fmt"

	"github.com/jon4hz/revish/interal/version"
	"github.com/muesli/coral"
)

var rootCmd = &coral.Command{
	Version: version.Version,
	Use:     "revish",
	Short:   "revish is a reverse ssh shell",
	Long:    "revish is a reverse ssh shell",
	RunE:    root,
}

func root(cmd *coral.Command, args []string) error {
	return cmd.Help()
}

func init() {
	rootCmd.AddCommand(serverCmd, clientCmd, versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

var versionCmd = &coral.Command{
	Use:   "version",
	Short: "Print the version info",
	Run: func(cmd *coral.Command, args []string) {
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Printf("Commit: %s\n", version.Commit)
		fmt.Printf("Date: %s\n", version.Date)
		fmt.Printf("Build by: %s\n", version.BuiltBy)
	},
}
