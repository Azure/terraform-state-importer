package terraform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/json"
	"github.com/azure/terraform-state-importer/types"
)

type IPlanClient interface {
	PlanAndGetResources() []*types.PlanResource
	PlanAsText()
}

type PlanClient struct {
	TerraformModulePath        string
	WorkingFolderPath          string
	SubscriptionID             string
	IgnoreResourceTypePatterns []string
	SkipInitPlanShow           bool
	SkipInitOnly               bool
	NameFormats                []NameFormat
	JsonClient                 json.IJsonClient
	Logger                     *logrus.Logger
}

func NewPlanClient(terraformModulePath string, workingFolderPath string, subscriptionID string, ignoreResourceTypePatterns []string, skipInitPlanShow bool, skipInitOnly bool, nameFormats []NameFormat, jsonClient json.IJsonClient, logger *logrus.Logger) *PlanClient {
	return &PlanClient{
		TerraformModulePath:        terraformModulePath,
		WorkingFolderPath:          workingFolderPath,
		SubscriptionID:             subscriptionID,
		IgnoreResourceTypePatterns: ignoreResourceTypePatterns,
		SkipInitPlanShow:           skipInitPlanShow,
		SkipInitOnly:               skipInitOnly,
		NameFormats:                nameFormats,
		JsonClient:                 jsonClient,
		Logger:                     logger,
	}
}

type NameFormat struct {
	Type                string
	NameFormat          string
	NameMatchType       types.NameMatchType
	NameFormatArguments []string
}

func (planClient *PlanClient) PlanAndGetResources() []*types.PlanResource {
	jsonFileName := "tfplan.json"

	if !planClient.SkipInitPlanShow {
		planFileName := "tfplan"
		backendOverrideFilePath := planClient.createBackendOverrideFile()
		chDir := fmt.Sprintf("-chdir=%s", planClient.TerraformModulePath)
		planClient.Logger.Info("Running Terraform init, plan and show")

		if !planClient.SkipInitOnly {
			planClient.executeTerraformInit(chDir)
		}
		planClient.executeTerraformPlan(chDir, planFileName)
		planClient.executeTerraformShow(chDir, planFileName, jsonFileName, true)
		planClient.removeBackendOverrideFile(backendOverrideFilePath)
	}

	plan := planClient.JsonClient.Import(jsonFileName)
	return planClient.readResourcesFromPlan(plan)
}

func (planClient *PlanClient) PlanAsText() {
	textFileName := "tfplan.txt"

	if !planClient.SkipInitPlanShow {
		planFileName := "tfplan"
		backendOverrideFilePath := planClient.createBackendOverrideFile()
		chDir := fmt.Sprintf("-chdir=%s", planClient.TerraformModulePath)
		planClient.Logger.Info("Running Terraform init, plan and show")

		if !planClient.SkipInitOnly {
			planClient.executeTerraformInit(chDir)
		}
		planClient.executeTerraformPlan(chDir, planFileName)
		planClient.executeTerraformShow(chDir, planFileName, textFileName, false)
		planClient.removeBackendOverrideFile(backendOverrideFilePath)
	}

	outputFileName := "tfplan_updates.txt"
	planClient.ExtractUpdateResourcesFromPlan(textFileName, outputFileName)
}

func (planClient *PlanClient) getCurrentSubscriptionID() string {
	cmd := exec.Command("az", "account", "show", "--query", "id", "-o", "tsv")
	env := cmd.Environ()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	cmd.Env = env
	cmd.Stderr = os.Stderr

	planClient.Logger.Debugf("Running az cli: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		planClient.Logger.Fatalf("Error: %s", err)
	}

	output := stdout.String()
	output = strings.ReplaceAll(output, "\r", "")
	output = strings.ReplaceAll(output, "\n", "")
	planClient.Logger.Debugf("Subscription ID: %s", output)

	return output
}

