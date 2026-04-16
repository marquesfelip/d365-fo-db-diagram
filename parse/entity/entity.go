package entity

import "encoding/xml"

type Descriptor struct {
	XMLName     xml.Name `xml:"AxModelInfo"`
	DisplayName string   `xml:"DisplayName"`
}

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

type AxTableField struct {
	Name             string `xml:"Name"`
	ExtendedDataType string `xml:"ExtendedDataType"`
}

type AxTableRelation struct {
	Name                    string `xml:"Name"`
	SourceTable             string `xml:"SourceTable"` // Tabela do arquivo que está sendo lido
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

type AxTableRelationConstraint struct {
	Name         string `xml:"Name"`
	Field        string `xml:"Field"`
	SourceEDT    string `xml:"SourceEDT"`
	RelatedField string `xml:"RelatedField"`
}

type TableFieldInfo struct {
	TableName        string
	FieldName        string
	ExtendedDataType string
}
