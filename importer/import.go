package importer

import (
	"crypto/sha256"

	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"strings"

	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/azure"
)

type Importer struct {
	TerraformModulePath        string
	SubscriptionID             string
	IgnoreResourceTypePatterns []string
	SkipInitPlanShow           bool
	ResourceGraphClient        azure.ResourceGraphClient
	NameFormats                []NameFormat
	Logger                     *logrus.Logger
}

type NameFormat struct {
	Type                string
	NameFormat          string
	NameMatchType       NameMatchType
	NameFormatArguments []string
}

type Resource struct {
	ID                    string
	Address               string
	Type                  string
	Name                  string
	Location              string
	ResourceName          string
	ResourceNameMatchType NameMatchType
	MappedResources       []azure.Resource
	Properties            map[string]any
	PropertiesCalculated  map[string]any
}

type ResourceIssue struct {
	Resource  Resource
	IssueType IssueType
}

type IssueType string

const (
	IssueTypeNone                IssueType = "None"
	IssueTypeNoResourceID        IssueType = "NoResourceID"
	IssueTypeMultipleResourceIDs IssueType = "MultipleResourceIDs"
	IssueTypeUnusedResourceID    IssueType = "UnusedResourceID"
)

type NameMatchType string

const (
	NameMatchTypeExact      NameMatchType = "Exact"
	NameMatchTypeIDContains NameMatchType = "IDContains"
)

type ActionType string

const (
	ActionTypeNone    ActionType = ""
	ActionTypeUse     ActionType = "use"
	ActionTypeIgnore  ActionType = "ignore"
	ActionTypeReplace ActionType = "replace"
)

func (importer *Importer) Import() {
	backendOverrideFilePath := importer.createBackendOverrideFile()
	chDir := fmt.Sprintf("-chdir=%s", importer.TerraformModulePath)
	jsonFilePath := filepath.Join(importer.TerraformModulePath, "tfplan.json")

	if !importer.SkipInitPlanShow {
		importer.Logger.Info("Running Terraform init, plan and show")
		importer.executeTerraformInit(chDir)
		importer.executeTerraformPlan(chDir)
		importer.executeTerraformShow(chDir, jsonFilePath)
	}
	plan := importer.loadJSONFromFile(jsonFilePath)

	graphResources, err := importer.ResourceGraphClient.GetResources()
	if err != nil {
		importer.Logger.Fatalf("Error getting resources from Resource Graph: %v", err)
	}
	planResources := importer.readResourcesFromPlan(plan)
	planResources, issues := importer.mapResourcesFromGraphToPlan(graphResources, planResources)

	exportToJSON(issues, "issues.json", importer.TerraformModulePath, importer.Logger)
	exportToJSON(planResources, "resources.json", importer.TerraformModulePath, importer.Logger)

	if len(issues) > 0 {
		importer.Logger.Warnf("Found %d issues based on the Terraform Plan", len(issues))
		importer.exportIssuesToCsv(issues)
	}

	importer.removeBackendOverrideFile(backendOverrideFilePath)
}

func (importer *Importer) mapResourcesFromGraphToPlan(graphResources []azure.Resource, planResources map[string]Resource) (map[string]Resource, map[string]ResourceIssue) {
	issues := map[string]ResourceIssue{}
	uniqueUsedResources := make(map[string]azure.Resource)

	for resourceKey, resource := range planResources {
		for _, graphResource := range graphResources {
			if resource.ResourceNameMatchType == NameMatchTypeExact && strings.ToLower(graphResource.Name) == strings.ToLower(resource.ResourceName) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
				planResources[resourceKey] = resource
			}

			if resource.ResourceNameMatchType == NameMatchTypeIDContains && strings.Contains(strings.ToLower(graphResource.ID), strings.ToLower(resource.ResourceName)) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
				planResources[resourceKey] = resource
			}
		}

		if len(resource.MappedResources) == 0 {
			importer.Logger.Warnf("No matching resource ID found for Name: %s, Type: %s, Address: %s", resource.ResourceName, resource.Type, resource.Address)
			addIssue(issues, resource, IssueTypeNoResourceID)
			continue
		}

		for _, mappedResource := range resource.MappedResources {
			if _, exists := uniqueUsedResources[mappedResource.ID]; !exists {
				uniqueUsedResources[mappedResource.ID] = mappedResource
			}
		}

		if len(resource.MappedResources) > 1 {
			mappedResourceIDsBasedOnLocation := []azure.Resource{}

			for _, mappedResource := range resource.MappedResources {
				if resource.Location == mappedResource.Location {
					mappedResourceIDsBasedOnLocation = append(mappedResourceIDsBasedOnLocation, mappedResource)
				} else if strings.Contains(strings.ToLower(mappedResource.ID), strings.ToLower(resource.Location)) {
					mappedResourceIDsBasedOnLocation = append(mappedResourceIDsBasedOnLocation, mappedResource)
				}
			}

			if len(mappedResourceIDsBasedOnLocation) == 1 {
				resource.MappedResources = mappedResourceIDsBasedOnLocation
			} else {
				mappedResourceIDs := []string{}
				for _, mappedResource := range resource.MappedResources {
					mappedResourceIDs = append(mappedResourceIDs, mappedResource.ID)
				}
				importer.Logger.Warnf("More than 1 Resource ID has been matched for Name: %s, Type: %s, Address: %s -> %v", resource.ResourceName, resource.Type, resource.Address, resource.MappedResources)
				addIssue(issues, resource, IssueTypeMultipleResourceIDs)
			}
		}
	}

	for _, graphResource := range graphResources {
		if _, exists := uniqueUsedResources[graphResource.ID]; !exists {
			importer.Logger.Warnf("Resource ID %s is not used in the Terraform plan", graphResource.ID)
			addIssue(issues, Resource{ID: getIdentityHash(graphResource.ID), MappedResources: []azure.Resource{graphResource}}, IssueTypeUnusedResourceID)
		}
	}
	return planResources, issues
}

