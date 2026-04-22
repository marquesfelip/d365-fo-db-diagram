package model

type AxTable struct {
	ID                 uint    `gorm:"column:id_ax_table;primaryKey;autoIncrement"`
	Name               string  `gorm:"column:name"`
	Model              string  `gorm:"column:model"`
	Layer              *string `gorm:"column:layer"`
	Extends            *string `gorm:"column:extends"`
	FkExtends          *uint   `gorm:"column:fk_extends"`
	SaveDataPerCompany bool    `gorm:"column:save_data_per_company"`
	TableGroup         string  `gorm:"column:table_group"`
	TableType          string  `gorm:"column:table_type"`
	PrimaryIndex       string  `gorm:"column:primary_index"`
	ReplacementKey     string  `gorm:"column:replacement_key"`
}

func (AxTable) TableName() string {
	return "ax_table"
}
