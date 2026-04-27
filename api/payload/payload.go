package payload

// AxTable represents the structure of an AxTable XML file.
type AxTable struct {
	Name               string `json:"Name"`
	Model              string `json:"-"`
	Layer              string `json:"-"`
	Extends            string `json:"Extends"`
	SaveDataPerCompany bool   `json:"SaveDataPerCompany"`
	TableGroup         string `json:"TableGroup"`
	TableType          string `json:"TableType"`
	PrimaryIndex       string `json:"PrimaryIndex"`
	ReplacementKey     string `json:"ReplacementKey"`
	Fields             struct {
		AxTableField []AxTableField `json:"AxTableField"`
	} `json:"Fields"`
	Relations struct {
		AxTableRelation []AxTableRelation `json:"AxTableRelation"`
	} `json:"Relations"`
}

// AxTableField represents a field in an AxTable.
type AxTableField struct {
	Name             string `json:"Name"`
	ExtendedDataType string `json:"ExtendedDataType"`
}

// AxTableRelation represents a relation in an AxTable.
type AxTableRelation struct {
	Name                    string `json:"Name"`
	SourceTable             string `json:"SourceTable"` // Table of the file being read
	RelatedTable            string `json:"RelatedTable"`
	EDTRelation             bool   `json:"EDTRelation"`
	OnDelete                string `json:"OnDelete"`
	Cardinality             string `json:"Cardinality"`
	RelatedTableCardinality string `json:"RelatedTableCardinality"`
	RelationshipType        string `json:"RelationshipType"`
	Constraints             struct {
		AxTableRelationConstraint []AxTableRelationConstraint `json:"AxTableRelationConstraint"`
	} `json:"Constraints"`
}

// AxTableRelationConstraint represents a constraint in an AxTable relation.
type AxTableRelationConstraint struct {
	Name         string `json:"Name"`
	Field        string `json:"Field"`
	SourceEDT    string `json:"SourceEDT"`
	RelatedField string `json:"RelatedField"`
}
