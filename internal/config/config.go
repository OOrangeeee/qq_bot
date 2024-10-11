package config

import (
	"GitHubBot/internal/log"
	"GitHubBot/internal/model"
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
)

type configCenter struct {
	AppConfig *model.AppConfig
	Flags     map[string]string
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
}

type SecretItem struct {
	Name    string `json:"name"`
	Version struct {
		Value string `json:"value"`
	} `json:"version"`
}

type SecretsResponse struct {
	Secrets []SecretItem `json:"secrets"`
}

var Config configCenter

func (c *configCenter) InitConfig(e *echo.Echo, args ...string) {
	c.Flags = make(map[string]string)
	c.Flags["env"] = args[0]
	c.Flags["clientID"] = args[1]
	c.Flags["clientSecret"] = args[2]
	err := c.GetAppConfig()
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("配置获取失败")
	}
	c.initMiddleware(e)
}

func (c *configCenter) initMiddleware(e *echo.Echo) {
	//recover
	e.Use(middleware.Recover())

	//CORS
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{"*"},
		MaxAge:       3600,
	}))

	// HMACMiddleware
	e.Use(HMACMiddleware)

}

func HMACMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"message": "读取请求体失败",
			})
		}
		// 重新设置请求体，以便后续处理逻辑使用
		c.Request().Body = io.NopCloser(bytes.NewReader(bodyBytes))

		// 从请求头中获取签名
		signatureHeader := c.Request().Header.Get("X-Signature")
		if len(signatureHeader) < len("sha1=") {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": "X-Signature 请求头格式错误或缺失",
			})
		}

		// 计算 HMAC SHA1
		mac := hmac.New(sha1.New, []byte(Config.AppConfig.Hmac.Key))
		mac.Write(bodyBytes)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))
		receivedMAC := signatureHeader[len("sha1="):]
		if !hmac.Equal([]byte(expectedMAC), []byte(receivedMAC)) {
			return c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"message": "HMAC 验证失败",
			})
		}
		return next(c)
	}
}

// verifyConfig 检查 AppConfig 中是否存在零值字段
func (c *configCenter) verifyConfig(appConfig *model.AppConfig) bool {
	if !checkAllFieldsSet(reflect.ValueOf(appConfig)) {
		log.Log.Error("配置文件存在空值")
		return false
	}
	return true
}

// checkAllFieldsSet 递归检查结构体中所有字段是否为零值
func checkAllFieldsSet(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	// 处理基本类型
	if v.Kind() != reflect.Struct {
		if v.Kind() == reflect.Bool {
			return true // 对于bool类型，总是返回true，即不视false为零值
		}
		return !v.IsZero()
	}
	// 递归处理每个结构体字段
	for i := 0; i < v.NumField(); i++ {
		if !checkAllFieldsSet(v.Field(i)) {
			return false
		}
	}
	return true
}

func fetchSecrets(clientID, clientSecret string) (map[string]string, error) {
	secrets := make(map[string]string)

	// Step 1: Get Access Token
	tokenURL := "https://auth.idp.hashicorp.com/oauth2/token"
	data := "client_id=" + clientID + "&client_secret=" + clientSecret + "&grant_type=client_credentials&audience=https://api.hashicorp.cloud"
	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("关闭请求体失败")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting access token, status: %s", resp.Status)
	}

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	if err != nil {
		return nil, fmt.Errorf("error decoding access token response: %v", err)
	}

	// Step 2: Use Access Token to fetch secrets
	secretsURL := "https://api.cloud.hashicorp.com/secrets/2023-06-13/organizations/8196eed4-4f13-4429-8f0e-b12f613cd493/projects/46f34615-819d-48eb-bf37-20ce5f17aad6/apps/qq-bot/open"
	req, _ = http.NewRequest("GET", secretsURL, nil)
	req.Header.Add("Authorization", "Bearer "+authResp.AccessToken)

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching secrets: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("关闭请求体失败")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error fetching secrets, status: %s, body: %s", resp.Status, string(body))
	}

	var secretsResp SecretsResponse
	err = json.NewDecoder(resp.Body).Decode(&secretsResp)
	if err != nil {
		return nil, fmt.Errorf("error decoding secrets response: %v", err)
	}

	// Step 3: Store the secrets in a map
	for _, item := range secretsResp.Secrets {
		secrets[item.Name] = item.Version.Value
	}

	return secrets, nil
}

// GetAppConfig 获取配置
func (c *configCenter) GetAppConfig() error {
	file, err := os.Open("./config/config.json")
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("打开配置文件失败")
		return errors.New("打开配置文件失败")
	}
	bytesS, err := io.ReadAll(file)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("读取配置文件失败")
		return errors.New("读取配置文件失败")
	}
	var appConfig model.AppConfig
	err = json.Unmarshal(bytesS, &appConfig)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("解析配置失败")
		return errors.New("解析配置失败")
	}
	secrets, err := fetchSecrets(c.Flags["clientID"], c.Flags["clientSecret"])
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error":        err.Error(),
			"clicentID":    c.Flags["clientID"],
			"clientSecret": c.Flags["clientSecret"],
		}).Panic("获取secrets失败")
		return errors.New("获取secrets失败")
	}
	log.Log.WithFields(logrus.Fields{
		"secrets": secrets,
	}).Info("获取secrets成功")
	appConfig.Hmac.Key = secrets["hmac_key"]
	appConfig.Github.Token = secrets["github_token"]
	appConfig.Character.Describe = secrets["character_describe"]
	appConfig.Llm.Secret = secrets["llm_secret"]
	appConfig.Llm.VipQQ = secrets["llm_vipqq"]
	appConfig.Llm.VipMessage = secrets["llm_vip_message"]
	appConfig.QQ.BotUrl = secrets["qq_bot_url"]
	appConfig.QQ.BotToken = secrets["qq_bot_token"]
	appConfig.QQ.BotQQ = secrets["qq_bot_qq"]
	appConfig.Llm.Version = secrets["llm_version"]
	appConfig.Gaode.Key = secrets["gaode_key"]
	appConfig.Llm.WeatherMessage = secrets["llm_weather_message"]
	appConfig.Llm.WeatherMessageVip = secrets["llm_weather_message_vip"]
	// 判断appConfig是否符合要求
	if !c.verifyConfig(&appConfig) {
		log.Log.WithFields(logrus.Fields{
			"error": "config error",
		}).Panic("配置文件错误")
		return errors.New("配置文件错误")
	}
	c.AppConfig = &appConfig
	log.Log.WithFields(logrus.Fields{
		"config": c.AppConfig,
	}).Info("配置获取成功")
	return nil
}
