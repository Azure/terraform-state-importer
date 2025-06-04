package csv

import (
	csvwriter "encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/azure/terraform-state-importer/types"
	"github.com/sirupsen/logrus"
)

type IIssueCsvClient interface {
	Export(issues map[string]types.Issue)
	Import() (*map[string]types.Issue, error)
}

type IssueCsvClient struct {
	WorkingFolderPath string
	IssueCsvPath      string
	IssueCsv          *IssueCsv
	Logger            *logrus.Logger
}

type IssueCsv struct {
	Header []string
	Rows   []*IssueCsvRow
}

func NewIssueCsvClient(workingFolderPath string, issueCsvPath string, logger *logrus.Logger) *IssueCsvClient {
	return &IssueCsvClient{
		WorkingFolderPath: workingFolderPath,
		IssueCsvPath:      issueCsvPath,
		IssueCsv:          &IssueCsv{Header: []string{"Issue ID", "Issue Type", "Resource Address", "Resource Name", "Resource Type", "Resource Sub Type", "Resource Location", "Mapped Resource ID", "Action", "Action ID"}},
		Logger:            logger,
	}
}

func (csv *IssueCsv) AddRow(row *IssueCsvRow) {
	csv.Rows = append(csv.Rows, row)
}

type IssueCsvRow struct {
	IssueID          string
	IssueType        types.IssueType
	ResourceAddress  string
	ResourceName     string
	ResourceType     string
	ResourceSubType  string
	ResourceLocation string
	MappedResourceID string
	Action           types.ActionType
	ActionID         string
}

