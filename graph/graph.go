package graph

import (
	"context"
	"regexp"

	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
)

type Graph struct {
	SubscriptionIDs          []string
	IgnoreResourceIDPatterns []string
	ResourceGraphQueries	[]ResourceGraphQuery
	Logger *logrus.Logger
}

type ResourceGraphQuery struct {
	Name string
	Query string
}

type Resource struct {
	ID string
	Type string
	Name string
	Location string
}

func (graph *Graph) GetResources() ([]Resource, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		graph.Logger.Fatal(err)
	}

	resources := []Resource{}

	for _, subscriptionID := range graph.SubscriptionIDs {
		graph.Logger.Infof("Checking Subscription ID: %s\n", subscriptionID)
		resources = graph.getResources(subscriptionID, cred, resources)
	}

	return resources, nil
}

func (graph *Graph) getResources(subscriptionID string, cred *azidentity.DefaultAzureCredential, resources []Resource) []Resource {

	for _ , query := range graph.ResourceGraphQueries {
		graph.Logger.Infof("Running Resource Graph Query: %s\n", query.Name)
		graph.Logger.Tracef("Query: %s\n", query.Query)

		resourcesClient, err := armresourcegraph.NewClient(cred, nil)
		if err != nil {
			graph.Logger.Fatal(err)
		}

		ctx := context.Background()
		res, err := resourcesClient.Resources(ctx, armresourcegraph.QueryRequest{
			Query: to.Ptr(query.Query),
			Subscriptions: []*string{to.Ptr(subscriptionID)},
		}, nil)
		if err != nil {
			graph.Logger.Fatal(err)
		}

		results := res.QueryResponse.Data.([]interface{})

		for _, result := range results {
			// Check if the resource ID matches any of the ignore patterns
			resource := result.(map[string]interface{})

			shouldIgnore := false
			resourceID := resource["id"].(string)
			for _, pattern := range graph.IgnoreResourceIDPatterns {
				matched, err := regexp.MatchString(pattern, resourceID)
				if err != nil {
					graph.Logger.Debugf("Error matching pattern %s: %v\n", pattern, err)
					continue
				}
				if matched {
					shouldIgnore = true
					break
				}
			}
			if shouldIgnore {
				graph.Logger.Tracef("Ignoring Resource ID: %s\n", resourceID)
				continue
			}
			graph.Logger.Tracef("Adding Resource ID: %s\n", resourceID)
			resourceResult := Resource{
				ID:   resourceID,
				Type: resource["type"].(string),
				Name: resource["name"].(string),
				Location: resource["location"].(string),
			}
			resources = append(resources, resourceResult)
		}
	}

	return resources
}
