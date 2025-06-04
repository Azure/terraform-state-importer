package types

type MappedResource struct {
	Type               MappedResourceType
	ResourceAddress    string
	ResourceID         string
	ResourceAPIVersion string
	ResourceType       string
	IssueType          IssueType
	ActionType         ActionType
}

type MappedResourceType string

const (
	MappedResourceTypeNone      MappedResourceType = "none"
	MappedResourceTypeGraph     MappedResourceType = "graph"
	MappedResourceTypeTerraform MappedResourceType = "terraform"
)
