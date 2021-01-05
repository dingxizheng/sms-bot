package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xeonx/timeago"
)

func formatAsTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
}

// Start - starts the web server
func Start() {
	router := gin.Default()
	router.SetFuncMap(map[string]interface{}{
		"formatAsTimeAgo": formatAsTimeAgo,
	})
	router.LoadHTMLGlob("templates/*")

	// Mount controllers
	MountIndexController(router)

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.Run()
}
