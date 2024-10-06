package weather

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
)

type ResponseBody struct {
	Status   string `json:"status"`
	Info     string `json:"info"`
	InfoCode string `json:"infocode"`
	Count    string `json:"count"`
	Geocodes []struct {
		Adcode   string `json:"adcode"`
		Citycode string `json:"citycode"`
	}
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

	var responseBody ResponseBody
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
