package database

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"GitHubBot/internal/model"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

type DataBaseTool struct {
	DataBase *gorm.DB
}

var DB DataBaseTool

func InitDataBase() {
	var maxTries int
	var err error
	var dsn string
	switch config.Config.Flags["env"] {
	case "local":
		dsn = config.Config.AppConfig.DataBase.DevDsn
	case "online":
		dsn = config.Config.AppConfig.DataBase.ProDsn
	default:
		log.Log.WithFields(logrus.Fields{
			"error": "环境变量错误",
		}).Panic("环境变量错误")
	}
	maxTries = 50
	for maxTries > 0 {
		maxTries--
		DB.DataBase, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("链接数据库失败，正在重试...")
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	if DB.DataBase == nil && maxTries == 0 && err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("链接数据库失败, 重试次数超过最大重试次数")
	}
	log.Log.WithFields(logrus.Fields{
		"database": "链接数据库成功",
	}).Info("链接数据库成功")
	err = DB.DataBase.AutoMigrate(&model.GbRepos{})
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("创建仓库表失败")
	}
}
