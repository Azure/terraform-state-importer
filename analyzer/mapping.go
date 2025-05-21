package analyzer

import (
	"crypto/sha256"

	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/azure"
	"github.com/azure/terraform-state-importer/csv"
	issuetypes "github.com/azure/terraform-state-importer/issues"
	"github.com/azure/terraform-state-importer/json"
	"github.com/azure/terraform-state-importer/terraform"
)

type MappingClient struct {
	WorkingFolderPath   string
	ResourceGraphClient azure.IResourceGraphClient
	PlanClient          terraform.IPlanClient
	IssueCsvClient      csv.IIssueCsvClient
	JsonClient          json.IJsonClient
	Logger              *logrus.Logger
}

func NewMappingClient(workingFolderPath string, resourceGraphClient azure.IResourceGraphClient, planClient terraform.IPlanClient, issueCsvClient csv.IIssueCsvClient, jsonClient json.IJsonClient, logger *logrus.Logger) *MappingClient {
	return &MappingClient{
		WorkingFolderPath:   workingFolderPath,
		ResourceGraphClient: resourceGraphClient,
		PlanClient:          planClient,
		IssueCsvClient:      issueCsvClient,
		JsonClient:          jsonClient,
		Logger:              logger,
	}
}

func (mappingClient *MappingClient) Map() {
	graphResources, err := mappingClient.ResourceGraphClient.GetResources()
	if err != nil {
		mappingClient.Logger.Fatalf("Error getting resources from Resource Graph: %v", err)
	}
	planResources := mappingClient.PlanClient.PlanAndGetResources()
	planResources, issues := mappingClient.mapResourcesFromGraphToPlan(graphResources, planResources)

	mappingClient.JsonClient.Export(issues, "issues.json")
	mappingClient.JsonClient.Export(planResources, "resources.json")

	if len(issues) > 0 {
		mappingClient.Logger.Warnf("Found %d issues based on the Terraform Plan and Resource Graph Queries", len(issues))
		mappingClient.IssueCsvClient.Export(issues)
	} else {
		mappingClient.Logger.Info("No issues found based on the Terraform Plan and Resource Graph Queries")
	}
}

func (importer *MappingClient) mapResourcesFromGraphToPlan(graphResources []azure.GraphResource, planResources []terraform.PlanResource) ([]terraform.PlanResource, map[string]issuetypes.Issue) {
	issues := map[string]issuetypes.Issue{}
	uniqueUsedResources := make(map[string]azure.GraphResource)

	for _, resource := range planResources {
		for _, graphResource := range graphResources {
			if resource.ResourceNameMatchType == terraform.NameMatchTypeExact && strings.ToLower(graphResource.Name) == strings.ToLower(resource.ResourceName) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
			}

			if resource.ResourceNameMatchType == terraform.NameMatchTypeIDContains && strings.Contains(strings.ToLower(graphResource.ID), strings.ToLower(resource.ResourceName)) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
			}
		}

		if len(resource.MappedResources) == 0 {
			importer.Logger.Warnf("No matching resource ID found for Name: %s, Type: %s, Address: %s", resource.ResourceName, resource.Type, resource.Address)
			addIssue(issues, IssueFromPlanResource(resource), issuetypes.IssueTypeNoResourceID)
			continue
		}

		for _, mappedResource := range resource.MappedResources {
			if _, exists := uniqueUsedResources[mappedResource.ID]; !exists {
				uniqueUsedResources[mappedResource.ID] = mappedResource
			}
		}

		if len(resource.MappedResources) > 1 {
			mappedResourceIDsBasedOnLocation := []azure.GraphResource{}

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
				importer.Logger.Warnf("More than 1 Resource ID has been matched for Name: %s, Type: %s, Address: %s", resource.ResourceName, resource.Type, resource.Address)
				addIssue(issues, IssueFromPlanResource(resource), issuetypes.IssueTypeMultipleResourceIDs)
			}
		}
	}

	for _, graphResource := range graphResources {
		if _, exists := uniqueUsedResources[graphResource.ID]; !exists {
			importer.Logger.Warnf("Resource ID %s is not used in the Terraform plan", graphResource.ID)
			addIssue(issues, IssueFromGraphResource(graphResource), issuetypes.IssueTypeUnusedResourceID)
		}
	}
	return planResources, issues
}

func addIssue(issues map[string]issuetypes.Issue, issue issuetypes.Issue, issueType issuetypes.IssueType) {
	issue.IssueType = issueType
	issues[issue.IssueID] = issue
}

func IssueFromGraphResource(graphResource azure.GraphResource) issuetypes.Issue {
	issue := issuetypes.Issue{}
	issue.IssueID = getIdentityHash(graphResource.ID)
	issue.ResourceAddress = graphResource.ID
	issue.ResourceName = graphResource.Name
	issue.ResourceType = graphResource.Type
	issue.ResourceLocation = graphResource.Location
	issue.MappedResourceIDs = []string{graphResource.ID}

	return issue
}

func IssueFromPlanResource(planResource terraform.PlanResource) issuetypes.Issue {
	issue := issuetypes.Issue{}
	issue.IssueID = getIdentityHash(planResource.Address)
	issue.ResourceAddress = planResource.Address
	issue.ResourceName = planResource.ResourceName
	issue.ResourceType = planResource.Type
	issue.ResourceLocation = planResource.Location
	issue.MappedResourceIDs = []string{}

	for _, mappedResource := range planResource.MappedResources {
		issue.MappedResourceIDs = append(issue.MappedResourceIDs, mappedResource.ID)
	}

	return issue
}

func getIdentityHash(id string) string {
	sha256ID := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", sha256ID)[0:7]
}
