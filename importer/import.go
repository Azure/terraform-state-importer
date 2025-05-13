package importer

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/azure/terraform-state-importer/graph"
)

type Importer struct{
	TerraformModulePath string
	SubscriptionID string
	IgnoreResourceTypePatterns []string
	SkipInitPlanShow bool
	GraphResources []graph.Resource
}

type Resource struct {
	Address string
	Type    string
	Name    string
	ResourceName string
	MappedResourceID string
	Properties map[string]interface{}
	PropertiesCalculated map[string]interface{}
}

func (importer *Importer) Import() {
	fmt.Println("Running Terraform init, plan and show", importer)

	backendOverrideFilePath := importer.createBackendOverrideFile()
	chDir := fmt.Sprintf("-chdir=%s", importer.TerraformModulePath)
	jsonFilePath := filepath.Join(importer.TerraformModulePath, "tfplan.json")

	if! importer.SkipInitPlanShow {
		fmt.Println("Running Terraform init, plan and show")
		importer.executeTerraformInit(chDir)
		importer.executeTerraformPlan(chDir)
		importer.executeTerraformShow(chDir, jsonFilePath)
	}
	plan := importer.loadJSONFromFile(jsonFilePath)

	resources := []Resource{}

	for _, resource := range plan["resource_changes"].([]interface{}) {
		resourceChange := resource.(map[string]interface{})

		mode := resourceChange["mode"].(string)
		if mode != "managed" {
			fmt.Printf("Skipping resource with mode %s\n", mode)
			continue
		}

		resource := Resource{}
		resource.Address = resourceChange["address"].(string)

		shouldIgnore := false
		for _, pattern := range importer.IgnoreResourceTypePatterns {
			matched, err := regexp.MatchString(pattern, resource.Address)
			if err != nil {
				fmt.Printf("Error matching pattern %s: %v\n", pattern, err)
				continue
			}
			if matched {
				shouldIgnore = true
				break
			}
		}
		if shouldIgnore {
			fmt.Printf("Ignoring Resource ID: %s\n", resource.Address)
			continue
		}

		resource.Type = resourceChange["type"].(string)
		resource.Name = resourceChange["name"].(string)
		resource.Properties = resourceChange["change"].(map[string]interface{})["after"].(map[string]interface{})
		resource.PropertiesCalculated = resourceChange["change"].(map[string]interface{})["after_unknown"].(map[string]interface{})
		if val, ok := resource.Properties["name"]; ok {
			resource.ResourceName = val.(string)

			for _, graphResource := range importer.GraphResources {
				if strings.ToLower(graphResource.Name) == strings.ToLower(resource.ResourceName) {
					resource.MappedResourceID = graphResource.ID
					continue
				}
				if strings.HasSuffix(strings.ToLower(graphResource.ID), strings.ToLower(resource.ResourceName)) {
					resource.MappedResourceID = graphResource.ID
					continue
				}
			}
		} else {
			log.Printf("Resource %s does not have a name property\n", resource.Address)
		}
		resources = append(resources, resource)
		fmt.Printf("Resource Address: %s\n", resource.Address)
	}

	jsonResources, err := json.Marshal(resources)
	if err != nil {
		log.Fatal("Error during Marshal(): ", err)
	}
	jsonFilePath = filepath.Join(importer.TerraformModulePath, "resources.json")
	err = os.WriteFile(jsonFilePath, jsonResources, 0644)
	if err != nil {
		log.Fatal("Error writing file: ", err)
	}

	importer.removeBackendOverrideFile(backendOverrideFilePath)
}

func (*Importer) removeBackendOverrideFile(backendOverrideFilePath string) {
	err := os.Remove(backendOverrideFilePath)
	if err != nil {
		log.Fatalf("Failed to remove file: %v", err)
	}
}

func (*Importer) loadJSONFromFile(jsonFilePath string) (map[string]interface{}) {
	content, err := os.ReadFile(jsonFilePath)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}

	var payload map[string]interface{}
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}
	return payload
}

func (importer *Importer) executeTerraformShow(chDir string, jsonFilePath string) {
	cmd := exec.Command("terraform", chDir, "show", "-json", "tfplan")
	file, err := os.Create(jsonFilePath)
	if err != nil {
		fmt.Printf("Failed to create file: %v", err)
	}
	defer file.Close()

	cmd.Stdout = file
	cmd.Stderr = os.Stderr

	// Run the command
	fmt.Printf("Running Terraform show: %s\n", cmd.String())
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %s\n", err)
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
	fmt.Printf("Running Terraform plan: %s\n", cmd.String())
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %s\n", err)
	}
}

func (*Importer) executeTerraformInit(chDir string) {
	cmd := exec.Command("terraform", chDir, "init", "-upgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	fmt.Printf("Running Terraform init: %s\n", cmd.String())
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %s\n", err)
	}
}

func (importer *Importer) createBackendOverrideFile() (string) {
	backendOverrideFilePath := filepath.Join(importer.TerraformModulePath, "backend_override.tf")
	backendOverrideFile, err := os.Create(backendOverrideFilePath)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer backendOverrideFile.Close()
	_, err = backendOverrideFile.WriteString(fmt.Sprintf("terraform {\n  backend \"local\" {}\n}\n"))
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
	return backendOverrideFilePath
}
