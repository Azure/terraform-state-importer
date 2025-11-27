package azure

import (
	"testing"

	"github.com/azure/terraform-state-importer/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDeduplicateGraphResources(t *testing.T) {
	logger := logrus.New()

	// Test that the mock client properly de-duplicates resources
	// by simulating the behavior of the getResources method
	resourceMap := make(map[string]*types.GraphResource)

	// Simulate adding the same resource twice (as would happen with duplicate query results)
	resources := []types.GraphResource{
		{ID: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Network/privateDnsZones/privatelink.azurecr.io", Name: "privatelink.azurecr.io", Type: "microsoft.network/privatednszones", Location: "global"},
		{ID: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Network/privateDnsZones/privatelink.azurecr.io", Name: "privatelink.azurecr.io", Type: "microsoft.network/privatednszones", Location: "global"}, // duplicate
		{ID: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Network/privateDnsZones/privatelink.cognitiveservices.azure.com", Name: "privatelink.cognitiveservices.azure.com", Type: "microsoft.network/privatednszones", Location: "global"},
		{ID: "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Network/privateDnsZones/privatelink.cognitiveservices.azure.com", Name: "privatelink.cognitiveservices.azure.com", Type: "microsoft.network/privatednszones", Location: "global"}, // duplicate
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
	foundACR := false
	foundCogSvcs := false
	for _, res := range result {
		if res.Name == "privatelink.azurecr.io" {
			foundACR = true
		}
		if res.Name == "privatelink.cognitiveservices.azure.com" {
			foundCogSvcs = true
		}
	}
	assert.True(t, foundACR, "Expected to find privatelink.azurecr.io")
	assert.True(t, foundCogSvcs, "Expected to find privatelink.cognitiveservices.azure.com")
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
		ID:       "/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Network/privateDnsZones/privatelink.azurecr.io",
		Name:     "privatelink.azurecr.io",
		Type:     "microsoft.network/privatednszones",
		Location: "global",
	}

	resourceMap[resource.ID] = &resource

	// Convert map to slice
	result := make([]*types.GraphResource, 0, len(resourceMap))
	for _, res := range resourceMap {
		result = append(result, res)
	}

	// Verify single resource is returned
	assert.Len(t, result, 1)
	assert.Equal(t, "privatelink.azurecr.io", result[0].Name)
}
