package main

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/database"
	"GitHubBot/internal/log"
	"GitHubBot/internal/route"
	"flag"
	"github.com/labstack/echo/v4"
)

func main() {
	// 环境变量解析
	env := flag.String("env", "online", "online or local, default is online")
	flag.Parse()
	// 启动echo
	e := echo.New()
	log.InitLog()
	config.Config.InitConfig(*env, e)
	database.InitDataBase()
	database.InitRedis()
	defer database.Redis.Exit()
	route.Route(e)
	e.Logger.Fatal(e.Start(":2077"))
}
