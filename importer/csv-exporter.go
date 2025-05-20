package importer

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
)

type IssueCsvRow struct {
	IssueID          string
	IssueType        IssueType
	ResourceAddress  string
	ResourceName     string
	ResourceType     string
	ResourceLocation string
	MappedResourceID string
	Action           ActionType
	ActionID         string
}

func (importer *Importer) exportIssuesToCsv(issues map[string]ResourceIssue) {
	issueCsvData := []IssueCsvRow{}

	for id, issue := range issues {
		resourceAddress := issue.Resource.Address
		resourceName := issue.Resource.ResourceName
		resourceType := issue.Resource.Type
		resourceLocation := issue.Resource.Location

		if issue.IssueType == IssueTypeUnusedResourceID {
			resourceAddress = issue.Resource.MappedResources[0].ID
			resourceName = issue.Resource.MappedResources[0].Name
			resourceType = issue.Resource.MappedResources[0].Type
			resourceLocation = issue.Resource.MappedResources[0].Location
		}

		if issue.IssueType == IssueTypeMultipleResourceIDs {
			for _, mappedResource := range issue.Resource.MappedResources {
				issueCsvData = append(issueCsvData, IssueCsvRow{
					IssueID:          id,
					IssueType:        issue.IssueType,
					ResourceAddress:  resourceAddress,
					ResourceName:     resourceName,
					ResourceType:     resourceType,
					ResourceLocation: resourceLocation,
					MappedResourceID: mappedResource.ID,
					Action:           ActionTypeNone,
					ActionID:         "",
				})
			}
		} else {
			issueCsvData = append(issueCsvData, IssueCsvRow{
				IssueID:          id,
				IssueType:        issue.IssueType,
				ResourceAddress:  resourceAddress,
				ResourceName:     resourceName,
				ResourceType:     resourceType,
				ResourceLocation: resourceLocation,
				MappedResourceID: "",
				Action:           ActionTypeNone,
				ActionID:         "",
			})
		}
	}

	sort.Sort(ByIssueTypeAddressResourceTypeAndMappedId(issueCsvData))

	csvData := [][]string{{"Issue ID", "Issue Type", "Resource Address", "Resource Name", "Resource Type", "Resource Location", "Mapped Resource ID", "Action", "Action ID"}}
	for _, issue := range issueCsvData {
		csvData = append(csvData, []string{
			issue.IssueID,
			string(issue.IssueType),
			issue.ResourceAddress,
			issue.ResourceName,
			issue.ResourceType,
			issue.ResourceLocation,
			issue.MappedResourceID,
			string(issue.Action),
			issue.ActionID,
		})
	}

	csvFilePath := filepath.Join(importer.TerraformModulePath, "issues.csv")
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		importer.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer csvFile.Close()
	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()
	err = csvWriter.WriteAll(csvData)
	if err != nil {
		importer.Logger.Fatalf("Failed to write CSV file: %v", err)
	}
	importer.Logger.Infof("Issues written to %s", csvFilePath)
}

type ByIssueTypeAddressResourceTypeAndMappedId []IssueCsvRow

func (o ByIssueTypeAddressResourceTypeAndMappedId) Len() int      { return len(o) }
func (o ByIssueTypeAddressResourceTypeAndMappedId) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ByIssueTypeAddressResourceTypeAndMappedId) Less(i, j int) bool {
	if o[i].IssueType != o[j].IssueType {
		return o[i].IssueType < o[j].IssueType
	}

	if o[i].ResourceType != o[j].ResourceType {
		return o[i].ResourceType < o[j].ResourceType
	}

	if o[i].ResourceAddress != o[j].ResourceAddress {
		return o[i].ResourceAddress < o[j].ResourceAddress
	}

	return o[i].MappedResourceID < o[j].MappedResourceID
}
