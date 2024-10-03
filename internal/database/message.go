package database

import (
	"gorm.io/gorm"
	"time"
)

type Message struct {
	gorm.Model
	token  string    `gorm:"unique;not null"`
	fromId uint      `gorm:"not null"`
	toId   uint      `gorm:"not null"`
	text   string    `gorm:"not null"`
	time   time.Time `gorm:"not null"`
}
