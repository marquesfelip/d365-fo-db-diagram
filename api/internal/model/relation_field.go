package model

type RelationField struct {
	ID              uint    `gorm:"column:id_relation_field;primaryKey;autoIncrement"`
	FkTableRelation *uint   `gorm:"column:fk_table_relation"`
	Name            string  `gorm:"column:name"`
	SourceField     string  `gorm:"column:source_field"`
	FkSourceField   *uint   `gorm:"column:fk_source_field"`
	RelatedField    string  `gorm:"column:related_field"`
	FkRelatedField  *uint   `gorm:"column:fk_related_field"`
	SourceEdt       *string `gorm:"column:source_edt"`
	FkSourceEdt     *uint   `gorm:"column:fk_source_edt"`
}

func (RelationField) TableName() string {
	return "relation_field"
}
