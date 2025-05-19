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
		log.SetFormatter(&logrus.TextFormatter{})
		if viper.GetBool("structuredLogs") {
			log.SetFormatter(&logrus.JSONFormatter{})
		}

		for key, value := range viper.GetViper().AllSettings() {
			log.Debugf("Command Flag: %s = %s", key, value)
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

		importerInstance := importer.Importer{}
		importerInstance.TerraformModulePath = viper.GetString("terraformModulePath")
		importerInstance.SubscriptionID = graphInstance.SubscriptionIDs[0]
		importerInstance.IgnoreResourceTypePatterns = viper.GetStringSlice("ignoreResourceTypePatterns")
		importerInstance.SkipInitPlanShow = viper.GetBool("skipInitPlanShow")

	    nameFormats := viper.Get("nameFormats").([]interface{})
		for _, rawNameFormat := range nameFormats {
			nameFormatMap := rawNameFormat.(map[string]interface{})
			nameFormatArguments := []string{}

			for _, arg := range nameFormatMap["nameformatarguments"].([]interface{}) {
				nameFormatArguments = append(nameFormatArguments, arg.(string))
			}

			importerInstance.NameFormats = append(importerInstance.NameFormats, importer.NameFormat{
				Type:  nameFormatMap["type"].(string),
				NameFormat: nameFormatMap["nameformat"].(string),
				NameMatchType: importer.NameMatchType(nameFormatMap["namematchtype"].(string)),
				NameFormatArguments: nameFormatArguments,
			})
		}

		importerInstance.Logger = log
		importerInstance.GraphResources, _ = graphInstance.GetResources()
		importerInstance.Import()
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
