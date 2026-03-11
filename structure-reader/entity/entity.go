package entity

import "encoding/xml"

type Descriptor struct {
	XMLName     xml.Name `xml:"AxModelInfo"`
	DisplayName string   `xml:"DisplayName"`
}

type AxTable struct {
	XMLName xml.Name `xml:"AxTable"`
	Name    string   `xml:"Name"`
	Fields  struct {
		AxTableField []AxTableField `xml:"AxTableField"`
	} `xml:"Fields"`
}

type AxTableField struct {
	XMLName          xml.Name `xml:"AxTableField"`
	Name             string   `xml:"Name"`
	ExtendedDataType string   `xml:"ExtendedDataType"`
}

type TableFieldInfo struct {
	TableName        string
	FieldName        string
	ExtendedDataType string
}
