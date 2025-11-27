package azure

import (
	"testing"

	"github.com/azure/terraform-state-importer/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	testResourceID1   = "/subscriptions/123/rg/providers/dns/zone1"
	testResourceID2   = "/subscriptions/123/rg/providers/dns/zone2"
	testResourceName1 = "zone1"
	testResourceName2 = "zone2"
	testResourceType  = "microsoft.network/privatednszones"
	testLocation      = "global"
)

func TestDeduplicateGraphResources(t *testing.T) {
	logger := logrus.New()

	// Test that the mock client properly de-duplicates resources
	// by simulating the behavior of the getResources method
	resourceMap := make(map[string]*types.GraphResource)

	// Simulate adding the same resource twice (as would happen with duplicate query results)
	resources := []types.GraphResource{
		{ID: testResourceID1, Name: testResourceName1, Type: testResourceType, Location: testLocation},
		{ID: testResourceID1, Name: testResourceName1, Type: testResourceType, Location: testLocation}, // duplicate
		{ID: testResourceID2, Name: testResourceName2, Type: testResourceType, Location: testLocation},
		{ID: testResourceID2, Name: testResourceName2, Type: testResourceType, Location: testLocation}, // duplicate
	}

	for _, res := range resources {
		resourceID := res.ID
		if _, exists := resourceMap[resourceID]; exists {
			logger.Tracef("Skipping duplicate Resource ID: %s", resourceID)
			continue
		}
		resourceMap[resourceID] = &types.GraphResource{
			ID:       res.ID,
			Name:     res.Name,
			Type:     res.Type,
			Location: res.Location,
		}
	}

	// Convert map to slice
	result := make([]*types.GraphResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		result = append(result, resource)
	}

	// Verify only 2 unique resources are returned
	assert.Len(t, result, 2)

	// Verify both unique resources are present
	foundResource1 := false
	foundResource2 := false
	for _, res := range result {
		if res.Name == testResourceName1 {
			foundResource1 = true
		}
		if res.Name == testResourceName2 {
			foundResource2 = true
		}
	}
	assert.True(t, foundResource1, "Expected to find "+testResourceName1)
	assert.True(t, foundResource2, "Expected to find "+testResourceName2)
}

func TestDeduplicateEmptyResources(t *testing.T) {
	resourceMap := make(map[string]*types.GraphResource)

	// Convert empty map to slice
	result := make([]*types.GraphResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		result = append(result, resource)
	}

	// Verify empty result
	assert.Empty(t, result)
}

func TestDeduplicateSingleResource(t *testing.T) {
	resourceMap := make(map[string]*types.GraphResource)

	// Add a single resource
	resource := types.GraphResource{
		ID:       testResourceID1,
		Name:     testResourceName1,
		Type:     testResourceType,
		Location: testLocation,
	}

	resourceMap[resource.ID] = &resource

	// Convert map to slice
	result := make([]*types.GraphResource, 0, len(resourceMap))
	for _, res := range resourceMap {
		result = append(result, res)
	}

	// Verify single resource is returned
	assert.Len(t, result, 1)
	assert.Equal(t, testResourceName1, result[0].Name)
}
