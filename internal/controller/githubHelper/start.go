package controller

import "github.com/labstack/echo/v4"

func Start(c echo.Context) error {
	return c.JSON(200, map[string]interface{}{
		"reply": "Hello, here is Orange's Github Helper!",
	})
}
