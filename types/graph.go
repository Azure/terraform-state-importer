package types

type ResourceGraphQuery struct {
	Name  string
	Scope ResourceGraphQueryScope
	Query string
}

type GraphResource struct {
	ID       string
	Type     string
	Name     string
	Location string
}

type ResourceGraphQueryScope string

const (
	ResourceGraphQueryScopeManagementGroup ResourceGraphQueryScope = "ManagementGroup"
	ResourceGraphQueryScopeSubscription    ResourceGraphQueryScope = "Subscription"
)
