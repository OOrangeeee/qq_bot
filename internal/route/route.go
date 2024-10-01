package route

import (
	controller "GitHubBot/internal/controller/event"
	"github.com/labstack/echo/v4"
)

func Route(e *echo.Echo) {
	getRoute(e)
	postRoute(e)
	deleteRoute(e)
	putRoute(e)
}

func getRoute(e *echo.Echo) {
}

func postRoute(e *echo.Echo) {
	e.POST("/event", controller.SolveEvent)
}

func putRoute(e *echo.Echo) {
}

func deleteRoute(e *echo.Echo) {
}
