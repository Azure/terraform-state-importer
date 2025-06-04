package analyzer

import (
	"crypto/sha256"

	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/azure/terraform-state-importer/azure"
	"github.com/azure/terraform-state-importer/csv"
	"github.com/azure/terraform-state-importer/hcl"
	"github.com/azure/terraform-state-importer/json"
	"github.com/azure/terraform-state-importer/terraform"
	"github.com/azure/terraform-state-importer/types"
)

type MappingClient struct {
	WorkingFolderPath   string
	HasInputCsv         bool
	ResourceGraphClient azure.IResourceGraphClient
	PlanClient          terraform.IPlanClient
	IssueCsvClient      csv.IIssueCsvClient
	JsonClient          json.IJsonClient
	HclClient           hcl.IHclClient
	Logger              *logrus.Logger
}

func NewMappingClient(workingFolderPath string, hasInputCsv bool, resourceGraphClient azure.IResourceGraphClient, planClient terraform.IPlanClient, issueCsvClient csv.IIssueCsvClient, jsonClient json.IJsonClient, hclClient hcl.IHclClient, logger *logrus.Logger) *MappingClient {
	return &MappingClient{
		WorkingFolderPath:   workingFolderPath,
		HasInputCsv:         hasInputCsv,
		ResourceGraphClient: resourceGraphClient,
		PlanClient:          planClient,
		IssueCsvClient:      issueCsvClient,
		JsonClient:          jsonClient,
		HclClient:           hclClient,
		Logger:              logger,
	}
}

func (mappingClient *MappingClient) Map() {
	resolvedIssues := mappingClient.getResolvedIssues()

	graphResources, err := mappingClient.ResourceGraphClient.GetResources()
	if err != nil {
		mappingClient.Logger.Fatalf("Error getting resources from Resource Graph: %v", err)
	}

	importsFileName := "imports.tf"
	destroyFileName := "destroy.tf"

	mappingClient.HclClient.CleanFiles([]string{importsFileName, destroyFileName})

	planResources := mappingClient.PlanClient.PlanAndGetResources()

	finalMappedResources, issues := mappingClient.mapResourcesFromGraphToPlan(graphResources, planResources, resolvedIssues)

	mappingClient.JsonClient.Export(issues, "issues.json")
	mappingClient.JsonClient.Export(planResources, "resources.json")

	if len(issues) > 0 {
		mappingClient.Logger.Warnf("Found %d issues based on the Terraform Plan and Resource Graph Queries", len(issues))
		mappingClient.IssueCsvClient.Export(issues)
		return
	} else {
		mappingClient.Logger.Info("No issues found based on the Terraform Plan and Resource Graph Queries")
		mappingClient.JsonClient.Export(finalMappedResources, "final.json")
	}

	importBlocks := []types.ImportBlock{}
	destroyBlocks := []types.DestroyBlock{}
	for _, finalMappedResource := range finalMappedResources {
		if finalMappedResource.ActionType == types.ActionTypeUse && finalMappedResource.Type == types.MappedResourceTypeTerraform {
			resourceID := finalMappedResource.ResourceID
			if finalMappedResource.ResourceAPIVersion != "" {
				resourceID = fmt.Sprintf("%s?api-version=%s", resourceID, finalMappedResource.ResourceAPIVersion)
			}

			importBlock := types.ImportBlock{
				To: finalMappedResource.ResourceAddress,
				ID: resourceID,
			}
			importBlocks = append(importBlocks, importBlock)
		}
		if (finalMappedResource.ActionType == types.ActionTypeReplace || finalMappedResource.ActionType == types.ActionTypeDestroy) && finalMappedResource.Type == types.MappedResourceTypeGraph {
			resourceID := finalMappedResource.ResourceID
			destroyBlock := types.DestroyBlock{
				ID:   resourceID,
				Type: finalMappedResource.ResourceType,
			}
			destroyBlocks = append(destroyBlocks, destroyBlock)
		}
	}

	mappingClient.HclClient.WriteImportBlocks(importBlocks, importsFileName)
	mappingClient.HclClient.WriteDestroyBlocks(destroyBlocks, destroyFileName)
}

