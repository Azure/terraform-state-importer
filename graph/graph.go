package graph

import (
	"context"
	"regexp"
	"github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type Graph struct {
	SubscriptionIDs          []string
	IgnoreResourceIDPatterns []string
	Logger *logrus.Logger
}

type Resource struct {
	ID string
	Type string
	Name string
}

func (graph *Graph) GetResources() ([]Resource, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		graph.Logger.Fatal(err)
	}

	resources := []Resource{}

	for _, subscriptionID := range graph.SubscriptionIDs {
		graph.Logger.Infof("Checking Subscription ID: %s\n", subscriptionID)

		resources = graph.getResourceGroups(subscriptionID, cred, resources)
		resources = graph.getResources(subscriptionID, cred, resources)
	}

	return resources, nil
}

func (graph *Graph) getResourceGroups(subscriptionID string, cred *azidentity.DefaultAzureCredential, resources []Resource) []Resource {
	resourceGroupsClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
	    graph.Logger.Fatal(err)
	}

	pager := resourceGroupsClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			graph.Logger.Fatal(err)
		}
		for _, resourceGroup := range page.Value {
			shouldIgnore := false
			for _, pattern := range graph.IgnoreResourceIDPatterns {
				matched, err := regexp.MatchString(pattern, *resourceGroup.ID)
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
				graph.Logger.Tracef("Ignoring Resource ID: %s\n", *resourceGroup.ID)
				continue
			}
			graph.Logger.Tracef("Adding Resource ID: %s\n", *resourceGroup.ID)
			resourceResult := Resource{
				ID:   *resourceGroup.ID,
				Type: *resourceGroup.Type,
				Name: *resourceGroup.Name,
			}
			resources = append(resources, resourceResult)
		}
	}
	return resources
}

func (graph *Graph) getResources(subscriptionID string, cred *azidentity.DefaultAzureCredential, resources []Resource) []Resource {
	resourcesClient, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		graph.Logger.Fatal(err)
	}

	pager := resourcesClient.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			graph.Logger.Fatal(err)
		}

		for _, resource := range page.Value {
			// Check if the resource ID matches any of the ignore patterns
			shouldIgnore := false
			for _, pattern := range graph.IgnoreResourceIDPatterns {
				matched, err := regexp.MatchString(pattern, *resource.ID)
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
				graph.Logger.Tracef("Ignoring Resource ID: %s\n", *resource.ID)
				continue
			}
			graph.Logger.Tracef("Adding Resource ID: %s\n", *resource.ID)
			resourceResult := Resource{
				ID:   *resource.ID,
				Type: *resource.Type,
				Name: *resource.Name,
			}
			resources = append(resources, resourceResult)
		}
	}
	return resources
}
