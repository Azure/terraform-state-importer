package issues

type Issue struct {
	IssueID           string
	IssueType         IssueType
	ResourceAddress   string
	ResourceName      string
	ResourceType      string
	ResourceLocation  string
	MappedResourceIDs []string
}

type IssueType string

const (
	IssueTypeNone                IssueType = "None"
	IssueTypeNoResourceID        IssueType = "NoResourceID"
	IssueTypeMultipleResourceIDs IssueType = "MultipleResourceIDs"
	IssueTypeUnusedResourceID    IssueType = "UnusedResourceID"
)
