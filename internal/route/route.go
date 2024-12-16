package route

import (
	controllerEvent "GitHubBot/internal/controller/event"
	controllerGithubHelper "GitHubBot/internal/controller/githubHelper"

	"github.com/labstack/echo/v4"
)

func Route(e *echo.Echo) {
	getRoute(e)
	postRoute(e)
	deleteRoute(e)
	putRoute(e)
}

func getRoute(e *echo.Echo) {
	e.GET("/start", controllerGithubHelper.Start)
}

func postRoute(e *echo.Echo) {
	e.POST("/event", controllerEvent.SolveEvent)
}

func putRoute(e *echo.Echo) {
}

func deleteRoute(e *echo.Echo) {
}
