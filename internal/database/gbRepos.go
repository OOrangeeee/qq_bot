package database

import (
	"gorm.io/gorm"
)

type GbRepos struct {
	gorm.Model
	RepoName string `gorm:"unique;not null;column:repo_name"`
	Url      string `gorm:"not null;column:url"`
}

func (gb *GbRepos) Add() error {
	return DB.DataBase.Create(gb).Error
}

func (gb *GbRepos) Delete() error {
	return DB.DataBase.Unscoped().Where("id = ?", gb.ID).Delete(gb).Error
}

func (gb *GbRepos) Update() error {
	return DB.DataBase.Model(&GbRepos{}).Where("id = ?", gb.ID).Updates(map[string]interface{}{
		"repo_name": gb.RepoName,
		"url":       gb.Url,
	}).Error
}

func (gb *GbRepos) GetByStr(str string, value string) error {
	// 清空结构体
	*gb = GbRepos{}
	var repos []*GbRepos
	result := DB.DataBase.Find(&repos, str+" = ?", value)
	if result.Error != nil {
		return result.Error
	}
	if len(repos) <= 0 {
		// 返回没有找到错误
		return gorm.ErrRecordNotFound
	}
	*gb = *repos[0]
	return nil
}

func (gb *GbRepos) GetAll() (*[]Record, error) {
	var repos []*GbRepos
	result := DB.DataBase.Find(&repos)
	if result.Error != nil {
		return nil, result.Error
	}
	var records []Record
	for _, repo := range repos {
		records = append(records, repo)
	}
	return &records, nil
}
