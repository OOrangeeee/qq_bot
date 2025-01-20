package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	llmService "GitHubBot/internal/service/llm"
	weatherService "GitHubBot/internal/service/weather"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SendWeatherMessage 每到北京时间早上七点，发送天气预报
func SendWeatherMessage() {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("加载时区失败")
		return
	}

	var lastSentDay int
	ticker := time.NewTicker(time.Minute) // 使用较短的检查间隔
	defer ticker.Stop()

	for {
		now := time.Now().In(location)
		if now.Hour() == 7 && now.Minute() == 0 && now.Day() != lastSentDay {
			log.Log.Info("开始发送天气预报")
			cities, err := database.Redis.GetAllCities()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("获取城市列表失败")
				continue
			}
			if len(*cities) == 0 {
				log.Log.Info("没有城市需要发送天气预报")
				continue
			}
			for _, city := range *cities {
				var messageSend []llmService.Message
				messageSend = make([]llmService.Message, 0)
				messageSend = append(messageSend, llmService.Message{
					Role:    "system",
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
						}).Error("转换VIP QQ号失败")
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
						}).Error("获取天气信息失败")
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
						}).Error("调用LLM发送消息失败")
						continue
					}
					ans += ansTmp
					err = SendMessageToQQ("private", vipInt, 0, ans)
					if err != nil {
						log.Log.WithFields(logrus.Fields{
							"error": err.Error(),
						}).Error("发送QQ消息失败")
						continue
					}
				}
			}
			lastSentDay = now.Day()
			log.Log.Info("天气预报发送完成")
		}
		<-ticker.C
	}
}
