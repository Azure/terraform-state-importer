package analyzer

import (
	"reflect"
	"testing"

	"github.com/azure/terraform-state-importer/azure"
	"github.com/azure/terraform-state-importer/issues"
	"github.com/azure/terraform-state-importer/terraform"
	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
)

type mockResourceGraphClient struct {
	Resources []azure.GraphResource
	Err       error
	Called    bool
}

func (m *mockResourceGraphClient) GetResources() ([]azure.GraphResource, error) {
	m.Called = true
	return m.Resources, m.Err
}

type mockPlanClient struct {
	Resources []terraform.PlanResource
	Called    bool
}

func (m *mockPlanClient) PlanAndGetResources() []terraform.PlanResource {
	m.Called = true
	return m.Resources
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
	Issues map[string]issues.Issue
	Called bool
}

func (m *mockIssueCsvClient) Export(issues map[string]issues.Issue) {
	m.Issues = issues
	m.Called = true
}

func TestMappingClient_Map_WithNoIssues(t *testing.T) {
	logger := logrus.New()
	graphResources := []azure.GraphResource{{ID: "1", Name: "res1", Type: "type1", Location: "eastus"}}
	planResources := []terraform.PlanResource{{
		Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus", ResourceNameMatchType: terraform.NameMatchTypeExact,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.False(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)
}

func TestMappingClient_Map_WithIssues(t *testing.T) {
	logger := logrus.New()
	graphResources := []azure.GraphResource{{ID: "1", Name: "res1", Type: "type1", Location: "eastus"}}
	planResources := []terraform.PlanResource{{
		Address: "addr1", ResourceName: "res2", Type: "type1", Location: "eastus", ResourceNameMatchType: terraform.NameMatchTypeExact,
	}}
	mappingClient := &MappingClient{
		WorkingFolderPath:   ".",
		ResourceGraphClient: &mockResourceGraphClient{Resources: graphResources},
		PlanClient:          &mockPlanClient{Resources: planResources},
		IssueCsvClient:      &mockIssueCsvClient{},
		JsonClient:          &mockJsonClient{},
		Logger:              logger,
	}

	mappingClient.Map()

	assert.True(t, mappingClient.ResourceGraphClient.(*mockResourceGraphClient).Called)
	assert.True(t, mappingClient.PlanClient.(*mockPlanClient).Called)
	assert.True(t, mappingClient.JsonClient.(*mockJsonClient).Called)
	assert.True(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Called)

	assert.Equal(t, len(mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues), 2)
	keys := reflect.ValueOf(mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues).MapKeys()
	assert.Equal(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues[keys[0].String()].IssueType, issues.IssueTypeNoResourceID)
	assert.Equal(t, mappingClient.IssueCsvClient.(*mockIssueCsvClient).Issues[keys[1].String()].IssueType, issues.IssueTypeUnusedResourceID)
}
