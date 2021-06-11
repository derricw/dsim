package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dsim",
	Short: "dsim is a simple process simulator",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		// what to do when run with no subcommands?
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
