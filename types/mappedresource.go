package types

type MappedResource struct {
	Type            MappedResourceType
	ResourceAddress string
	ResourceID      string
	IssueType       IssueType
	ActionType      ActionType
}

type MappedResourceType string

const (
	MappedResourceTypeNone      MappedResourceType = "none"
	MappedResourceTypeGraph     MappedResourceType = "graph"
	MappedResourceTypeTerraform MappedResourceType = "terraform"
)
