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
	Called           bool
	CleanFilesCalled bool
}

func (m *mockHclClient) WriteImportBlocks(importBlocks []types.ImportBlock, fileName string) {
	m.Called = true
}

func (m *mockHclClient) WriteDestroyBlocks(destroyBlocks []types.DestroyBlock, fileName string) {
	m.Called = true
}

func (m *mockHclClient) CleanFiles(filesToRemove []string) {
	m.CleanFilesCalled = true
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
	assert.True(t, mappingClient.HclClient.(*mockHclClient).CleanFilesCalled)

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
func Test_mapResourcesFromGraphToPlan_ExactMatch(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeExact,
		},
	}
	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, mapped[0].ResourceID, "1")
	assert.Equal(t, mapped[0].ActionType, types.ActionTypeUse)
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_IDContainsMatch(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "/foo/bar/res1", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "bar", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeIDContains,
		},
	}
	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, mapped[0].ResourceID, "/foo/bar/res1")
	assert.Equal(t, mapped[0].ActionType, types.ActionTypeUse)
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_IDEndsWithMatch(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "/foo/bar/res1", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeIDEndsWith,
		},
	}
	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, mapped[0].ResourceID, "/foo/bar/res1")
	assert.Equal(t, mapped[0].ActionType, types.ActionTypeUse)
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_SubTypePrefilter_AvoidsDefinitions_ForAssignments(t *testing.T) {
	logger := logrus.New()
	// Two resources with same name fragment, different types
	assignment := &types.GraphResource{ID: "/providers/Microsoft.Management/managementGroups/alz-aks-corp/providers/Microsoft.Authorization/policyAssignments/Allow-Vnet-Peering", Name: "Allow-Vnet-Peering", Type: "Microsoft.Authorization/policyAssignments", Location: "n/a"}
	definition := &types.GraphResource{ID: "/providers/Microsoft.Management/managementGroups/alz/providers/Microsoft.Authorization/policyDefinitions/Allow-Vnet-Peering", Name: "Allow-Vnet-Peering", Type: "Microsoft.Authorization/policyDefinitions", Location: "n/a"}

	graphResources := []*types.GraphResource{assignment, definition}
	planResources := []*types.PlanResource{
		{
			Address: "addr1",
			Type:    "azapi_resource",
			SubType: "Microsoft.Authorization/policyAssignments",
			// Match by ID suffix against assignment only
			ResourceName:          "policyAssignments/Allow-Vnet-Peering",
			ResourceNameMatchType: types.NameMatchTypeIDEndsWith,
		},
	}

	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, assignment.ID, mapped[0].ResourceID)
	// We may still report UnusedResourceID for the definition; important is we didn't create MultipleResourceIDs
	for _, iss := range issues {
		assert.NotEqual(t, types.IssueTypeMultipleResourceIDs, iss.IssueType)
	}
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_SubTypePrefilter_DefinitionVsSetDefinition(t *testing.T) {
	logger := logrus.New()
	definition := &types.GraphResource{ID: "/providers/Microsoft.Management/managementGroups/alz/providers/Microsoft.Authorization/policyDefinitions/Enforce-Tag", Name: "Enforce-Tag", Type: "Microsoft.Authorization/policyDefinitions", Location: "n/a"}
	setdef := &types.GraphResource{ID: "/providers/Microsoft.Management/managementGroups/alz/providers/Microsoft.Authorization/policySetDefinitions/Enforce-Tag", Name: "Enforce-Tag", Type: "Microsoft.Authorization/policySetDefinitions", Location: "n/a"}

	graphResources := []*types.GraphResource{definition, setdef}
	planResources := []*types.PlanResource{
		{
			Address:               "addr1",
			Type:                  "azapi_resource",
			SubType:               "Microsoft.Authorization/policyDefinitions",
			ResourceName:          "policyDefinitions/Enforce-Tag",
			ResourceNameMatchType: types.NameMatchTypeIDEndsWith,
		},
	}

	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, definition.ID, mapped[0].ResourceID)
	for _, iss := range issues {
		assert.NotEqual(t, types.IssueTypeMultipleResourceIDs, iss.IssueType)
	}
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_IDExact_SingleMatch(t *testing.T) {
	logger := logrus.New()
	id := "/providers/Microsoft.Management/managementGroups/alz-aks-public/providers/Microsoft.Authorization/policyAssignments/Allow-Vnet-Peering"
	assignment := &types.GraphResource{ID: id, Name: "Allow-Vnet-Peering", Type: "Microsoft.Authorization/policyAssignments", Location: "n/a"}
	other := &types.GraphResource{ID: "/providers/Microsoft.Management/managementGroups/alz-aks-corp/providers/Microsoft.Authorization/policyAssignments/Allow-Vnet-Peering", Name: "Allow-Vnet-Peering", Type: "Microsoft.Authorization/policyAssignments", Location: "n/a"}

	graphResources := []*types.GraphResource{assignment, other}
	planResources := []*types.PlanResource{
		{
			Address:               "addr1",
			Type:                  "azapi_resource",
			SubType:               "Microsoft.Authorization/policyAssignments",
			ResourceName:          id,
			ResourceNameMatchType: types.NameMatchTypeIDExact,
		},
	}

	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, id, mapped[0].ResourceID)
	for _, iss := range issues {
		assert.NotEqual(t, types.IssueTypeMultipleResourceIDs, iss.IssueType)
	}
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_IDExact_SingleMatch_Definition(t *testing.T) {
	logger := logrus.New()
	id := "/providers/Microsoft.Management/managementGroups/alz/providers/Microsoft.Authorization/policyDefinitions/Enforce-Tag"
	def := &types.GraphResource{ID: id, Name: "Enforce-Tag", Type: "Microsoft.Authorization/policyDefinitions", Location: "n/a"}
	other := &types.GraphResource{ID: "/providers/Microsoft.Management/managementGroups/alz/providers/Microsoft.Authorization/policyDefinitions/Other", Name: "Other", Type: "Microsoft.Authorization/policyDefinitions", Location: "n/a"}

	graphResources := []*types.GraphResource{def, other}
	planResources := []*types.PlanResource{
		{
			Address:               "addr1",
			Type:                  "azapi_resource",
			SubType:               "Microsoft.Authorization/policyDefinitions",
			ResourceName:          id,
			ResourceNameMatchType: types.NameMatchTypeIDExact,
		},
	}

	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, id, mapped[0].ResourceID)
	for _, iss := range issues {
		assert.NotEqual(t, types.IssueTypeMultipleResourceIDs, iss.IssueType)
	}
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_MultipleMatches_LocationFilter(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
		{ID: "2", Name: "res1", Type: "type1", Location: "westus"},
	}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeExact,
		},
	}
	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Len(t, mapped, 1)
	assert.Equal(t, mapped[0].ResourceID, "1")
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_MultipleMatches_NoLocationFilter_Issue(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
		{ID: "2", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeExact,
		},
	}
	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Empty(t, mapped)
	assert.Len(t, issues, 1)
	for _, issue := range issues {
		assert.Equal(t, types.IssueTypeMultipleResourceIDs, issue.IssueType)
	}
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_UnusedGraphResource_Issue(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{}
	client := &MappingClient{Logger: logger}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, nil)
	assert.Empty(t, mapped)
	assert.Len(t, issues, 1)
	for _, issue := range issues {
		assert.Equal(t, types.IssueTypeUnusedResourceID, issue.IssueType)
	}
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_ResolvedIssue_Ignore(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "notfound", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeExact,
		},
	}
	issueID := getIdentityHash("addr1")
	resolvedIssues := map[string]types.Issue{
		issueID: {
			IssueID: issueID,
			Resolution: types.IssueResolution{
				ActionType: types.ActionTypeIgnore,
			},
		},
	}
	client := &MappingClient{Logger: logger, HasInputCsv: true}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, &resolvedIssues)
	assert.Len(t, mapped, 1)
	assert.Equal(t, types.ActionTypeIgnore, mapped[0].ActionType)
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_ResolvedIssue_UnusedGraphResource_Destroy(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "1", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{}
	issueID := getIdentityHash("1")
	resolvedIssues := map[string]types.Issue{
		issueID: {
			IssueID: issueID,
			Resolution: types.IssueResolution{
				ActionType: types.ActionTypeDestroy,
			},
		},
	}
	client := &MappingClient{Logger: logger, HasInputCsv: true}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, &resolvedIssues)
	assert.Len(t, mapped, 1)
	assert.Equal(t, types.ActionTypeDestroy, mapped[0].ActionType)
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_ResolvedIssue_MultipleResourceIDs_Use(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{
		{ID: "id1", Name: "res1", Type: "type1", Location: "eastus"},
		{ID: "id2", Name: "res1", Type: "type1", Location: "eastus"},
	}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "res1", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeExact,
		},
	}
	issueID := getIdentityHash("addr1")
	resolvedIssues := map[string]types.Issue{
		issueID: {
			IssueID:           issueID,
			MappedResourceIDs: []string{"id2"},
		},
	}
	client := &MappingClient{Logger: logger, HasInputCsv: true}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, &resolvedIssues)
	assert.Len(t, mapped, 1)
	assert.Equal(t, "id2", mapped[0].ResourceID)
	assert.Equal(t, types.ActionTypeUse, mapped[0].ActionType)
	assert.Empty(t, issues)
	assert.Empty(t, errs)
}

func Test_mapResourcesFromGraphToPlan_ResolvedIssue_MissingResolution_Error(t *testing.T) {
	logger := logrus.New()
	graphResources := []*types.GraphResource{}
	planResources := []*types.PlanResource{
		{
			Address: "addr1", ResourceName: "notfound", Type: "type1", Location: "eastus",
			ResourceNameMatchType: types.NameMatchTypeExact,
		},
	}
	resolvedIssues := map[string]types.Issue{}
	client := &MappingClient{Logger: logger, HasInputCsv: true}
	mapped, issues, errs := client.mapResourcesFromGraphToPlan(graphResources, planResources, &resolvedIssues)
	assert.Empty(t, mapped)
	assert.Len(t, issues, 1)
	assert.Len(t, errs, 1)
}
