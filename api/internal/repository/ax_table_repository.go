package repository

import (
	"github.com/marquesfelip/d365-fo-db-diagram/internal/model"
	"gorm.io/gorm"
)

type AxTableRepository struct {
	db *gorm.DB
}

func NewAxTableRepository(db *gorm.DB) *AxTableRepository {
	return &AxTableRepository{
		db: db,
	}
}

func (r *AxTableRepository) CreateBatch(items []model.AxTable) error {
	return r.db.Create(&items).Error
}
