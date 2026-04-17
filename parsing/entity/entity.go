package entity

import "encoding/xml"

// Descriptor represents the metadata of a D365 model (Descriptor folder XML file).
// ModelFolder is manually filled after reading, with the XML filename without extension,
// corresponding to the model folder inside the package.
// ModelName is the readable alias for DisplayName to make code easier to read.
type Descriptor struct {
	XMLName     xml.Name `xml:"AxModelInfo"`
	DisplayName string   `xml:"DisplayName"` // readable model name (e.g., "Application Suite")
	ModelFolder string   `xml:"-"`           // XML filename without extension (e.g., "Foundation")
}

// AxTable represents the structure of an AxTable XML file.
type AxTable struct {
	XMLName            xml.Name `xml:"AxTable"`
	Name               string   `xml:"Name"`
	Model              string   `xml:"-"`
	Layer              string   `xml:"-"`
	Extends            string   `xml:"Extends"`
	SaveDataPerCompany string   `xml:"SaveDataPerCompany"`
	TableGroup         string   `xml:"TableGroup"`
	TableType          string   `xml:"TableType"`
	PrimaryIndex       string   `xml:"PrimaryIndex"`
	ReplacementKey     string   `xml:"ReplacementKey"`
	Fields             struct {
		AxTableField []AxTableField `xml:"AxTableField"`
	} `xml:"Fields"`
	Relations struct {
		AxTableRelation []AxTableRelation `xml:"AxTableRelation"`
	} `xml:"Relations"`
}

// AxTableField represents a field in an AxTable.
type AxTableField struct {
	Name             string `xml:"Name"`
	ExtendedDataType string `xml:"ExtendedDataType"`
}

// AxTableRelation represents a relation in an AxTable.
type AxTableRelation struct {
	Name                    string `xml:"Name"`
	SourceTable             string `xml:"SourceTable"` // Table of the file being read
	RelatedTable            string `xml:"RelatedTable"`
	EDTRelation             bool   `xml:"EDTRelation"` // TODO: default false
	OnDelete                string `xml:"OnDelete"`    // default null
	Cardinality             string `xml:"Cardinality"`
	RelatedTableCardinality string `xml:"RelatedTableCardinality"`
	RelationshipType        string `xml:"RelationshipType"`
	Constraints             struct {
		AxTableRelationConstraint []AxTableRelationConstraint `xml:"AxTableRelationConstraint"`
	} `xml:"Constraints"`
}

// AxTableRelationConstraint represents a constraint in an AxTable relation.
type AxTableRelationConstraint struct {
	Name         string `xml:"Name"`
	Field        string `xml:"Field"`
	SourceEDT    string `xml:"SourceEDT"`
	RelatedField string `xml:"RelatedField"`
}

// TableFieldInfo holds information about a table field.
type TableFieldInfo struct {
	TableName        string
	FieldName        string
	ExtendedDataType string
}
