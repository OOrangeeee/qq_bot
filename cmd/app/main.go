package main

import (
	"GitHubBot/internal/config"
	"GitHubBot/internal/log"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	log.InitLog()
	config.Config.InitConfig(e)
	e.Logger.Fatal(e.Start(":2077"))
}
