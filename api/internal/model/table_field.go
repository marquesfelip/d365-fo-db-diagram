package model

type TableField struct {
	ID        uint    `gorm:"column:id_table_field;primaryKey;autoIncrement"`
	FkAxTable *uint   `gorm:"column:fk_ax_table"`
	Name      string  `gorm:"column:name"`
	Edt       *string `gorm:"column:edt"`
	FkEdt     *int    `gorm:"column:fk_edt"`
}

func (TableField) TableName() string {
	return "table_field"
}
