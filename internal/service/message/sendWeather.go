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

	var lastSentDay int // 记录上次发送消息的日期

	for {
		now := time.Now().In(location)
		currentDay := now.Day() // 获取今天的日期

		// 计算下一个7点的时间
		next7AM := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, location)
		if now.After(next7AM) { // 如果当前时间已经过了今天的7点
			next7AM = next7AM.Add(24 * time.Hour) // 设置下一个7点为明天的7点
		}

		// 睡眠直到下一个7点
		sleepDuration := next7AM.Sub(now)
		log.Log.WithFields(logrus.Fields{
			"sleepDuration": sleepDuration,
		}).Info("进入睡眠等待下一个7点")
		time.Sleep(sleepDuration)

		currentDay = now.Day() // 获取今天的日期

		// 如果日期改变了，或者今天还没有发送
		if currentDay != lastSentDay {
			log.Log.Info("发送天气预报")
			cities, err := database.Redis.GetAllCities()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("获取城市列表失败")
				continue // 如果获取城市列表失败，跳过这次循环，但继续下一天
			}

			if len(*cities) == 0 {
				log.Log.Info("没有城市需要发送天气预报")
				continue
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
			// 成功发送后，记录今天的日期，避免重复发送
			lastSentDay = currentDay
		} else {
			log.Log.Info("今天的天气预报已经发送过，跳过")
		}
	}
}
