package azure

import (
	"context"
	"regexp"

	"github.com/azure/terraform-state-importer/types"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
)

type IResourceGraphClient interface {
	GetResources() ([]types.GraphResource, error)
}

type ResourceGraphClient struct {
	SubscriptionIDs          []string
	IgnoreResourceIDPatterns []string
	ResourceGraphQueries     []types.ResourceGraphQuery
	Logger                   *logrus.Logger
}

func NewResourceGraphClient(subscriptionIDs []string, ignoreResourceIDPatterns []string, resourceGraphQueries []types.ResourceGraphQuery, logger *logrus.Logger) *ResourceGraphClient {
	return &ResourceGraphClient{
		SubscriptionIDs:          subscriptionIDs,
		IgnoreResourceIDPatterns: ignoreResourceIDPatterns,
		ResourceGraphQueries:     resourceGraphQueries,
		Logger:                   logger,
	}
}

func (graph *ResourceGraphClient) GetResources() ([]types.GraphResource, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		graph.Logger.Fatal(err)
	}

	resources := []types.GraphResource{}

	for _, subscriptionID := range graph.SubscriptionIDs {
		graph.Logger.Infof("Checking Subscription ID: %s", subscriptionID)
		resources = graph.getResources(subscriptionID, cred, resources)
	}

	return resources, nil
}

func (graph *ResourceGraphClient) getResources(subscriptionID string, cred *azidentity.DefaultAzureCredential, resources []types.GraphResource) []types.GraphResource {
	for _, query := range graph.ResourceGraphQueries {
		graph.Logger.Infof("Running Resource Graph Query: %s", query.Name)
		graph.Logger.Tracef("Query: %s", query.Query)

		resourcesClient, err := armresourcegraph.NewClient(cred, nil)
		if err != nil {
			graph.Logger.Fatal(err)
		}

		ctx := context.Background()
		res, err := resourcesClient.Resources(ctx, armresourcegraph.QueryRequest{
			Query:         to.Ptr(query.Query),
			Subscriptions: []*string{to.Ptr(subscriptionID)},
		}, nil)
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
			graph.Logger.Tracef("Adding Resource ID: %s", resourceID)
			resourceResult := types.GraphResource{
				ID:       resourceID,
				Type:     resource["type"].(string),
				Name:     resource["name"].(string),
				Location: resource["location"].(string),
			}
			resources = append(resources, resourceResult)
		}
	}

	return resources
}
