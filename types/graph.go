package types

type ResourceGraphQuery struct {
	Name  string
	Query string
}

type GraphResource struct {
	ID       string
	Type     string
	Name     string
	Location string
}
