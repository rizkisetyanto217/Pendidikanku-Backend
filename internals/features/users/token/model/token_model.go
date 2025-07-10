package model

import (
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	Token     string    `gorm:"type:text;not null" json:"token"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
