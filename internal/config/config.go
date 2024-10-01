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
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
)

type configCenter struct {
	AppConfig *model.AppConfig
	Flags     map[string]string
}

var Config configCenter

func (c *configCenter) InitConfig(env string, e *echo.Echo) {
	err := c.GetAppConfig()
	c.Flags = make(map[string]string)
	c.Flags["env"] = env
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

	/*	// RequestLoggerMiddleware
		e.Use(RequestLoggerMiddleware)*/

	// HMACMiddleware
	e.Use(HMACMiddleware)

}

func RequestLoggerMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// 打印请求方法和请求路径
		log.Log.Info("Method: %s, Path: %s", c.Request().Method, c.Request().URL.Path)

		// 打印所有请求头
		for header, values := range c.Request().Header {
			for _, value := range values {
				log.Log.Info("Header: %s, Value: %s", header, value)
			}
		}

		// 读取并打印请求体
		bodyBytes, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			log.Log.Info("Error reading body: %s", err)
			return err
		}
		// 重新设置请求体，以便后续处理逻辑可以使用
		c.Request().Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))

		log.Log.Info("Body: %s", string(bodyBytes))

		// 继续处理请求
		return next(c)
	}
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

// GetAppConfig 获取配置
func (c *configCenter) GetAppConfig() error {
	file, err := os.Open("./config/config.json")
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("打开配置文件失败")
		return errors.New("打开配置文件失败")
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("读取配置文件失败")
		return errors.New("读取配置文件失败")
	}
	var appConfig model.AppConfig
	err = json.Unmarshal(bytes, &appConfig)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("解析配置失败")
		return errors.New("解析配置失败")
	}
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
