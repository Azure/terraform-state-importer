/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/graph"
	"github.com/azure/terraform-state-importer/importer"
	"github.com/spf13/cobra"
)

var log = logrus.New()

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		logVerbosity, _ := cmd.Flags().GetString("verbosity")
		logLevel, err := logrus.ParseLevel(logVerbosity)
		if err != nil {
			log.Fatalf("Invalid log level: %s", logVerbosity)
		}
		log.SetLevel(logLevel)
		log.SetFormatter(&logrus.JSONFormatter{})

		graph := graph.Graph{}
		graph.SubscriptionIDs, _ = cmd.Flags().GetStringSlice("subscriptionIDs")
		graph.IgnoreResourceIDPatterns, _ = cmd.Flags().GetStringSlice("ignoreResourceIDPatterns")
		graph.Logger = log

		importer := importer.Importer{}
		importer.TerraformModulePath, _ = cmd.Flags().GetString("terraformModulePath")
		importer.SubscriptionID = graph.SubscriptionIDs[0]
		importer.IgnoreResourceTypePatterns, _ = cmd.Flags().GetStringSlice("ignoreResourceTypePatterns")
		importer.SkipInitPlanShow, _ = cmd.Flags().GetBool("skipInitPlanShow")
		importer.GraphResources, _ = graph.GetResources()
		importer.Logger = log
		importer.Import()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	runCmd.Flags().StringSliceP("subscriptionIDs", "s", nil, "Subscription IDs to use")
	runCmd.Flags().StringSliceP("ignoreResourceIDPatterns", "i", nil, "Resource ID patterns to ignore")
	runCmd.Flags().StringSliceP("ignoreResourceTypePatterns", "r", nil, "Resource type patterns to ignore")
	runCmd.Flags().StringP("terraformModulePath", "t", ".", "Terraform module path to use")
	runCmd.Flags().BoolP("skipInitPlanShow", "x", false, "Skip init, plan, and show steps")
}
