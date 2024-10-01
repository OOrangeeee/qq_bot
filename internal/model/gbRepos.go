package model

import "gorm.io/gorm"

type GbRepos struct {
	gorm.Model
	RepoName string `gorm:"unique;not null"`
	Url      string `gorm:"not null"`
}
