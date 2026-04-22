package model

type TableRelation struct {
	ID                      uint    `gorm:"column:id_table_relation;primaryKey;autoIncrement"`
	Name                    string  `gorm:"column:name"`
	SourceTable             string  `gorm:"column:source_table"`
	FkSourceTable           *uint   `gorm:"column:fk_source_table"`
	RelatedTable            string  `gorm:"column:related_table"`
	FkRelatedTable          *uint   `gorm:"column:fk_related_table"`
	EdtRelation             *string `gorm:"column:edt_relation"`
	OnDelete                *string `gorm:"column:on_delete"`
	Cardinality             *string `gorm:"column:cardinality"`
	RelatedTableCardinality string  `gorm:"column:related_table_cardinality"`
	RelationshipType        string  `gorm:"column:relationship_type"`
}

func (TableRelation) TableName() string {
	return "table_relation"
}