func (mappingClient *MappingClient) getResolvedIssues() *map[string]types.Issue {
	if mappingClient.HasInputCsv {
		mappingClient.Logger.Info("Importing issues from supplied CSV file")
		resolvedIssues, err := mappingClient.IssueCsvClient.Import()
		if err != nil {
			mappingClient.Logger.Fatalf("Error importing issues from CSV: %v", err)
		}
		mappingClient.Logger.Infof("Imported %d resolved issues from CSV file", len(*resolvedIssues))
		return resolvedIssues
	}
	return nil
}

func (importer *MappingClient) mapResourcesFromGraphToPlan(graphResources []*types.GraphResource, planResources []*types.PlanResource, resolvedIssues *map[string]types.Issue) ([]types.MappedResource, map[string]types.Issue) {
	finalMappedResources := []types.MappedResource{}
	issues := map[string]types.Issue{}
	uniqueUsedResources := make(map[string]*types.GraphResource)

	for _, resource := range planResources {
		finalMappedResource := types.MappedResource{
			Type:               types.MappedResourceTypeTerraform,
			ResourceAddress:    resource.Address,
			ResourceAPIVersion: resource.APIVersion,
			ResourceType:       resource.Type,
		}

		for _, graphResource := range graphResources {
			if resource.ResourceNameMatchType == types.NameMatchTypeExact && strings.ToLower(graphResource.Name) == strings.ToLower(resource.ResourceName) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
			}

			if resource.ResourceNameMatchType == types.NameMatchTypeIDContains && strings.Contains(strings.ToLower(graphResource.ID), strings.ToLower(resource.ResourceName)) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
			}

			if resource.ResourceNameMatchType == types.NameMatchTypeIDEndsWith && strings.HasSuffix(strings.ToLower(graphResource.ID), strings.ToLower(resource.ResourceName)) {
				resource.MappedResources = append(resource.MappedResources, graphResource)
			}
		}

		hadIssue := false

		if len(resource.MappedResources) == 0 {
			hadIssue = true
			issue := IssueFromPlanResource(resource)

			resolved := false
			if importer.HasInputCsv {
				if resolvedIssue, exists := (*resolvedIssues)[getIdentityHash(resource.Address)]; exists {
					if resolvedIssue.Resolution.ActionType == types.ActionTypeIgnore {
						importer.Logger.Debugf("Ignoring Issue ID: %s, Action: %s", resolvedIssue.IssueID, resolvedIssue.Resolution.ActionType)
						finalMappedResource.IssueType = types.IssueTypeNoResourceID
						finalMappedResource.ActionType = types.ActionTypeIgnore
						resolved = true
					}
					if resolvedIssue.Resolution.ActionType == types.ActionTypeReplace {
						matchedGraphResourceID := (*resolvedIssues)[resolvedIssue.Resolution.ActionID].ResourceAddress
						for _, graphResource := range graphResources {
							if strings.Contains(strings.ToLower(graphResource.ID), strings.ToLower(matchedGraphResourceID)) {
								resource.MappedResources = []*types.GraphResource{graphResource}
								finalMappedResource.ResourceID = graphResource.ID
								finalMappedResource.IssueType = types.IssueTypeNoResourceID
								finalMappedResource.ActionType = types.ActionTypeReplace
								resolved = true
								break
							}
						}
					}
				} else {
					importer.Logger.Fatalf("Error: No matching issue resolution found for Issue ID, check your CSV file and try again: %s Name: %s, Type: %s, Address: %s", issue.IssueID, resource.ResourceName, resource.Type, resource.Address)
				}

			}

			if !resolved {
				importer.Logger.Warnf("No matching resource ID found for Name: %s, Type: %s, Address: %s", resource.ResourceName, resource.Type, resource.Address)
				addIssue(issues, issue, types.IssueTypeNoResourceID)
			} else {
				finalMappedResources = append(finalMappedResources, finalMappedResource)
			}
			continue
		}

		for _, mappedResource := range resource.MappedResources {
			if _, exists := uniqueUsedResources[mappedResource.ID]; !exists {
				uniqueUsedResources[mappedResource.ID] = mappedResource
			}
		}

		if len(resource.MappedResources) > 1 {
			mappedResourceIDsBasedOnLocation := []*types.GraphResource{}

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
				hadIssue = true
				issue := IssueFromPlanResource(resource)
				resolved := false
				if importer.HasInputCsv {
					if resolvedIssue, exists := (*resolvedIssues)[getIdentityHash(resource.Address)]; exists {
						for _, mappedResource := range resource.MappedResources {
							if strings.Contains(strings.ToLower(mappedResource.ID), strings.ToLower(resolvedIssue.MappedResourceIDs[0])) {
								resource.MappedResources = []*types.GraphResource{mappedResource}
								finalMappedResource.ResourceID = mappedResource.ID
								finalMappedResource.IssueType = types.IssueTypeMultipleResourceIDs
								finalMappedResource.ActionType = types.ActionTypeUse
								resolved = true
								break
							}
						}
					} else {
						importer.Logger.Fatalf("Error: No matching issue resolution found for Issue ID, check your CSV file and try again: %s Name: %s, Type: %s, Address: %s", issue.IssueID, resource.ResourceName, resource.Type, resource.Address)
					}
				}

				if !resolved {
					importer.Logger.Warnf("More than 1 Resource ID has been matched for Name: %s, Type: %s, Address: %s", resource.ResourceName, resource.Type, resource.Address)
					addIssue(issues, issue, types.IssueTypeMultipleResourceIDs)
				} else {
					finalMappedResources = append(finalMappedResources, finalMappedResource)
				}
			}
		}
		if !hadIssue {
			finalMappedResource.ResourceID = resource.MappedResources[0].ID
			finalMappedResource.IssueType = types.IssueTypeNone
			finalMappedResource.ActionType = types.ActionTypeUse
			finalMappedResources = append(finalMappedResources, finalMappedResource)
		}
	}

	for _, graphResource := range graphResources {
		if _, exists := uniqueUsedResources[graphResource.ID]; !exists {
			finalMappedResource := types.MappedResource{
				Type:            types.MappedResourceTypeGraph,
				ResourceAddress: "",
				ResourceID:      graphResource.ID,
				ResourceType:    graphResource.Type,
				IssueType:       types.IssueTypeUnusedResourceID,
			}

			issue := IssueFromGraphResource(graphResource)
			resolved := false
			if importer.HasInputCsv {
				if resolvedIssue, exists := (*resolvedIssues)[issue.IssueID]; exists {
					if resolvedIssue.Resolution.ActionType == types.ActionTypeIgnore {
						importer.Logger.Debugf("Ignoring Issue ID: %s, Action: %s", resolvedIssue.IssueID, resolvedIssue.Resolution.ActionType)
						finalMappedResource.ActionType = resolvedIssue.Resolution.ActionType
						resolved = true
					}
					if resolvedIssue.Resolution.ActionType == types.ActionTypeReplace || resolvedIssue.Resolution.ActionType == types.ActionTypeDestroy {
						importer.Logger.Debugf("Destroying via Replace or Destroy Issue ID: %s, Action: %s", resolvedIssue.IssueID, resolvedIssue.Resolution.ActionType)
						finalMappedResource.ActionType = resolvedIssue.Resolution.ActionType
						resolved = true
					}
				} else {
					importer.Logger.Fatalf("Error: No matching issue resolution found for Issue ID, check your CSV file and try again: %s Name: %s, Type: %s, Address: %s", issue.IssueID, graphResource.Name, graphResource.Type, graphResource.ID)
				}
			}

			if !resolved {
				importer.Logger.Warnf("Resource ID %s is not used in the Terraform plan", graphResource.ID)
				addIssue(issues, IssueFromGraphResource(graphResource), types.IssueTypeUnusedResourceID)
			} else {
				finalMappedResources = append(finalMappedResources, finalMappedResource)
			}
		}
	}
	return finalMappedResources, issues
}

func addIssue(issues map[string]types.Issue, issue types.Issue, issueType types.IssueType) {
	issue.IssueType = issueType
	issues[issue.IssueID] = issue
}

func IssueFromGraphResource(graphResource *types.GraphResource) types.Issue {
	issue := types.Issue{}
	issue.IssueID = getIdentityHash(graphResource.ID)
	issue.ResourceAddress = graphResource.ID
	issue.ResourceName = graphResource.Name
	issue.ResourceType = graphResource.Type
	issue.ResourceLocation = graphResource.Location
	issue.MappedResourceIDs = []string{graphResource.ID}

	return issue
}

func IssueFromPlanResource(planResource *types.PlanResource) types.Issue {
	issue := types.Issue{}
	issue.IssueID = getIdentityHash(planResource.Address)
	issue.ResourceAddress = planResource.Address
	issue.ResourceName = planResource.ResourceName
	issue.ResourceType = planResource.Type
	issue.ResourceSubType = planResource.SubType
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
