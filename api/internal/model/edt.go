package model

type Edt struct {
	ID               uint    `gorm:"column:id_edt;primaryKey;autoIncrement"`
	Name             string  `gorm:"column:name"`
	Extends          *string `gorm:"column:extends"`
	FkExtends        *uint   `gorm:"column:fk_extends"`
	ReferenceTable   string  `gorm:"column:reference_table"`
	FkReferenceTable *uint   `gorm:"column:fk_reference_table"`
	RelatedField     string  `gorm:"column:related_field"`
	FkRelatedField   *uint   `gorm:"column:fk_related_field"`
}

func (Edt) TableName() string {
	return "edt"
}
