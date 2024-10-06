package database

import (
	"gorm.io/gorm"
	"time"
)

type Message struct {
	gorm.Model
	FromId int       `gorm:"not null"`
	ToId   int       `gorm:"not null"`
	Text   string    `gorm:"not null"`
	Time   time.Time `gorm:"not null"`
}

func (m *Message) Add() error {
	return DB.DataBase.Create(m).Error
}

func (m *Message) Delete() error {
	return DB.DataBase.Unscoped().Where("id = ?", m.ID).Delete(m).Error
}

func (m *Message) Update() error {
	return DB.DataBase.Model(&Message{}).Where("id = ?", m.ID).Updates(map[string]interface{}{
		"fromId": m.FromId,
		"toId":   m.ToId,
		"text":   m.Text,
		"time":   m.Time,
	}).Error
}

func (m *Message) GetByStr(str string, value string) error {
	*m = Message{}
	var messages []*Message
	result := DB.DataBase.Find(&messages, str+" = ?", value)
	if result.Error != nil {
		return result.Error
	}
	if len(messages) <= 0 {
		return gorm.ErrRecordNotFound
	}
	*m = *messages[0]
	return nil
}

func (m *Message) GetAll() (*[]Record, error) {
	var messages []*Message
	result := DB.DataBase.Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}
	var records []Record
	for _, message := range messages {
		records = append(records, message)
	}
	return &records, nil
}

func (m *Message) GetByFromId(fromId int) (*[]*Message, error) {
	var messages []*Message
	result := DB.DataBase.Find(&messages, "from_id = ?", fromId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &messages, nil
}
