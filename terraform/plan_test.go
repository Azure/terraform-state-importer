package terraform

import (
	"testing"

	"github.com/azure/terraform-state-importer/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_NameFormatAppliedBySubType(t *testing.T) {
	logger := logrus.New()
	pc := &PlanClient{
		PropertyMappings: []types.PropertyMapping{},
		NameFormats: []types.NameFormat{
			{
				Type:                "Microsoft.Authorization/policyAssignments",
				SubType:             "",
				NameFormat:          "%s/providers/Microsoft.Authorization/policyAssignments/%s",
				NameMatchType:       types.NameMatchTypeIDEndsWith,
				NameFormatArguments: []string{"parent_id", "name"},
			},
		},
		Logger: logger,
	}

	res := &types.PlanResource{
		Address: "addr1",
		Type:    "azapi_resource",
		SubType: "Microsoft.Authorization/policyAssignments",
		Properties: map[string]any{
			"parent_id": "/providers/Microsoft.Management/managementGroups/alz-aks-public",
			"name":      "Allow-Vnet-Peering",
		},
	}

	out := pc.mapPropertiesAndNames([]*types.PlanResource{res})
	assert.Equal(t, 1, len(out))
	want := "/providers/Microsoft.Management/managementGroups/alz-aks-public/providers/Microsoft.Authorization/policyAssignments/Allow-Vnet-Peering"
	assert.Equal(t, want, out[0].ResourceName)
	assert.Equal(t, types.NameMatchTypeIDEndsWith, out[0].ResourceNameMatchType)
}

func Test_NameFormat_PreferExactSubTypeOverType(t *testing.T) {
	logger := logrus.New()
	pc := &PlanClient{
		PropertyMappings: []types.PropertyMapping{},
		NameFormats: []types.NameFormat{
			// Broader Type match
			{
				Type:                "azapi_resource",
				SubType:             "Microsoft.Authorization/policyAssignments",
				NameFormat:          "BROADER:%s/%s",
				NameMatchType:       types.NameMatchTypeIDEndsWith,
				NameFormatArguments: []string{"parent_id", "name"},
			},
			// Exact SubType match
			{
				Type:                "Microsoft.Authorization/policyAssignments",
				SubType:             "",
				NameFormat:          "EXACT:%s/%s",
				NameMatchType:       types.NameMatchTypeIDEndsWith,
				NameFormatArguments: []string{"parent_id", "name"},
			},
		},
		Logger: logger,
	}

	res := &types.PlanResource{
		Address: "addr1",
		Type:    "azapi_resource",
		SubType: "Microsoft.Authorization/policyAssignments",
		Properties: map[string]any{
			"parent_id": "/p",
			"name":      "n",
		},
	}

	out := pc.mapPropertiesAndNames([]*types.PlanResource{res})
	assert.Equal(t, 1, len(out))
	// Expect exact SubType match chosen first
	assert.Equal(t, "EXACT:/p/n", out[0].ResourceName)
}

func Test_NameFormat_IDExact_ForPolicyDefinitions(t *testing.T) {
	logger := logrus.New()
	pc := &PlanClient{
		PropertyMappings: []types.PropertyMapping{},
		NameFormats: []types.NameFormat{
			{
				Type:                "Microsoft.Authorization/policyDefinitions",
				SubType:             "",
				NameFormat:          "%s/providers/Microsoft.Authorization/policyDefinitions/%s",
				NameMatchType:       types.NameMatchTypeIDExact,
				NameFormatArguments: []string{"parent_id", "name"},
			},
		},
		Logger: logger,
	}

	res := &types.PlanResource{
		Address: "addr1",
		Type:    "azapi_resource",
		SubType: "Microsoft.Authorization/policyDefinitions",
		Properties: map[string]any{
			"parent_id": "/providers/Microsoft.Management/managementGroups/alz",
			"name":      "Enforce-Tag",
		},
	}

	out := pc.mapPropertiesAndNames([]*types.PlanResource{res})
	assert.Equal(t, 1, len(out))
	want := "/providers/Microsoft.Management/managementGroups/alz/providers/Microsoft.Authorization/policyDefinitions/Enforce-Tag"
	assert.Equal(t, want, out[0].ResourceName)
	assert.Equal(t, types.NameMatchTypeIDExact, out[0].ResourceNameMatchType)
}
