package entity

import "encoding/xml"

type Descriptor struct {
	XMLName     xml.Name `xml:"AxModelInfo"`
	DisplayName string   `xml:"DisplayName"`
}

type AxTable struct {
	XMLName            xml.Name `xml:"AxTable"`
	Name               string   `xml:"Name"`
	SaveDataPerCompany string   `xml:"SaveDataPerCompany"`
	TableGroup         string   `xml:"TableGroup"`
	TableType          string   `xml:"TableType"`
	PrimaryIndex       string   `xml:"PrimaryIndex"`
	ReplacementKey     string   `xml:"ReplacementKey"`
	Fields             struct {
		AxTableField []AxTableField `xml:"AxTableField"`
	} `xml:"Fields"`
}

type AxTableField struct {
	XMLName          xml.Name `xml:"AxTableField"`
	Name             string   `xml:"Name"`
	ExtendedDataType string   `xml:"ExtendedDataType"`
}

type AxTableRelation struct {
	XMLName                 xml.Name `xml:"AxTableRelation"`
	Name                    string   `xml:"Name"`
	SourceTable             string   `xml:"SourceTable"`
	RelatedTable            string   `xml:"RelatedTable"`
	EDTRelation             string   `xml:"EDTRelation"`
	OnDelete                string   `xml:"OnDelete"`
	Cardinality             string   `xml:"Cardinality"`
	RelatedTableCardinality string   `xml:"RelatedTableCardinality"`
	RelationshipType        string   `xml:"RelationshipType"`
}

type TableFieldInfo struct {
	TableName        string
	FieldName        string
	ExtendedDataType string
}
