/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/analyzer"
	"github.com/azure/terraform-state-importer/azure"
	"github.com/azure/terraform-state-importer/csv"
	"github.com/azure/terraform-state-importer/filepathparser"
	"github.com/azure/terraform-state-importer/hcl"
	"github.com/azure/terraform-state-importer/json"
	"github.com/azure/terraform-state-importer/terraform"
	"github.com/azure/terraform-state-importer/types"
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

		workingFolderPath, err := filepathparser.ParsePath(viper.GetString("workingFolderPath"))
		if err != nil {
			log.Fatalf("Error getting working folder path: %v", err)
		}
		terraformModulePath, err := filepathparser.ParsePath(viper.GetString("terraformModulePath"))
		if err != nil {
			log.Fatalf("Error getting terraform module path: %v", err)
		}

		planAsTextOnly, _ := cmd.Flags().GetBool("planAsTextOnly")

		resourceGraphQueries := []types.ResourceGraphQuery{}
		resourceGraphQueriesRaw := viper.Get("resourceGraphQueries").([]any)
		for _, rawQuery := range resourceGraphQueriesRaw {
			queryMap := rawQuery.(map[string]any)
			resourceGraphQueries = append(resourceGraphQueries, types.ResourceGraphQuery{
				Name:  queryMap["name"].(string),
				Scope: types.ResourceGraphQueryScope(queryMap["scope"].(string)),
				Query: queryMap["query"].(string),
			})
		}

		nameFormats := []terraform.NameFormat{}
		if viper.InConfig("nameFormats") {
			nameFormatsRaw := viper.Get("nameFormats").([]any)
			for _, rawNameFormat := range nameFormatsRaw {
				nameFormatMap := rawNameFormat.(map[string]any)
				nameFormatArguments := []string{}

				for _, arg := range nameFormatMap["nameformatarguments"].([]any) {
					nameFormatArguments = append(nameFormatArguments, arg.(string))
				}

				nameFormats = append(nameFormats, terraform.NameFormat{
					Type:                nameFormatMap["type"].(string),
					NameFormat:          nameFormatMap["nameformat"].(string),
					NameMatchType:       types.NameMatchType(nameFormatMap["namematchtype"].(string)),
					NameFormatArguments: nameFormatArguments,
				})
			}
		}

		deleteCommands := []types.DeleteCommand{}
		if viper.InConfig("deleteCommands") {
			deleteCommandsRaw := viper.Get("deleteCommands").([]any)
			for _, rawDeleteCommand := range deleteCommandsRaw {
				deleteCommandMap := rawDeleteCommand.(map[string]any)
				deleteCommands = append(deleteCommands, types.DeleteCommand{
					Type:    deleteCommandMap["type"].(string),
					Command: deleteCommandMap["command"].(string),
				})
			}
		}

		resourceGraphClient := azure.NewResourceGraphClient(
			viper.GetStringSlice("managementGroupIDs"),
			viper.GetStringSlice("subscriptionIDs"),
			viper.GetStringSlice("ignoreResourceIDPatterns"),
			resourceGraphQueries,
			log,
		)

		jsonClient := json.NewJsonClient(
			workingFolderPath,
			log,
		)

		planClient := terraform.NewPlanClient(
			terraformModulePath,
			workingFolderPath,
			viper.GetString("planSubscriptionID"),
			viper.GetStringSlice("ignoreResourceTypePatterns"),
			viper.GetBool("skipInitPlanShow"),
			viper.GetBool("skipInitOnly"),
			nameFormats,
			jsonClient,
			log,
		)

		if planAsTextOnly {
			planClient.PlanAsText()
			return
		}

		issueCsvClient := csv.NewIssueCsvClient(
			workingFolderPath,
			viper.GetString("issuesCsv"),
			log,
		)

		hclClient := hcl.NewHclClient(
			terraformModulePath,
			log,
		)

		mappingClient := analyzer.NewMappingClient(
			workingFolderPath,
			viper.GetString("issuesCsv") != "",
			resourceGraphClient,
			planClient,
			issueCsvClient,
			jsonClient,
			hclClient,
			log,
		)

		mappingClient.Map()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringP("terraformModulePath", "t", ".", "Terraform module path to use")
	viper.BindPFlag("terraformModulePath", runCmd.PersistentFlags().Lookup("terraformModulePath"))
	runCmd.PersistentFlags().StringP("workingFolderPath", "w", ".", "Working folder path to use")
	viper.BindPFlag("workingFolderPath", runCmd.PersistentFlags().Lookup("workingFolderPath"))
	runCmd.PersistentFlags().StringP("issuesCsv", "c", "", "CSV File path to use")
	viper.BindPFlag("issuesCsv", runCmd.PersistentFlags().Lookup("issuesCsv"))
	runCmd.PersistentFlags().BoolP("skipInitPlanShow", "x", false, "Skip init, plan, and show steps")
	viper.BindPFlag("skipInitPlanShow", runCmd.PersistentFlags().Lookup("skipInitPlanShow"))
	runCmd.PersistentFlags().BoolP("skipInitOnly", "k", false, "Skip init step")
	viper.BindPFlag("skipInitOnly", runCmd.PersistentFlags().Lookup("skipInitOnly"))
	runCmd.PersistentFlags().BoolP("planAsTextOnly", "p", false, "Run the tool to generate a textual plan only")
	viper.BindPFlag("planAsTextOnly", runCmd.PersistentFlags().Lookup("planAsTextOnly"))
	runCmd.PersistentFlags().StringP("planSubscriptionID", "s", "", "Subscription ID to use for terraform plan if not using the az cli subscription ID")
	viper.BindPFlag("planSubscriptionID", runCmd.PersistentFlags().Lookup("planSubscriptionID"))
}
