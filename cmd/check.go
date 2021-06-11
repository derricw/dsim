package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	//atCmd.Flags().StringVarP(&storageAddr, "storage", "s", "", "storage address")
	rootCmd.AddCommand(checkCmd)
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "checks a model for errors",
	Long:  ``,
	//Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("at time <> with args %v", args)
	},
}
