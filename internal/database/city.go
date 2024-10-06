package database

import "gorm.io/gorm"

type City struct {
	gorm.Model
	City     string `gorm:"column:city;not null;unique"`
	DiLiCode string `gorm:"column:di_li_code;not null;unique"`
}

func (c *City) Add() error {
	return DB.DataBase.Create(c).Error
}

func (c *City) Delete() error {
	return DB.DataBase.Unscoped().Where("id = ?", c.ID).Delete(c).Error
}

func (c *City) Update() error {
	return DB.DataBase.Model(&City{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
		"city":       c.City,
		"di_li_code": c.DiLiCode,
	}).Error
}

func (c *City) GetByStr(str string, value string) error {
	*c = City{}
	var weathers []*City
	result := DB.DataBase.Find(&weathers, str+" = ?", value)
	if result.Error != nil {
		return result.Error
	}
	if len(weathers) <= 0 {
		return gorm.ErrRecordNotFound
	}
	*c = *weathers[0]
	return nil
}

func (c *City) GetAll() (*[]Record, error) {
	var weathers []*City
	result := DB.DataBase.Find(&weathers)
	if result.Error != nil {
		return nil, result.Error
	}
	var records []Record
	for _, weather := range weathers {
		records = append(records, weather)
	}
	return &records, nil
}
