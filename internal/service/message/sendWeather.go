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
	sentToday := false // 今天是否已发送的标记
	for {
		now := time.Now().In(location)
		// 计算下一个7点的时间
		next7AM := time.Date(now.Year(), now.Month(), now.Day(), 4, 6, 0, 0, location)
		if now.After(next7AM) { // 如果当前时间已经过了今天的7点
			next7AM = next7AM.Add(24 * time.Hour) // 设置下一个7点为明天的7点
		}
		time.Sleep(next7AM.Sub(now)) // 睡眠直到下一个7点
		if !sentToday {
			log.Log.Info("发送天气预报")
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
			sentToday = true
		}
		if now.Hour() == 5 { // 如果当前时间是7点
			sentToday = false
		}
	}
}
