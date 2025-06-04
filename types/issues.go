package types

type Issue struct {
	IssueID           string
	IssueType         IssueType
	ResourceAddress   string
	ResourceName      string
	ResourceType      string
	ResourceSubType   string
	ResourceLocation  string
	MappedResourceIDs []string
	Resolution        IssueResolution
}

type IssueType string

const (
	IssueTypeNone                IssueType = "None"
	IssueTypeNoResourceID        IssueType = "NoResourceID"
	IssueTypeMultipleResourceIDs IssueType = "MultipleResourceIDs"
	IssueTypeUnusedResourceID    IssueType = "UnusedResourceID"
)

func (issueType IssueType) IsValidIssueType() bool {
	switch issueType {
	case IssueTypeNone,
		IssueTypeNoResourceID,
		IssueTypeMultipleResourceIDs,
		IssueTypeUnusedResourceID:
		return true
	default:
		return false
	}
}

type IssueResolution struct {
	ActionType ActionType
	ActionID   string
}

type ActionType string

const (
	ActionTypeNone    ActionType = ""
	ActionTypeUse     ActionType = "Use"
	ActionTypeIgnore  ActionType = "Ignore"
	ActionTypeReplace ActionType = "Replace"
	ActionTypeDestroy ActionType = "Destroy"
)

func (actionType ActionType) IsValidActionType() bool {
	switch actionType {
	case ActionTypeNone,
		ActionTypeUse,
		ActionTypeIgnore,
		ActionTypeReplace,
		ActionTypeDestroy:
		return true
	default:
		return false
	}
}
