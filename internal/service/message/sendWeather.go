package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	llmService "GitHubBot/internal/service/llm"
	weatherService "GitHubBot/internal/service/weather"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

// SendWeatherMessage 每到北京时间早上七点，发送天气预报
func SendWeatherMessage() {
	location, err := time.LoadLocation("Asia/Shanghai") // 加载北京时间时区
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("加载时区失败")
		return
	}
	for {
		now := time.Now().In(location)
		// 计算下一个7点的时间
		next := now
		if now.Hour() >= 7 {
			next = next.AddDate(0, 0, 1) // 如果已经过了今天的7点，则计划在明天的7点执行
		}
		next = time.Date(next.Year(), next.Month(), next.Day(), 3, 52, 0, 0, location)
		duration := next.Sub(now)
		time.Sleep(duration) // 等待直到下一个7点
		log.Log.Info("开始发送天气预报")
		cities, err := database.Redis.GetAllCities()
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("获取城市列表失败")
		}
		for _, city := range *cities {
			var messageSend []llmService.Message
			messageSend = make([]llmService.Message, 0)
			messageSend = append(messageSend, llmService.Message{
				Role:    "user",
				Content: config.Config.AppConfig.Character.Describe,
			})
			vipStr := config.Config.AppConfig.Llm.VipQQ
			vips := strings.Split(vipStr, ",")
			messageSend = append(messageSend, llmService.Message{
				Role:    "user",
				Content: config.Config.AppConfig.Llm.VipMessage,
			})
			for _, vip := range vips {
				vipInt, err := strconv.Atoi(vip)
				if err != nil {
					log.Log.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("转换QQ号失败")
					continue
				}
				cityInfo, err := database.Redis.GetCity(city.City)
				if err != nil {
					log.Log.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("获取城市信息失败")
					continue
				}
				code := cityInfo.DiLiCode
				weatherInfo, err := weatherService.GetWeather(code, "all")
				if err != nil {
					log.Log.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("获取天气失败")
					continue
				}
				var ans string
				messageSend = append(messageSend, llmService.Message{
					Role:    "user",
					Content: weatherInfo + config.Config.AppConfig.Llm.WeatherMessage,
				})
				ansTmp, err := llmService.SendMessage(config.Config.AppConfig.Llm.Secret, messageSend)
				if err != nil {
					log.Log.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("发送消息失败")
					continue
				}
				ans += ansTmp
				err = SendMessageToQQ("private", vipInt, 0, ans)
				if err != nil {
					log.Log.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("发送消息失败")
					continue
				}
			}
		}
	}

}
