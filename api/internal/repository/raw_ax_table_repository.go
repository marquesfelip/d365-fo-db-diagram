package repository

import (
	"github.com/marquesfelip/d365-fo-db-diagram/internal/model"
	"gorm.io/gorm"
)

type RawAxTableRepository struct {
	db *gorm.DB
}

func NewRawAxTableRepository(db *gorm.DB) *RawAxTableRepository {
	return &RawAxTableRepository{
		db: db,
	}
}

func (r *RawAxTableRepository) CreateBatch(items []model.RawAxTable) error {
	return r.db.Create(&items).Error
}
