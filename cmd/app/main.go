package main

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	"GitHubBot/internal/route"
	service "GitHubBot/internal/service/message"
	"flag"
	"github.com/labstack/echo/v4"
	"os"
)

func main() {
	// 环境变量解析
	env := flag.String("env", "online", "online or local, default is online")
	flag.Parse()
	clientID := os.Getenv("HCP_CLIENT_ID")
	clientSecret := os.Getenv("HCP_CLIENT_SECRET")
	// 启动echo
	e := echo.New()
	log.InitLog()
	config.Config.InitConfig(e, *env, clientID, clientSecret)
	database.InitDataBase()
	database.InitRedis()
	defer database.Redis.Exit()
	route.Route(e)
	go service.SendWeatherMessage()
	e.Logger.Fatal(e.Start(":2077"))
}
