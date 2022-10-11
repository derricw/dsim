package cmd

import (
	"time"

	"github.com/derricw/dsim/model"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var Verbose bool
var Quiet bool

func init() {
	//atCmd.Flags().StringVarP(&storageAddr, "storage", "s", "", "storage address")
	runCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose mode")
	runCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "s", false, "don't print a report")
	runCmd.AddCommand(forCmd)
	runCmd.AddCommand(untilCmd)
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "runs a model",
	Long:  ``,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var forCmd = &cobra.Command{
	Use:   "for <duration> <model>",
	Short: "run a model for a certain duration",
	Long:  ``,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if Verbose {
			log.SetLevel(log.DebugLevel)
		}
		log.Infof("running model @ %v", args)
		m, err := model.NewModelFromFile(args[1])
		if err != nil {
			log.Fatal(err)
		}
		//log.Debugf("model: %+#v", m)
		duration, err := time.ParseDuration(args[0])
		if err != nil {
			log.Fatal(err)
		}

		m.RunUntilTime(duration)
		time.Sleep(1 * time.Second) // come up with a way to know when we're done.
		if !Quiet {
			m.Report()
		}
	},
}

var untilCmd = &cobra.Command{
	Use:   "until <condition> <model>",
	Short: "run a model until a condition is met",
	Long:  ``,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if Verbose {
			log.SetLevel(log.DebugLevel)
		}
		log.Infof("running model @ %v", args)
		m, err := model.NewModelFromFile(args[1])
		if err != nil {
			log.Fatal(err)
		}
		//log.Debugf("model: %+#v", m)
		conditions, err := ParseConditions(args[0])
		if err != nil {
			log.Fatal(err)
		}

		m.RunUntilConditions(conditions)
		if !Quiet {
			m.Report()
		}
	},
}

func ParseConditions(conditionsFile string) (model.ConditionSet, error) {
	return model.ConditionSet{}, nil
}
