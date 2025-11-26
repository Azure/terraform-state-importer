package types

type PropertyMapping struct {
	Type     string
	SubType  string
	Mappings []PropertyMappingEntry
}

type PropertyMappingEntry struct {
	TargetProperties       []PropertyMappingTargetProperty
	SourceLookupProperties []PropertyMappingSourceLookupProperty
}

type PropertyMappingTargetProperty struct {
	Name string
	From string
}

type PropertyMappingSourceLookupProperty struct {
	Name         string
	Target       string
	Replacements []PropertyMappingSourceLookupPropertyReplacement
}

type PropertyMappingSourceLookupPropertyReplacement struct {
	Regex       string
	Replacement string
}
