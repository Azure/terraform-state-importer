package analyzer

import (
	"fmt"
	"testing"

	"github.com/azure/terraform-state-importer/types"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

type mockResourceGraphClient struct {
	Resources []*types.GraphResource
	Err       error
	Called    bool
}

func (m *mockResourceGraphClient) GetResources() ([]*types.GraphResource, error) {
	m.Called = true
	return m.Resources, m.Err
}

type mockPlanClient struct {
	Resources []*types.PlanResource
	Called    bool
}

func (m *mockPlanClient) PlanAndGetResources() []*types.PlanResource {
	m.Called = true
	return m.Resources
}

func (m *mockPlanClient) PlanAsText() {
	m.Called = true
}

type mockJsonClient struct {
	Called    bool
	Resources map[string]any
}

func (m *mockJsonClient) Export(resources any, fileName string) {
	m.Called = true
}

func (m *mockJsonClient) Import(fileName string) map[string]any {
	m.Called = true
	return m.Resources
}

type mockIssueCsvClient struct {
	Issues *map[string]types.Issue
	Called bool
}

func (m *mockIssueCsvClient) Export(issues map[string]types.Issue) {
	m.Issues = &issues
	m.Called = true
}

func (m *mockIssueCsvClient) Import() (*map[string]types.Issue, error) {
	m.Called = true
	return m.Issues, nil
}

type mockHclClient struct {
	Called bool
}

func (m *mockHclClient) Export(importBlocks []types.ImportBlock, fileName string) {
	m.Called = true
}

func TestMappingClient_Map_WithNoIssues(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{{ID: "1", Name: "res1", Type: "type1", Location: "eastus"}}
	planResources := []*types.PlanResource{{
		Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus", ResourceNameMatchType: types.NameMatchTypeExact,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.False(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
	assert.True(t, mappingClient.HclClient.(*mockHclClient).Called)
}

func TestMappingClient_Map_WithIssues(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{{ID: "1", Name: "res1", Type: "type1", Location: "eastus"}}
	planResources := []*types.PlanResource{{
		Address: "addr1", ResourceName: "res2", Type: "type1", Location: "eastus", ResourceNameMatchType: types.NameMatchTypeExact,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.True(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
	assert.False(t, mappingClient.HclClient.(*mockHclClient).Called)

	assert.Equal(t, len(*mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues), 2)

	graphResourceIssueId := getIdentityHash(graphResources[0].ID)
	planResourceIssueId := getIdentityHash(planResources[0].Address)

	assert.Equal(t, (*mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues)[planResourceIssueId].IssueType, types.IssueTypeNoResourceID)
	assert.Equal(t, (*mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues)[graphResourceIssueId].IssueType, types.IssueTypeUnusedResourceID)
}

func TestMappingClient_Map_WithErrorFromResourceGraphClient(t *testing.T) {
	logger := logrus.New()
	// Setting output to discard to avoid panic stack trace in test output
	logger.SetOutput(nil)

	graphClient := &mockResourceGraphClient{
		Err: fmt.Errorf("failed to connect to resource graph"),
	}

	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: graphClient,
		PlanClient:          &mockPlanClient{},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	// This should panic with a log.Fatalf, we're using recover to catch it
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when ResourceGraphClient returns error, but got none")
		}
		assert.True(t, graphClient.Called)
	}()

	mappingClient.Map()
}

func TestMappingClient_Map_WithIDContainsMatch(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{{ID: "/subscriptions/123/resourcegroups/rg1/providers/type1/res1", Name: "res1", Type: "type1", Location: "eastus"}}
	planResources := []*types.PlanResource{{
		Address: "addr1", ResourceName: "rg1", Type: "type1", Location: "eastus", ResourceNameMatchType: types.NameMatchTypeIDContains,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.False(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
}

func TestMappingClient_Map_WithMultipleResourcesSameLocation(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
		{ID: "2", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{{
		Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus", ResourceNameMatchType: types.NameMatchTypeExact,
	}}
	issueMap := make(map[string]types.Issue)
	issueClient := &mockIssueCsvClient{
		Issues: &issueMap,
	}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      issueClient,
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.True(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)

	// Verify we have a MultipleResourceIDs issue
	hasMultipleResourceIDIssue := false
	for _, issue := range *issueClient.Issues {
		if issue.IssueType == types.IssueTypeMultipleResourceIDs {
			hasMultipleResourceIDIssue = true
			break
		}
	}
	assert.True(t, hasMultipleResourceIDIssue)
}

func TestMappingClient_Map_WithEmptyResources(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{}
	planResources := []*types.PlanResource{}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.False(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
}

func TestMappingClient_Map_LocationBasedFiltering(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
		{ID: "2", Name: "res1", Type: "type1", Location: "westus"},
	}
	planResources := []*types.PlanResource{{
		Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus", ResourceNameMatchType: types.NameMatchTypeExact,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.False(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
}

func TestMappingClient_Map_WithResourceIDContainedInLocation(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1/eastus/resource", Name: "res1", Type: "type1", Location: "somewhere"},
		{ID: "2/westus/resource", Name: "res1", Type: "type1", Location: "somewhere"},
	}
	planResources := []*types.PlanResource{{
		Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus", ResourceNameMatchType: types.NameMatchTypeExact,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		HclClient:           &mockHclClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.False(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
}
