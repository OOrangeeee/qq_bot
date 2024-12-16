package controller

import (
	"GitHubBot/internal/log"
	service "GitHubBot/internal/service/github"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func GetRepoInfo(c echo.Context) error {
	repoName := c.QueryParam("repoName")
	owner := c.QueryParam("owner")
	ans, err := service.GetJsonInfoOfRepo(owner, repoName)
	if err != nil {
		log.Log.WithFields(logrus.Fields{
			"error":   err.Error(),
			"message": "GetRepoInfo failed",
		}).Error("GetRepoInfo failed")
		return c.JSON(400, map[string]interface{}{
			"error":   err.Error(),
			"message": "GetRepoInfo failed",
		})

	}
	return c.JSON(200, map[string]interface{}{
		"RepoInfo": ans,
	})
}
