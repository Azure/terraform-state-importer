package graph

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

type Graph struct {
	SubscriptionIDs          []string
	IgnoreResourceIDPatterns []string
}

func (graph *Graph) GetResources() ([]armresources.GenericResourceExpanded, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		panic(err)
	}

	resources := []armresources.GenericResourceExpanded{}

	for _, subscriptionID := range graph.SubscriptionIDs {
		fmt.Printf("Checking Subscription ID: %s\n", subscriptionID)
		client, err := armresources.NewClient(subscriptionID, cred, nil)
		if err != nil {
			panic(err)
		}
		pager := client.NewListPager(nil)

		for pager.More() {
			page, err := pager.NextPage(context.Background())
			if err != nil {
				panic(err)
			}

			for _, resource := range page.Value {
				// Check if the resource ID matches any of the ignore patterns
				shouldIgnore := false
				for _, pattern := range graph.IgnoreResourceIDPatterns {
					matched, err := regexp.MatchString(pattern, *resource.ID)
					if err != nil {
						fmt.Printf("Error matching pattern %s: %v\n", pattern, err)
						continue
					}
					if matched {
						shouldIgnore = true
						break
					}
				}
				if shouldIgnore {
					fmt.Printf("Ignoring Resource ID: %s\n", *resource.ID)
					continue
				}
				fmt.Printf("Adding Resource ID: %s\n", *resource.ID)
				resources = append(resources, *resource)
			}
		}
	}

	return resources, nil
}
