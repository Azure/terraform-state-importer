package azure

import (
	"context"
	"regexp"

	"github.com/azure/terraform-state-importer/types"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
)

type IResourceGraphClient interface {
	GetResources() ([]*types.GraphResource, error)
}

type ResourceGraphClient struct {
	Cloud                    cloud.Configuration
	ManagementGroupIDs       []*string
	SubscriptionIDs          []*string
	IgnoreResourceIDPatterns []string
	ResourceGraphQueries     []types.ResourceGraphQuery
	Logger                   *logrus.Logger
}

func NewResourceGraphClient(cloudConfiguration string, managementGroupIDs []string, subscriptionIDs []string, ignoreResourceIDPatterns []string, resourceGraphQueries []types.ResourceGraphQuery, logger *logrus.Logger) *ResourceGraphClient {
	// Convert string slices to pointer slices
	managementGroupIDsPtr := make([]*string, len(managementGroupIDs))
	for i, id := range managementGroupIDs {
		managementGroupIDsPtr[i] = &id
	}
	subscriptionIDsPtr := make([]*string, len(subscriptionIDs))
	for i, id := range subscriptionIDs {
		if id == "" || id == "00000000-0000-0000-0000-000000000000" {
			logger.Fatalf("Subscription ID is not valid, please update your config file with valid subscription IDs: %s", id)
		}
		subscriptionIDsPtr[i] = &id
	}

	var cloudConfigurationFinal cloud.Configuration
	switch cloudConfiguration {
	case "AzurePublic":
		cloudConfigurationFinal = cloud.AzurePublic
	case "AzureUSGovernment":
		cloudConfigurationFinal = cloud.AzureGovernment
	case "AzureGovernment":
		cloudConfigurationFinal = cloud.AzureGovernment
	case "AzureChina":
		cloudConfigurationFinal = cloud.AzureChina
	default:
		logger.Fatalf("Unsupported cloud specified: %s", cloudConfiguration)
	}

	return &ResourceGraphClient{
		Cloud:                    cloudConfigurationFinal,
		ManagementGroupIDs:       managementGroupIDsPtr,
		SubscriptionIDs:          subscriptionIDsPtr,
		IgnoreResourceIDPatterns: ignoreResourceIDPatterns,
		ResourceGraphQueries:     resourceGraphQueries,
		Logger:                   logger,
	}
}

func (graph *ResourceGraphClient) GetResources() ([]*types.GraphResource, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		graph.Logger.Fatal(err)
	}

	resourceMap := make(map[string]*types.GraphResource)

	if len(graph.SubscriptionIDs) > 0 {
		emptyGuid := "00000000-0000-0000-0000-000000000000"
		guidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
		for _, subscriptionID := range graph.SubscriptionIDs {
			if subscriptionID == &emptyGuid || !guidRegex.MatchString(*subscriptionID) {
				graph.Logger.Fatalf("Invalid Subscription ID: %s", *subscriptionID)
				continue
			}
		}
		graph.Logger.Info("Running graph queries for Subscriptions")
		graph.getResourcesBySubscriptionID(cred, resourceMap)
	}

	if len(graph.ManagementGroupIDs) > 0 {
		graph.Logger.Info("Running graph queries for Management Groups")
		graph.getResourcesByManagementGroupID(cred, resourceMap)
	}

	if len(graph.SubscriptionIDs) == 0 && len(graph.ManagementGroupIDs) == 0 {
		graph.Logger.Fatal("Subscription IDs or Management Group IDs must be provided")
	}

	resources := make([]*types.GraphResource, 0, len(resourceMap))
	for _, resource := range resourceMap {
		resources = append(resources, resource)
	}

	return resources, nil
}

func (graph *ResourceGraphClient) getResourcesByManagementGroupID(cred *azidentity.DefaultAzureCredential, resourceMap map[string]*types.GraphResource) {
	queryRequest := armresourcegraph.QueryRequest{
		Options: &armresourcegraph.QueryRequestOptions{
			AuthorizationScopeFilter: to.Ptr(armresourcegraph.AuthorizationScopeFilterAtScopeAndBelow),
		},
		ManagementGroups: graph.ManagementGroupIDs,
	}

	graph.getResources(types.ResourceGraphQueryScopeManagementGroup, queryRequest, cred, resourceMap)
}

func (graph *ResourceGraphClient) getResourcesBySubscriptionID(cred *azidentity.DefaultAzureCredential, resourceMap map[string]*types.GraphResource) {
	queryRequest := armresourcegraph.QueryRequest{
		Options: &armresourcegraph.QueryRequestOptions{
			AuthorizationScopeFilter: to.Ptr(armresourcegraph.AuthorizationScopeFilterAtScopeAndBelow),
		},
		Subscriptions: graph.SubscriptionIDs,
	}

	graph.getResources(types.ResourceGraphQueryScopeSubscription, queryRequest, cred, resourceMap)
}

func (graph *ResourceGraphClient) getResources(scope types.ResourceGraphQueryScope, queryRequest armresourcegraph.QueryRequest, cred *azidentity.DefaultAzureCredential, resourceMap map[string]*types.GraphResource) {
	for _, query := range graph.ResourceGraphQueries {
		if query.Scope != scope {
			graph.Logger.Debugf("Skipping query %s for scope %s", query.Name, scope)
			continue
		}

		graph.Logger.Infof("Running Resource Graph Query: %s", query.Name)
		graph.Logger.Tracef("Query: %s", query.Query)

		opts := azcore.ClientOptions{Cloud: cloud.AzurePublic}
		resourcesClient, err := armresourcegraph.NewClient(cred, &arm.ClientOptions{
			ClientOptions: opts,
		})
		if err != nil {
			graph.Logger.Fatal(err)
		}

		ctx := context.Background()

		queryRequest.Query = to.Ptr(query.Query)

		res, err := resourcesClient.Resources(ctx, queryRequest, nil)
		if err != nil {
			graph.Logger.Fatal(err)
		}

		results := res.QueryResponse.Data.([]any)

		for _, result := range results {
			// Check if the resource ID matches any of the ignore patterns
			resource := result.(map[string]any)

			shouldIgnore := false
			resourceID := resource["id"].(string)
			for _, pattern := range graph.IgnoreResourceIDPatterns {
				matched, err := regexp.MatchString(pattern, resourceID)
				if err != nil {
					graph.Logger.Debugf("Error matching pattern %s: %v", pattern, err)
					continue
				}
				if matched {
					shouldIgnore = true
					break
				}
			}
			if shouldIgnore {
				graph.Logger.Tracef("Ignoring Resource ID: %s", resourceID)
				continue
			}
			// Skip if the resource ID is already in the map (de-duplication)
			if _, exists := resourceMap[resourceID]; exists {
				graph.Logger.Tracef("Skipping duplicate Resource ID: %s", resourceID)
				continue
			}
			graph.Logger.Tracef("Adding Resource ID: %s", resourceID)
			resourceResult := types.GraphResource{
				ID:       resourceID,
				Type:     resource["type"].(string),
				Name:     resource["name"].(string),
				Location: resource["location"].(string),
			}
			resourceMap[resourceID] = &resourceResult
		}
	}
}