func addIssue(issues map[string]ResourceIssue, resource Resource, issueType IssueType) {
	issues[resource.ID] = ResourceIssue{Resource: resource, IssueType: issueType}
}

func getIdentityHash(id string) string {
	sha256ID := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", sha256ID)[0:7]
}

func exportToJSON[V Resource | ResourceIssue](resources map[string]V, fileName string, modulePath string, logger *logrus.Logger) {
	jsonResources, err := json.Marshal(resources)
	if err != nil {
		logger.Fatal("Error during Marshal(): ", err)
	}
	jsonFilePath := filepath.Join(modulePath, fileName)
	err = os.WriteFile(jsonFilePath, jsonResources, 0644)
	if err != nil {
		logger.Fatal("Error writing file: ", err)
	}
}

func (importer *Importer) readResourcesFromPlan(plan map[string]any) map[string]Resource {
	resources := map[string]Resource{}

	for _, resource := range plan["resource_changes"].([]any) {
		resourceChange := resource.(map[string]any)

		mode := resourceChange["mode"].(string)
		if mode != "managed" {
			importer.Logger.Tracef("Skipping resource with mode %s", mode)
			continue
		}

		resource := Resource{}
		resource.Address = resourceChange["address"].(string)
		resource.ID = getIdentityHash(resource.Address)

		shouldIgnore := false
		for _, pattern := range importer.IgnoreResourceTypePatterns {
			matched, err := regexp.MatchString(pattern, resource.Address)
			if err != nil {
				importer.Logger.Debugf("Error matching pattern %s: %v", pattern, err)
				continue
			}
			if matched {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore {
			importer.Logger.Tracef("Ignoring Resource: %s", resource.Address)
			continue
		}

		resource.Type = resourceChange["type"].(string)
		resource.Name = resourceChange["name"].(string)

		resource.Properties = resourceChange["change"].(map[string]any)["after"].(map[string]any)
		resource.PropertiesCalculated = resourceChange["change"].(map[string]any)["after_unknown"].(map[string]any)

		foundName := false

		for _, nameFormat := range importer.NameFormats {
			if nameFormat.Type == resource.Type {
				nameFormatArguments := []any{}
				for _, arg := range nameFormat.NameFormatArguments {
					if val, ok := resource.Properties[arg]; ok {
						nameFormatArguments = append(nameFormatArguments, val.(string))
					} else {
						importer.Logger.Debugf("Name format argument %s not found in resource properties", arg)
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
				resource.ResourceNameMatchType = NameMatchTypeExact
				foundName = true
			}
		}

		if !foundName {
			importer.Logger.Tracef("Resource %s does not have a name property or mapped name property", resource.Address)
		}

		if val, ok := resource.Properties["location"]; ok {
			if val != nil {
				resource.Location = val.(string)
			}
		}

		resources[resource.ID] = resource
		importer.Logger.Tracef("Adding Resource: %s", resource.Address)
	}
	return resources
}

func (importer *Importer) removeBackendOverrideFile(backendOverrideFilePath string) {
	err := os.Remove(backendOverrideFilePath)
	if err != nil {
		importer.Logger.Fatalf("Failed to remove file: %v", err)
	}
}

func (importer *Importer) loadJSONFromFile(jsonFilePath string) map[string]any {
	content, err := os.ReadFile(jsonFilePath)
	if err != nil {
		importer.Logger.Fatal("Error when opening file: ", err)
	}

	var payload map[string]any
	err = json.Unmarshal(content, &payload)
	if err != nil {
		importer.Logger.Fatal("Error during Unmarshal(): ", err)
	}
	return payload
}

func (importer *Importer) executeTerraformShow(chDir string, jsonFilePath string) {
	cmd := exec.Command("terraform", chDir, "show", "-json", "tfplan")
	file, err := os.Create(jsonFilePath)
	if err != nil {
		importer.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	cmd.Stdout = file
	cmd.Stderr = os.Stderr

	// Run the command
	importer.Logger.Infof("Running Terraform show: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		importer.Logger.Fatalf("Error: %s", err)
	}
}

func (importer *Importer) executeTerraformPlan(chDir string) {
	cmd := exec.Command("terraform", chDir, "plan", "-out=tfplan")
	env := cmd.Environ()
	env = append(env, fmt.Sprintf("ARM_SUBSCRIPTION_ID=%s", importer.SubscriptionID))
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	importer.Logger.Infof("Running Terraform plan: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		importer.Logger.Fatalf("Error: %s", err)
	}
}

func (importer *Importer) executeTerraformInit(chDir string) {
	cmd := exec.Command("terraform", chDir, "init", "-upgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	importer.Logger.Infof("Running Terraform init: %s", cmd.String())
	if err := cmd.Run(); err != nil {
		importer.Logger.Fatalf("Error: %s", err)
	}
}

func (importer *Importer) createBackendOverrideFile() string {
	backendOverrideFilePath := filepath.Join(importer.TerraformModulePath, "backend_override.tf")

	importer.Logger.Tracef("Creating backend override file: %s", backendOverrideFilePath)

	backendOverrideFile, err := os.Create(backendOverrideFilePath)
	if err != nil {
		importer.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer backendOverrideFile.Close()
	_, err = backendOverrideFile.WriteString(fmt.Sprintf("terraform {\n  backend \"local\" {}\n}\n"))
	if err != nil {
		importer.Logger.Fatalf("Failed to write to file: %v", err)
	}
	return backendOverrideFilePath
}
