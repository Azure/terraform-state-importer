package types

type FieldMapping struct {
	Type              string
	SubType		      string
	Mappings		  []FieldMappingEntry
}

type FieldMappingEntry struct {
	TargetFields        []FieldMappingTargetField
	SourceLookupFields  []FieldMappingSourceLookupField
}

type FieldMappingTargetField struct {
	Name string
	From string
}

type FieldMappingSourceLookupField struct {
	Name         string
	Target       string
	Replacements []FieldMappingSourceLookupFieldReplacement
}

type FieldMappingSourceLookupFieldReplacement struct {
	Regex       string
	Replacement string
}
