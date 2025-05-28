package types

type PlanResource struct {
	Address               string
	Type                  string
	SubType			      string
	Name                  string
	Location              string
	ResourceName          string
	ResourceNameMatchType NameMatchType
	MappedResources       []*GraphResource
	Properties            map[string]any
	PropertiesCalculated  map[string]any
}

type NameMatchType string

const (
	NameMatchTypeExact      NameMatchType = "Exact"
	NameMatchTypeIDContains NameMatchType = "IDContains"
	NameMatchTypeIDEndsWith NameMatchType = "IDEndsWith"
)