func (csvClient *IssueCsvClient) Export(issues map[string]types.Issue) {
	for id, issue := range issues {
		resourceAddress := issue.ResourceAddress
		resourceName := issue.ResourceName
		resourceType := issue.ResourceType
		resourceSubType := issue.ResourceSubType
		resourceLocation := issue.ResourceLocation

		if issue.IssueType == types.IssueTypeMultipleResourceIDs {
			for _, mappedResource := range issue.MappedResourceIDs {
				csvRow := IssueCsvRow{
					IssueID:          id,
					IssueType:        issue.IssueType,
					ResourceAddress:  resourceAddress,
					ResourceName:     resourceName,
					ResourceType:     resourceType,
					ResourceSubType:  resourceSubType,
					ResourceLocation: resourceLocation,
					MappedResourceID: mappedResource,
					Action:           types.ActionTypeNone,
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
				ResourceSubType:  resourceSubType,
				ResourceLocation: resourceLocation,
				MappedResourceID: "",
				Action:           types.ActionTypeNone,
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
			issue.ResourceSubType,
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

func (csvClient *IssueCsvClient) Import() (*map[string]types.Issue, error) {
	csvFile, err := os.Open(csvClient.IssueCsvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer csvFile.Close()

	csvReader := csvwriter.NewReader(csvFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}

	if len(records) < 1 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	header := records[0]
	if !validateHeader(header) {
		return nil, fmt.Errorf("invalid CSV header")
	}

	issues := make(map[string]types.Issue)

	// Get all the issue keys, so we can use them for validation
	for _, record := range records[1:] {
		issueID := record[0]
		if _, ok := issues[issueID]; !ok {
			issues[issueID] = types.Issue{}
		}
	}

	for _, record := range records[1:] {
		if len(record) != len(header) {
			csvClient.Logger.Fatalf("Malformed row in CSV file: %v", record)
		}

		issueAction := types.ActionType(record[8])

		if !issueAction.IsValidActionType() || issueAction == types.ActionTypeNone {
			csvClient.Logger.Fatalf("Action is missing or malformed for Issue ID: %s, Action: %s", record[0], record[8])
		}

		issue := types.Issue{
			IssueID:           record[0],
			IssueType:         types.IssueType(record[1]),
			ResourceAddress:   record[2],
			ResourceName:      record[3],
			ResourceType:      record[4],
			ResourceSubType:   record[5],
			ResourceLocation:  record[7],
			MappedResourceIDs: []string{record[7]},
		}

		switch issue.IssueType {
		case types.IssueTypeMultipleResourceIDs:
			if issueAction != types.ActionTypeIgnore && issueAction != types.ActionTypeUse {
				csvClient.Logger.Fatalf("Action for MultiResourceIDs must be Use or Ignore for Issue ID: %s, Action: %s", issue.IssueID, issueAction)
			}
			if issueAction == types.ActionTypeIgnore {
				csvClient.Logger.Debugf("Ignoring Issue ID: %s, Action: %s", issue.IssueID, issueAction)
				continue
			}
			if issueAction == types.ActionTypeUse {
				if issue, ok := issues[issue.IssueID]; ok {
					if issue.IssueID != "" {
						csvClient.Logger.Fatalf("Duplicate Use Action found for Issue ID %s", issue.IssueID)
					}
				}
				issue.Resolution = types.IssueResolution{
					ActionType: issueAction,
					ActionID:   "",
				}
			}
		case types.IssueTypeNoResourceID:
			if issueAction != types.ActionTypeIgnore && issueAction != types.ActionTypeReplace {
				csvClient.Logger.Fatalf("Action for NoResourceID must be Ignore or Replace for Issue ID: %s, Action: %s", issue.IssueID, issueAction)
			}

			if issueAction == types.ActionTypeIgnore {
				csvClient.Logger.Debugf("Ignoring Issue ID: %s, Action: %s", issue.IssueID, issueAction)
				issue.Resolution = types.IssueResolution{
					ActionType: issueAction,
					ActionID:   "",
				}
			}

			if issueAction == types.ActionTypeReplace {
				actionID := record[9]

				if actionID == "" {
					csvClient.Logger.Fatalf("Action ID is missing for Issue ID: %s, Action: %s", issue.IssueID, issueAction)
				}

				if _, ok := issues[actionID]; !ok {
					csvClient.Logger.Fatalf("Action ID %s not found in CSV file for Issue ID: %s", actionID, issue.IssueID)
				}

				issue.Resolution = types.IssueResolution{
					ActionType: issueAction,
					ActionID:   actionID,
				}
			}
		case types.IssueTypeUnusedResourceID:
			if issueAction != types.ActionTypeIgnore && issueAction != types.ActionTypeReplace && issueAction != types.ActionTypeDestroy {
				csvClient.Logger.Fatalf("Action for UnusedResourceID must be Ignore, Replace, or Destroy for Issue ID: %s, Action: %s", issue.IssueID, issueAction)
			}
			if issueAction == types.ActionTypeIgnore {
				csvClient.Logger.Debugf("Ignoring Issue ID: %s, Action: %s", issue.IssueID, issueAction)
				issue.Resolution = types.IssueResolution{
					ActionType: issueAction,
					ActionID:   "",
				}
			}
			if issueAction == types.ActionTypeReplace || issueAction == types.ActionTypeDestroy {
				csvClient.Logger.Debugf("Destroying via Replace or Destroy Issue ID: %s, Action: %s", issue.IssueID, issueAction)
				issue.Resolution = types.IssueResolution{
					ActionType: issueAction,
					ActionID:   "",
				}
			}
		default:
			csvClient.Logger.Fatalf("Invalid Issue Type: %s for Issue ID: %s", issue.IssueType, issue.IssueID)
		}

		issues[issue.IssueID] = issue
	}

	return &issues, nil
}

func validateHeader(header []string) bool {
	expectedHeader := []string{"Issue ID", "Issue Type", "Resource Address", "Resource Name", "Resource Type", "Resource Sub Type", "Resource Location", "Mapped Resource ID", "Action", "Action ID"}
	if len(header) != len(expectedHeader) {
		return false
	}
	for i, h := range header {
		if h != expectedHeader[i] {
			return false
		}
	}
	return true
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

	if o[i].ResourceSubType != o[j].ResourceSubType {
		return o[i].ResourceSubType < o[j].ResourceSubType
	}

	if o[i].ResourceAddress != o[j].ResourceAddress {
		return o[i].ResourceAddress < o[j].ResourceAddress
	}

	return o[i].MappedResourceID < o[j].MappedResourceID
}
