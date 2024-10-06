package service

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	llmService "GitHubBot/internal/service/llm"
	service "GitHubBot/internal/service/message"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ResponseBodyCode struct {
	Status   string `json:"status"`
	Info     string `json:"info"`
	InfoCode string `json:"infocode"`
	Count    string `json:"count"`
	Geocodes []struct {
		Adcode   string `json:"adcode"`
		Citycode string `json:"citycode"`
	}
}

type ResponseBodyWeather struct {
	Status   string `json:"status"`
	Count    string `json:"count"`
	Info     string `json:"info"`
	InfoCode string `json:"infocode"`
	Lives    []struct {
		Weather       string `json:"weather"`
		City          string `json:"city"`
		Temperature   string `json:"temperature"`
		Winddirection string `json:"winddirection"`
		Windpower     string `json:"windpower"`
		Humidity      string `json:"humidity"`
		Reporttime    string `json:"reporttime"`
	} `json:"lives"`
	Forecasts []struct {
		Reporttime string `json:"reporttime"`
		City       string `json:"city"`
		Casts      []struct {
			Date         string `json:"date"`
			Week         string `json:"week"`
			Dayweather   string `json:"dayweather"`
			Nightweather string `json:"nightweather"`
			Daytemp      string `json:"daytemp"`
			Nighttemp    string `json:"nighttemp"`
			Daywind      string `json:"daywind"`
			Nightwind    string `json:"nightwind"`
			Daypower     string `json:"daypower"`
			Nightpower   string `json:"nightpower"`
		}
	} `json:"forecasts"`
}

func GetDiLiCode(address, city string) (string, error) {
	// 查询参数
	params := url.Values{}
	params.Set("key", config.Config.AppConfig.Gaode.Key)
	params.Set("address", address)
	params.Set("city", city)
	req, err := http.NewRequest("GET", config.Config.AppConfig.Gaode.DiLiCodeUrl+"?"+params.Encode(), nil)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("创建请求失败")
		return "", err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("发送请求失败")
		return "", err
	}
	defer func(Body io.ReadCloser) {
		if resp != nil {
			err := resp.Body.Close()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("关闭请求失败")
			}
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("读取响应失败")
		return "", err
	}

	var responseBody ResponseBodyCode
	if err := json.Unmarshal(body, &responseBody); err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("解析响应失败")
		return "", err
	}
	if responseBody.Status != "1" {
		errorMsg := "高德地图API请求失败，错误码：" + responseBody.InfoCode + "，错误信息：" + responseBody.Info
		log.Log.WithFields(logrus.Fields{
			"error": errorMsg,
		}).Error("请求失败")
		return "", errors.New(errorMsg)
	}
	return responseBody.Geocodes[0].Adcode, nil
}

func GetWeather(diLiCode string, typeCode string) (string, error) {
	params := url.Values{}
	params.Set("key", config.Config.AppConfig.Gaode.Key)
	params.Set("city", diLiCode)
	params.Set("extensions", typeCode)
	client := &http.Client{}
	req, err := http.NewRequest("GET", config.Config.AppConfig.Gaode.WeatherUrl+"?"+params.Encode(), nil)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("创建请求失败")
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("发送请求失败")
		return "", err
	}
	defer func(Body io.ReadCloser) {
		if resp != nil {
			err := resp.Body.Close()
			if err != nil {
				log.Log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("关闭请求失败")
			}
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("读取响应失败")
		return "", err
	}
	var responseBody ResponseBodyWeather
	if err := json.Unmarshal(body, &responseBody); err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("解析响应失败")
		return "", err
	}
	if responseBody.Status != "1" {
		errorMsg := "高德地图API请求失败，错误码：" + responseBody.InfoCode + "，错误信息：" + responseBody.Info
		log.Log.WithFields(logrus.Fields{
			"error": errorMsg,
		}).Error("请求失败")
		return "", errors.New(errorMsg)
	}
	if typeCode == "base" {
		// 组合成天气预报的字符串，尽可能包含所有信息
		var weather string
		weather += "天气预报\n"
		weather += "城市：" + responseBody.Lives[0].City + "\n"
		weather += "更新时间：" + responseBody.Lives[0].Reporttime + "\n"
		weather += "天气：" + responseBody.Lives[0].Weather + "\n" + "温度：" + responseBody.Lives[0].Temperature + "℃\n" + "风向：" + responseBody.Lives[0].Winddirection + "\n" + "风力：" + responseBody.Lives[0].Windpower + "\n" + "湿度：" + responseBody.Lives[0].Humidity + "\n"
		return weather, nil
	} else {
		// 组合成天气预报的字符串，尽可能包含所有信息，包括未来几天的天气预报，casts[0]表示今天，casts[1]表示明天，以此类推
		var weather string
		weather += "天气预报\n"
		weather += "城市：" + responseBody.Forecasts[0].City + "\n"
		weather += "今日日期：" + responseBody.Forecasts[0].Casts[0].Date + "\n"
		weather += "更新时间：" + responseBody.Forecasts[0].Reporttime + "\n"
		for _, cast := range responseBody.Forecasts[0].Casts {
			weather += "日期:" + cast.Date + "\n" + "星期:" + cast.Week + "\n" + "白天天气:" + cast.Dayweather + "\n" + "夜晚天气:" + cast.Nightweather + "\n" + "白天温度:" + cast.Daytemp + "℃\n" + "夜晚温度:" + cast.Nighttemp + "℃\n" + "白天风向:" + cast.Daywind + "\n" + "夜晚风向:" + cast.Nightwind + "\n" + "白天风力:" + cast.Daypower + "\n" + "夜晚风力:" + cast.Nightpower + "\n"
		}
		return weather, nil
	}
}

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
		next = time.Date(next.Year(), next.Month(), next.Day(), 3, 38, 0, 0, location)
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
				weatherInfo, err := GetWeather(code, "all")
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
				err = service.SendMessageToQQ("private", vipInt, 0, ans)
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