func (planClient *PlanClient) readResourcesFromPlan(plan map[string]any) []*types.PlanResource {
	resources := []*types.PlanResource{}

	for _, resource := range plan["resource_changes"].([]any) {
		resourceChange := resource.(map[string]any)

		mode := resourceChange["mode"].(string)
		if mode != "managed" {
			planClient.Logger.Tracef("Skipping resource with mode %s", mode)
			continue
		}

		resource := types.PlanResource{}
		resource.Address = resourceChange["address"].(string)

		shouldIgnore := false
		for _, pattern := range planClient.IgnoreResourceTypePatterns {
			matched, err := regexp.MatchString(pattern, resource.Address)
			if err != nil {
				planClient.Logger.Debugf("Error matching pattern %s: %v", pattern, err)
				continue
			}
			if matched {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore {
			planClient.Logger.Tracef("Ignoring Resource: %s", resource.Address)
			continue
		}

		resource.Type = resourceChange["type"].(string)
		resource.Name = resourceChange["name"].(string)

		resource.Properties = resourceChange["change"].(map[string]any)["after"].(map[string]any)
		resource.PropertiesCalculated = resourceChange["change"].(map[string]any)["after_unknown"].(map[string]any)

		if resource.Type == "azapi_resource" {
			if subType, ok := resource.Properties["type"]; ok {
				resourceTypeSplit := strings.Split(subType.(string), "@")
				resource.SubType = resourceTypeSplit[0]
				resource.APIVersion = resourceTypeSplit[1]
			}
		}

		foundName := false

		for _, nameFormat := range planClient.NameFormats {
			if nameFormat.Type == resource.Type || nameFormat.Type == resource.SubType {
				nameFormatArguments := []any{}
				for _, arg := range nameFormat.NameFormatArguments {
					if val, ok := resource.Properties[arg]; ok {
						nameFormatArguments = append(nameFormatArguments, val.(string))
					} else {
						planClient.Logger.Debugf("Name format argument %s not found in resource properties", arg)
					}
				}

				resource.ResourceName = fmt.Sprintf(nameFormat.NameFormat, nameFormatArguments...)
				resource.ResourceNameMatchType = nameFormat.NameMatchType
				foundName = true
			}
		}

		if !foundName {
			if val, ok := resource.Properties["name"]; ok {
				resource.ResourceName = val.(string)
				resource.ResourceNameMatchType = types.NameMatchTypeExact
				foundName = true
			}
		}

		if !foundName {
			planClient.Logger.Tracef("Resource %s does not have a name property or mapped name property", resource.Address)
		}

		if val, ok := resource.Properties["location"]; ok {
			if val != nil {
				resource.Location = val.(string)
			}
		}

		resources = append(resources, &resource)
		planClient.Logger.Tracef("Adding Resource: %s", resource.Address)
	}
	return resources
}

func (planClient *PlanClient) executeTerraformInit(chDir string) {
	cmd := exec.Command("terraform", chDir, "init", "-upgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	planClient.Logger.Infof("Running Terraform init: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		planClient.Logger.Fatalf("Error: %s", err)
	}
}

func (planClient *PlanClient) executeTerraformPlan(chDir string, planFileName string) {
	planFilePath := filepath.Join(planClient.WorkingFolderPath, planFileName)

	cmd := exec.Command("terraform", chDir, "plan", fmt.Sprintf("-out=%s", planFilePath))
	env := cmd.Environ()

	subscriptionID := planClient.SubscriptionID
	if subscriptionID == "" {
		subscriptionID = planClient.getCurrentSubscriptionID()
	}

	env = append(env, fmt.Sprintf("ARM_SUBSCRIPTION_ID=%s", subscriptionID))
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	planClient.Logger.Infof("Running Terraform plan: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		planClient.Logger.Fatalf("Error: %s", err)
	}
}

func (planClient *PlanClient) executeTerraformShow(chDir string, planFileName string, outputFileName string, jsonPlan bool) {
	planFilePath := filepath.Join(planClient.WorkingFolderPath, planFileName)
	jsonFilePath := filepath.Join(planClient.WorkingFolderPath, outputFileName)

	argument := "-json"
	if !jsonPlan {
		argument = "-no-color"
	}

	cmd := exec.Command("terraform", chDir, "show", argument, planFilePath)
	file, err := os.Create(jsonFilePath)
	if err != nil {
		planClient.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	cmd.Stdout = file
	cmd.Stderr = os.Stderr

	planClient.Logger.Infof("Running Terraform show: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		planClient.Logger.Fatalf("Error: %s", err)
	}
}

func (planClient *PlanClient) createBackendOverrideFile() string {
	backendOverrideFilePath := filepath.Join(planClient.TerraformModulePath, "backend_override.tf")

	planClient.Logger.Tracef("Creating backend override file: %s", backendOverrideFilePath)

	backendOverrideFile, err := os.Create(backendOverrideFilePath)
	if err != nil {
		planClient.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer backendOverrideFile.Close()
	_, err = backendOverrideFile.WriteString(fmt.Sprintf("terraform {\n  backend \"local\" {}\n}\n"))
	if err != nil {
		planClient.Logger.Fatalf("Failed to write to file: %v", err)
	}
	return backendOverrideFilePath
}

func (planClient *PlanClient) removeBackendOverrideFile(backendOverrideFilePath string) {
	err := os.Remove(backendOverrideFilePath)
	if err != nil {
		planClient.Logger.Fatalf("Failed to remove file: %v", err)
	}
}
