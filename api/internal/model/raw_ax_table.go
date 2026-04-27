package model

import (
	"time"
)

type RawAxTable struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"index"`
	Model     string `gorm:"index"`
	Layer     *string
	Payload   []byte `gorm:"type:jsonb"`
	Processed bool
	CreatedAt time.Time
}
