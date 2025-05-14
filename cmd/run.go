/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/graph"
	"github.com/azure/terraform-state-importer/importer"
	"github.com/spf13/cobra"
 	"github.com/spf13/viper"
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

		for key, value := range viper.GetViper().AllSettings() {
			log.Infof("Command Flag: %s = %s", key, value)
		}

		graphInstance := graph.Graph{}
		graphInstance.SubscriptionIDs = viper.GetStringSlice("subscriptionIDs")
		graphInstance.IgnoreResourceIDPatterns = viper.GetStringSlice("ignoreResourceIDPatterns")

		rawQueries := viper.Get("resourceGraphQueries").([]interface{})
		for _, rawQuery := range rawQueries {
			queryMap := rawQuery.(map[string]interface{})
			graphInstance.ResourceGraphQueries = append(graphInstance.ResourceGraphQueries, graph.ResourceGraphQuery{
				Name:  queryMap["name"].(string),
				Query: queryMap["query"].(string),
			})
		}
		graphInstance.Logger = log

		importer := importer.Importer{}
		importer.TerraformModulePath = viper.GetString("terraformModulePath")
		importer.SubscriptionID = graphInstance.SubscriptionIDs[0]
		importer.IgnoreResourceTypePatterns = viper.GetStringSlice("ignoreResourceTypePatterns")
		importer.SkipInitPlanShow = viper.GetBool("skipInitPlanShow")
		importer.GraphResources, _ = graphInstance.GetResources()
		importer.Logger = log
		importer.Import()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringSliceP("subscriptionIDs", "s", nil, "Subscription IDs to use")
	viper.BindPFlag("subscriptionIDs", runCmd.PersistentFlags().Lookup("subscriptionIDs"))
	runCmd.PersistentFlags().StringSliceP("ignoreResourceIDPatterns", "i", nil, "Resource ID patterns to ignore")
	viper.BindPFlag("ignoreResourceIDPatterns", runCmd.PersistentFlags().Lookup("ignoreResourceIDPatterns"))
	runCmd.PersistentFlags().StringSliceP("ignoreResourceTypePatterns", "r", nil, "Resource type patterns to ignore")
	viper.BindPFlag("ignoreResourceTypePatterns", runCmd.PersistentFlags().Lookup("ignoreResourceTypePatterns"))
	runCmd.PersistentFlags().StringP("terraformModulePath", "t", ".", "Terraform module path to use")
	viper.BindPFlag("terraformModulePath", runCmd.PersistentFlags().Lookup("terraformModulePath"))
	runCmd.PersistentFlags().BoolP("skipInitPlanShow", "x", false, "Skip init, plan, and show steps")
	viper.BindPFlag("skipInitPlanShow", runCmd.PersistentFlags().Lookup("skipInitPlanShow"))
}
