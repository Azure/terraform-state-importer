package csv

import (
	csvwriter "encoding/csv"
	"os"
	"path/filepath"
	"sort"

	issuetypes "github.com/azure/terraform-state-importer/issues"
	"github.com/sirupsen/logrus"
)

type IIssueCsvClient interface {
	Export(issues map[string]issuetypes.Issue)
}

type IssueCsvClient struct {
	WorkingFolderPath string
	IssueCsv          *IssueCsv
	Logger            *logrus.Logger
}

type IssueCsv struct {
	Header []string
	Rows   []*IssueCsvRow
}

func NewIssueCsvClient(workingFolderPath string, logger *logrus.Logger) *IssueCsvClient {
	return &IssueCsvClient{
		WorkingFolderPath: workingFolderPath,
		IssueCsv:          &IssueCsv{Header: []string{"Issue ID", "Issue Type", "Resource Address", "Resource Name", "Resource Type", "Resource Location", "Mapped Resource ID", "Action", "Action ID"}},
		Logger:            logger,
	}
}

type ActionType string

const (
	ActionTypeNone    ActionType = ""
	ActionTypeUse     ActionType = "use"
	ActionTypeIgnore  ActionType = "ignore"
	ActionTypeReplace ActionType = "replace"
)

func (csv *IssueCsv) AddRow(row *IssueCsvRow) {
	csv.Rows = append(csv.Rows, row)
}

type IssueCsvRow struct {
	IssueID          string
	IssueType        issuetypes.IssueType
	ResourceAddress  string
	ResourceName     string
	ResourceType     string
	ResourceLocation string
	MappedResourceID string
	Action           ActionType
	ActionID         string
}

func (csvClient *IssueCsvClient) Export(issues map[string]issuetypes.Issue) {

	for id, issue := range issues {
		resourceAddress := issue.ResourceAddress
		resourceName := issue.ResourceName
		resourceType := issue.ResourceType
		resourceLocation := issue.ResourceLocation

		if issue.IssueType == issuetypes.IssueTypeMultipleResourceIDs {
			for _, mappedResource := range issue.MappedResourceIDs {
				csvRow := IssueCsvRow{
					IssueID:          id,
					IssueType:        issue.IssueType,
					ResourceAddress:  resourceAddress,
					ResourceName:     resourceName,
					ResourceType:     resourceType,
					ResourceLocation: resourceLocation,
					MappedResourceID: mappedResource,
					Action:           ActionTypeNone,
					ActionID:         "",
				}
				csvClient.IssueCsv.AddRow(&csvRow)
			}
		} else {
			csvRow := IssueCsvRow{
				IssueID:          id,
				IssueType:        issue.IssueType,
				ResourceAddress:  resourceAddress,
				ResourceName:     resourceName,
				ResourceType:     resourceType,
				ResourceLocation: resourceLocation,
				MappedResourceID: "",
				Action:           ActionTypeNone,
				ActionID:         "",
			}
			csvClient.IssueCsv.AddRow(&csvRow)
		}
	}

	sort.Sort(ByIssueTypeAddressResourceTypeAndMappedId(csvClient.IssueCsv.Rows))

	csvClient.writeCsv()
}

func (csvClient *IssueCsvClient) writeCsv() {
	csvData := [][]string{csvClient.IssueCsv.Header}
	for _, issue := range csvClient.IssueCsv.Rows {
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

	csvFilePath := filepath.Join(csvClient.WorkingFolderPath, "issues.csv")
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		csvClient.Logger.Fatalf("Failed to create file: %v", err)
	}
	defer csvFile.Close()
	csvWriter := csvwriter.NewWriter(csvFile)
	defer csvWriter.Flush()
	err = csvWriter.WriteAll(csvData)
	if err != nil {
		csvClient.Logger.Fatalf("Failed to write CSV file: %v", err)
	}
	csvClient.Logger.Infof("Issues written to %s", csvFilePath)
}

type ByIssueTypeAddressResourceTypeAndMappedId []*IssueCsvRow

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
